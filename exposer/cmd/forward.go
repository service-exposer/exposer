package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	forward_addr = ""
	local_port   = 0
	server_url   = ""
)

// forwardCmd represents the forward command
var forwardCmd = &cobra.Command{
	Use:   "forward",
	Short: "A client for forwarding network traffic via remote server",
	Long: `The subcmd forward and forward-server are a pair for forwarding network traffic
via websocket protocal. So it can through 80 or 443 port via HTTP(s) protocal.`,
	Run: func(cmd *cobra.Command, args []string) {
		// TODO: Work your own magic here
		fmt.Println("forward called")
	},
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

	forwardCmd.Flags().IntVarP(&local_port, "local-port", "l", 7000, "local port")
	forwardCmd.Flags().StringVarP(&forward_addr, "forward-addr", "f", "", "forward address")
	forwardCmd.Flags().StringVarP(&server_url, "server-url", "s", "", "server url")
}
