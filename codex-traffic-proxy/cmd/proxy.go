package cmd

import (
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"codex-traffic-proxy/internal/config"
	"codex-traffic-proxy/internal/logger"
	"codex-traffic-proxy/internal/proxy"

	"github.com/spf13/cobra"
)

var StartCmd = &cobra.Command{
	Use:   "start",
	Short: "Start the Codex traffic proxy server",
	Long: `Start the HTTP proxy server that will intercept and analyze traffic
from the Codex CLI. The proxy will listen on the specified address and port,
and log all requests and responses for analysis.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Load configuration
		cfg, err := config.LoadConfig()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		// Initialize logger
		log := logger.NewLogger(cfg)

		log.Info("Starting Codex Traffic Proxy",
			"listen_addr", cfg.Proxy.ListenAddr,
			"port", cfg.Proxy.Port,
			"verbose", cfg.Proxy.Verbose,
		)

		// Create and start proxy
		proxyServer := proxy.NewProxy(cfg, log)

		// Set up signal handling for graceful shutdown
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

		// Start proxy in a goroutine
		go func() {
			if err := proxyServer.Start(); err != nil && err != http.ErrServerClosed {
				log.Error("Proxy server failed", "error", err)
				os.Exit(1)
			}
		}()

		log.Info("Proxy server started successfully", "address", fmt.Sprintf("%s:%d", cfg.Proxy.ListenAddr, cfg.Proxy.Port))
		log.Info("Configure your Codex CLI to use this proxy by setting HTTP_PROXY and HTTPS_PROXY environment variables")
		log.Info("Example: export HTTP_PROXY=http://127.0.0.1:8080 && export HTTPS_PROXY=http://127.0.0.1:8080")

		// Wait for shutdown signal
		<-sigChan
		log.Info("Received shutdown signal, stopping proxy server...")

		// Gracefully shutdown the proxy
		if err := proxyServer.Shutdown(); err != nil {
			log.Error("Error during proxy shutdown", "error", err)
			return err
		}

		log.Info("Proxy server stopped")
		return nil
	},
}