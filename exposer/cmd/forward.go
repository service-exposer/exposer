package cmd

import (
	"fmt"
	"net"
	"os"

	"github.com/service-exposer/exposer"
	"github.com/service-exposer/exposer/listener/utils"
	"github.com/service-exposer/exposer/protocal/forward"
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
		server_url   = ""
		key          = ""
	)
	forwardCmd.Flags().IntVarP(&local_port, "local-port", "l", local_port, "local port")
	forwardCmd.Flags().StringVarP(&forward_addr, "forward-addr", "f", forward_addr, "forward address")
	forwardCmd.Flags().StringVarP(&server_url, "server-url", "s", server_url, "server url")
	forwardCmd.Flags().StringVarP(&key, "key", "k", key, "auth key")
	forwardCmd.Run = func(cmd *cobra.Command, args []string) {
		ln, err := net.Listen("tcp", fmt.Sprintf("0.0.0.0:%d", local_port))
		if err != nil {
			fmt.Fprintln(os.Stderr, "listen", fmt.Sprintf("0.0.0.0:%d", local_port), "failure", err)
			os.Exit(-2)
		}
		defer ln.Close()

		conn, err := utils.DialWebsocket(server_url)
		if err != nil {
			fmt.Fprintln(os.Stderr, "dial server", server_url, "failure", err)
		}
		defer conn.Close()

		proto := exposer.NewProtocal(conn)
		proto.On = forward.ClientSide(forward.Forward{
			Network: "tcp",
			Address: forward_addr,
		}, ln)

		proto.Request(forward.CMD_AUTH, &forward.Auth{
			Key: key,
		})
	}
}
