package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

// RootCmd represents the base command when called without any subcommands
var RootCmd = &cobra.Command{
	Use:   "exposer",
	Short: "A way to expose tcp service via websocket",
	Long: `exposer is a agent to expose and link tcp service via websocket.

It is designed to work at somewhere cannot connect tcp server directly,
like firewall limitation.`,
}

const (
	ENV_SERVER_URL = "EXPOSER_SERVER"
	ENV_KEY        = "EXPOSER_KEY"
)

var (
	server_url = ""
	key        = ""
)

func init() {
	RootCmd.PersistentFlags().StringVarP(&server_url, "server", "s", os.Getenv(ENV_SERVER_URL), "server url <http(s)://host:port> ,you can set env EXPOSER_SERVER")
	RootCmd.PersistentFlags().StringVarP(&key, "key", "k", os.Getenv(ENV_KEY), "auth key,you can set env EXPOSER_KEY")
}

func server_http_url() string {
	return server_url
}

func server_websocket_url() string {
	if strings.HasPrefix(server_url, "http") {
		return strings.Replace(server_url, "http", "ws", 1)
	}

	return server_url
}

// Execute adds all child commands to the root command sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := RootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(-1)
	}
}

func exit(code int, outs ...interface{}) {
	fmt.Fprintln(os.Stderr, outs...)
	os.Exit(code)
}
