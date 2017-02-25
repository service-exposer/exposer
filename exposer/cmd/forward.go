package cmd

import (
	"fmt"
	"log"
	"net"
	"os"

	"github.com/service-exposer/exposer"
	"github.com/service-exposer/exposer/listener/utils"
	"github.com/service-exposer/exposer/protocal/auth"
	"github.com/service-exposer/exposer/protocal/forward"
	"github.com/service-exposer/exposer/protocal/keepalive"
	"github.com/service-exposer/exposer/protocal/route"
	"github.com/spf13/cobra"
)

// forwardCmd represents the forward command
var forwardCmd = &cobra.Command{
	Use:   "forward",
	Short: "A client for forwarding network traffic via remote server",
	Long: `The subcmd forward and forward-server are a pair for forwarding network traffic
		via websocket protocal. So it can through 80 or 443 port via HTTP(s) protocal.`,
}

func init() {
	RootCmd.AddCommand(forwardCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// forwardCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// forwardCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
	var (
		forward_addr = ""
		local_port   = 0
	)
	forwardCmd.Flags().IntVarP(&local_port, "local-port", "l", local_port, "local port")
	forwardCmd.Flags().StringVarP(&forward_addr, "forward-addr", "f", forward_addr, "forward address")
	forwardCmd.Run = func(cmd *cobra.Command, args []string) {
		log.Print("listen ", local_port)
		ln, err := net.Listen("tcp", fmt.Sprintf(":%d", local_port))
		if err != nil {
			fmt.Fprintln(os.Stderr, "listen", fmt.Sprintf(":%d", local_port), "failure", err)
			os.Exit(-2)
		}
		defer ln.Close()

		log.Print("connect to server ", server_websocket_url())
		conn, err := utils.DialWebsocket(server_websocket_url())
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
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

			nextRoutes <- auth.NextRoute{
				Req: route.RouteReq{
					Type: route.Forward,
				},
				HandleFunc: forward.ClientSide(ln),
				Cmd:        forward.CMD_FORWARD,
				Details: &forward.Forward{
					Network: "tcp",
					Address: forward_addr,
				},
			}
			log.Print("setup forward route")
		}()

		proto.Request(auth.CMD_AUTH, &auth.AuthReq{
			Key: key,
		})
	}
}
