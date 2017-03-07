// Copyright © 2017 NAME HERE <EMAIL ADDRESS>
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/service-exposer/exposer"
	"github.com/service-exposer/exposer/listener/utils"
	"github.com/service-exposer/exposer/protocal/auth"
	"github.com/service-exposer/exposer/service"
	"github.com/spf13/cobra"
	"github.com/urfave/negroni"
)

// daemonCmd represents the daemon command
var daemonCmd = &cobra.Command{
	Use:   "daemon",
	Short: "The daemon is server-side of exposer",
}

func init() {
	RootCmd.AddCommand(daemonCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// daemonCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// daemonCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
	var (
		addr = "0.0.0.0:9000"
	)
	daemonCmd.Flags().StringVarP(&addr, "addr", "a", addr, "listen address")

	daemonCmd.Run = func(cmd *cobra.Command, args []string) {
		ln, err := net.Listen("tcp", addr)
		if err != nil {
			fmt.Fprintln(os.Stderr, "listen", addr, "failure", err)
			os.Exit(-1)
		}
		defer ln.Close()
		log.Print("listen ", addr)

		wsln, wsconnHandler, err := utils.WebsocketHandlerListener(ln.Addr())
		if err != nil {
			fmt.Fprintln(os.Stderr, "listen ws", addr, "failure", err)
			os.Exit(-2)
		}
		defer wsln.Close()

		serviceRouter := service.NewRouter()

		r := mux.NewRouter()

		r.Path("/api/services").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			services := serviceRouter.All()

			result := make(map[string]*json.RawMessage)
			for _, s := range services {
				s.Attribute().View(func(attr service.Attribute) error {
					data, err := json.Marshal(attr)
					if err != nil {
						return err
					}

					rawmsg := json.RawMessage(data)
					result[s.Name()] = &rawmsg
					return nil
				})
			}

			json.NewEncoder(w).Encode(&result)

		}).Methods("GET")

		r.PathPrefix("/service/{name}").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			vars := mux.Vars(r)
			var (
				name = vars["name"]
			)

			s := serviceRouter.Get(name)
			if s == nil {
				http.Error(w, "service is not exist", 404)
				return
			}

			var attr service.Attribute
			s.Attribute().View(func(a service.Attribute) error {
				attr = a
				return nil
			})

			if !attr.HTTP.Is {
				http.Error(w, "service is not a HTTP service", 404)
				return
			}

			if r.URL.Path == "/service/"+name {
				http.Redirect(w, r, "/service/"+name+"/", 302)
				return
			}

			hj, ok := w.(http.Hijacker)
			if !ok {
				http.Error(w, "webserver doesn't support hijacking", 500)
				return
			}

			client, clientbufrw, err := hj.Hijack()
			if err != nil {
				http.Error(w, err.Error(), 500)
				return
			}

			server, err := s.Open()
			if err != nil {
				http.Error(w, err.Error(), 500)
				return
			}

			go func(r *http.Request) {
				var err error
				for err == nil {
					subPath := r.URL.Path[len("/service/"+name):]
					if subPath == "" {
						client.Close()
						server.Close()
						return
					}
					if subPath[0] != '/' {
						subPath = "/" + subPath
					}
					url, _ := url.Parse(subPath)

					r.URL = url
					if attr.HTTP.Host != "" {
						r.Host = attr.HTTP.Host
					}
					r.Header.Set("X-Origin-IP", client.RemoteAddr().String())

					r.Write(server)

					if r.Header.Get("Upgrade") != "" {
						break
					}
					r, err = http.ReadRequest(clientbufrw.Reader)
				}

				io.Copy(server, clientbufrw)
				client.Close()
			}(r)

			io.Copy(clientbufrw, server)
			server.Close()
		})

		n := negroni.New()

		// ws
		n.UseFunc(func(w http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
			if strings.HasPrefix(r.URL.Path, "/service/") {
				next(w, r)
				return
			}

			connection := r.Header.Get("Connection")
			upgrade := r.Header.Get("Upgrade")
			if connection == "Upgrade" && upgrade == "websocket" {
				wsconnHandler.ServeHTTP(w, r)
				return
			}

			next(w, r)
		})

		// auth
		n.UseFunc(func(w http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
			if !strings.HasPrefix(r.URL.Path, "/api/") {
				next(w, r)
				return
			}

			auth := r.Header.Get("Authorization")
			if auth != key {
				w.WriteHeader(401)
				fmt.Fprintln(w, "Please set Header Authorization as Key")
				return
			}

			next(w, r)
		})

		n.UseHandler(r)

		go func() {
			server := &http.Server{
				ReadTimeout:  30 * time.Second,
				WriteTimeout: 30 * time.Second,
				Handler:      n,
			}

			err := server.Serve(ln)
			if err != nil {
				fmt.Fprintln(os.Stderr, "HTTP server shutdown. occur error:", err)
			}
		}()
		exposer.Serve(wsln, func(conn net.Conn) exposer.ProtocalHandler {
			proto := exposer.NewProtocal(conn)
			proto.On = auth.ServerSide(serviceRouter, func(k string) bool {
				return k == key
			})
			return proto
		})
	}
}
