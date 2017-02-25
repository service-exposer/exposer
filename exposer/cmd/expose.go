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
	"os"

	"github.com/service-exposer/exposer"
	"github.com/service-exposer/exposer/listener/utils"
	"github.com/service-exposer/exposer/protocal/auth"
	"github.com/service-exposer/exposer/protocal/expose"
	"github.com/service-exposer/exposer/protocal/keepalive"
	"github.com/service-exposer/exposer/protocal/route"
	"github.com/service-exposer/exposer/service"
	"github.com/spf13/cobra"
)

// exposeCmd represents the expose command
var exposeCmd = &cobra.Command{
	Use:   "expose",
	Short: "expose  service via daemon",
}

func init() {
	RootCmd.AddCommand(exposeCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// exposeCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// exposeCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
	var (
		service_name = ""
		service_addr = "" // [host]:port
		is_http      = false
		http_host    = ""
	)
	exposeCmd.Flags().StringVarP(&service_name, "name", "n", service_name, "service name")
	exposeCmd.Flags().StringVarP(&service_addr, "addr", "a", service_addr, "service address format: [host]:port")
	exposeCmd.Flags().BoolVar(&is_http, "http", is_http, "expose service as HTTP")
	exposeCmd.Flags().StringVar(&http_host, "http.host", "", "set HTTP host")
	exposeCmd.Run = func(cmd *cobra.Command, args []string) {
		if service_name == "" {
			exit(1, "not set service name")
		}

		if service_addr == "" {
			exit(2, "not set service address")
		}

		log.Print("connect to server ", server_url)
		conn, err := utils.DialWebsocket(server_url)
		if err != nil {
			fmt.Fprintln(os.Stderr, "dial server", server_url, "failure", err)
			os.Exit(-3)
		}
		defer conn.Close()

		nextRoutes := make(chan auth.NextRoute)
		proto := exposer.NewProtocal(conn)
		proto.On = auth.ClientSide(nextRoutes)

		go func() {
			nextRoutes <- auth.NextRoute{
				Req: route.RouteReq{
					Type: route.KeepAlive,
				},
				HandleFunc: keepalive.ClientSide(0),
				Cmd:        keepalive.CMD_PING,
			}
			log.Print("setup keepalive route")

			handleFn := expose.ClientSide(func() (net.Conn, error) {
				return net.Dial("tcp", service_addr)
			})
			nextRoutes <- auth.NextRoute{
				Req: route.RouteReq{
					Type: route.Expose,
				},
				HandleFunc: func(proto *exposer.Protocal, cmd string, details []byte) error {
					log.Print("link:", cmd, string(details))
					return handleFn(proto, cmd, details)
				},
				Cmd: expose.CMD_EXPOSE,
				Details: &expose.ExposeReq{
					Name: service_name,
					Attr: func() (attr service.Attribute) {
						attr.HTTP.Is = is_http
						attr.HTTP.Host = http_host
						return
					}(),
				},
			}
			log.Print("setup expose route")
		}()

		proto.Request(auth.CMD_AUTH, &auth.AuthReq{
			Key: key,
		})

	}
}

func exit(code int, outs ...interface{}) {
	fmt.Fprintln(os.Stderr, outs...)
	os.Exit(code)
}
