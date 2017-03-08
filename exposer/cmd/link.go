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
	"github.com/service-exposer/exposer/protocal/keepalive"
	"github.com/service-exposer/exposer/protocal/link"
	"github.com/service-exposer/exposer/protocal/route"
	"github.com/spf13/cobra"
)

// linkCmd represents the link command
var linkCmd = &cobra.Command{Use: "link",
	Short: "link service via daemon",
}

func init() {
	RootCmd.AddCommand(linkCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// linkCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// linkCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
	var (
		service_name = ""
		listen_addr  = "localhost:" // [host]:port
	)
	linkCmd.Flags().StringVarP(&service_name, "name", "n", service_name, "service name")
	linkCmd.Flags().StringVarP(&listen_addr, "listen", "l", listen_addr, "listen address. format: [host]:port")

	linkCmd.Run = func(cmd *cobra.Command, args []string) {
		exit := func(code int, outs ...interface{}) {
			fmt.Fprintln(os.Stderr, outs...)
			os.Exit(code)
		}

		if service_name == "" {
			exit(1, "not set service name")
		}

		if listen_addr == "" {
			exit(2, "not set listen address")
		}

		ln, err := net.Listen("tcp", listen_addr)
		if err != nil {
			fmt.Fprintln(os.Stderr, "listen", listen_addr, "failure", err)
			os.Exit(-2)
		}
		defer ln.Close()
		log.Print("listen ", ln.Addr())

		conn, err := utils.DialWebsocket(server_websocket_url())
		if err != nil {
			fmt.Fprintln(os.Stderr, "connect to server", server_websocket_url(), "failure", err)
			os.Exit(-3)
		}
		defer conn.Close()
		log.Print("connect to server ", server_websocket_url())

		nextRoutes := make(chan auth.NextRoute)
		proto := exposer.NewProtocal(conn)
		proto.On = auth.ClientSide(nextRoutes)

		go func() {
			nextRoutes <- auth.NextRoute{
				Req: route.RouteReq{
					Type: route.KeepAlive,
				},
				HandleFunc: keepalive.ClientSide(0, 0),
				Cmd:        keepalive.CMD_PING,
			}
			log.Print("setup keepalive route")

			nextRoutes <- auth.NextRoute{
				Req: route.RouteReq{
					Type: route.Link,
				},
				HandleFunc: link.ClientSide(ln),
				Cmd:        link.CMD_LINK,
				Details: &link.LinkReq{
					Name: service_name,
				},
			}
			log.Print("setup link route")
		}()

		go proto.Request(auth.CMD_AUTH, &auth.AuthReq{
			Key: key,
		})
		exit(0, proto.Wait())
	}
}
