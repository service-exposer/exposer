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
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
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
		key  = ""
	)
	daemonCmd.Flags().StringVarP(&addr, "addr", "a", addr, "listen address")
	daemonCmd.Flags().StringVarP(&key, "key", "k", key, "auth key")

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
		n := negroni.New()

		// ws
		n.UseFunc(func(w http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
			connection := r.Header.Get("Connection")
			upgrade := r.Header.Get("Upgrade")
			if connection == "Upgrade" && upgrade == "websocket" {
				wsconnHandler.ServeHTTP(w, r)
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
