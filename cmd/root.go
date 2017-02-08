package cmd

import (
	"fmt"
	"os"

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

// Execute adds all child commands to the root command sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := RootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(-1)
	}
}
