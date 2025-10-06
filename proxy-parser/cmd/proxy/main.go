package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"codex-proxy-parser/internal/config"
	"codex-proxy-parser/internal/proxy"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var (
	cfgFile string
	verbose bool
)

var rootCmd = &cobra.Command{
	Use:   "codex-proxy",
	Short: "Codex CLI Traffic Proxy and Parser",
	Long: `A Go-based HTTP proxy that intercepts and parses traffic from the Codex CLI,
specifically designed to monitor GenAI and vibe coding tool requests.`,
	Run: runProxy,
}

func init() {
	cobra.OnInitialize(initConfig)
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is ./config.yaml)")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose output")
}

func initConfig() {
	if cfgFile != "" {
		config.SetConfigFile(cfgFile)
	}
	config.SetConfigName("config")
	config.AddConfigPath(".")
	config.AddConfigPath("./configs")
	config.AddConfigPath("/etc/codex-proxy")
	
	if err := config.ReadInConfig(); err != nil {
		if _, ok := err.(config.ConfigFileNotFoundError); !ok {
			logrus.Fatalf("Error reading config file: %v", err)
		}
	}
	
	if verbose {
		logrus.SetLevel(logrus.DebugLevel)
	}
}

func runProxy(cmd *cobra.Command, args []string) {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		logrus.Fatalf("Failed to load configuration: %v", err)
	}

	// Create proxy server
	proxyServer := proxy.NewServer(cfg)

	// Create HTTP server
	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.Server.Port),
		Handler: proxyServer,
	}

	// Start server in a goroutine
	go func() {
		logrus.Infof("Starting proxy server on port %d", cfg.Server.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logrus.Fatalf("Server failed to start: %v", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	logrus.Info("Shutting down server...")

	// Give outstanding requests 30 seconds to complete
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		logrus.Fatalf("Server forced to shutdown: %v", err)
	}

	logrus.Info("Server exited")
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}