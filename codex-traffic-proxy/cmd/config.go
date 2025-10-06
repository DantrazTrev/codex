package cmd

import (
	"fmt"

	"codex-traffic-proxy/internal/config"

	"github.com/spf13/cobra"
)

var ConfigCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage proxy configuration",
	Long:  `Generate or display configuration for the Codex traffic proxy.`,
}

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize default configuration file",
	Long:  `Create a default configuration file in the user's home directory.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := config.SaveDefaultConfig(); err != nil {
			return fmt.Errorf("failed to save default config: %w", err)
		}

		fmt.Println("Default configuration file created at ~/.codex-traffic-proxy/config.yaml")
		fmt.Println("You can now edit this file to customize your proxy settings.")
		return nil
	},
}

func init() {
	ConfigCmd.AddCommand(initCmd)
}