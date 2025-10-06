package main

import (
	"fmt"
	"os"

	"codex-traffic-proxy/cmd"

	"github.com/spf13/cobra"
)

var cfgFile string

func main() {
	var rootCmd = &cobra.Command{
		Use:   "codex-traffic-proxy",
		Short: "A proxy server for monitoring Codex CLI traffic",
		Long: `Codex Traffic Proxy is a transparent HTTP proxy that monitors and analyzes
traffic from the Codex CLI to various AI service endpoints. It can intercept,
log, and analyze requests to help understand API usage patterns.`,
	}

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.codex-traffic-proxy.yaml)")

	// Add subcommands
	rootCmd.AddCommand(StartCmd)
	rootCmd.AddCommand(ConfigCmd)

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}