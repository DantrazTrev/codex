package main

import (
	"fmt"
	"os"

	"github.com/openai/codex/proxy-parser/pkg/config"
	"github.com/openai/codex/proxy-parser/pkg/logger"
	"github.com/openai/codex/proxy-parser/pkg/parser"
	"github.com/openai/codex/proxy-parser/pkg/proxy"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

var (
	cfgFile      string
	port         int
	host         string
	verbose      bool
	outputFile   string
	analyze      bool
	genaiOnly    bool
	monitorTerms string
)

var rootCmd = &cobra.Command{
	Use:   "codex-proxy",
	Short: "Proxy server for monitoring Codex CLI traffic",
	Long: `A proxy server that intercepts and analyzes HTTP/HTTPS traffic 
from the Codex CLI to understand GenAI and coding tool interactions.`,
}

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start the proxy server",
	RunE: func(cmd *cobra.Command, args []string) error {
		// Initialize logger
		log, err := logger.NewLogger(verbose)
		if err != nil {
			return fmt.Errorf("failed to initialize logger: %w", err)
		}
		defer log.Sync()

		// Load configuration
		cfg := &config.Config{
			Port:       port,
			Host:       host,
			OutputFile: outputFile,
			Verbose:    verbose,
			Analyze:    analyze,
			GenAIOnly:  genaiOnly,
		}

		if cfgFile != "" {
			if err := cfg.LoadFromFile(cfgFile); err != nil {
				log.Error("Failed to load config file", zap.Error(err))
				return err
			}
		}

		// Create and start proxy server
		server, err := proxy.NewServer(cfg, log)
		if err != nil {
			log.Error("Failed to create proxy server", zap.Error(err))
			return err
		}

		log.Info("Starting proxy server",
			zap.String("host", cfg.Host),
			zap.Int("port", cfg.Port),
			zap.Bool("analyze", cfg.Analyze),
		)

		return server.Start()
	},
}

var analyzeCmd = &cobra.Command{
	Use:   "analyze",
	Short: "Analyze captured traffic",
	RunE: func(cmd *cobra.Command, args []string) error {
		log, err := logger.NewLogger(verbose)
		if err != nil {
			return fmt.Errorf("failed to initialize logger: %w", err)
		}
		defer log.Sync()

		if outputFile == "" {
			return fmt.Errorf("input file required (--input)")
		}

		analyzer := parser.NewAnalyzer(log)
		return analyzer.AnalyzeFile(outputFile, &parser.AnalyzeOptions{
			GenAIOnly: genaiOnly,
			Verbose:   verbose,
		})
	},
}

var statsCmd = &cobra.Command{
	Use:   "stats",
	Short: "Generate statistics from captured traffic",
	RunE: func(cmd *cobra.Command, args []string) error {
		log, err := logger.NewLogger(verbose)
		if err != nil {
			return fmt.Errorf("failed to initialize logger: %w", err)
		}
		defer log.Sync()

		if outputFile == "" {
			return fmt.Errorf("input file required (--input)")
		}

		analyzer := parser.NewAnalyzer(log)
		return analyzer.GenerateStats(outputFile)
	},
}

var generateCertCmd = &cobra.Command{
	Use:   "generate-cert",
	Short: "Generate CA certificate for HTTPS interception",
	RunE: func(cmd *cobra.Command, args []string) error {
		return proxy.GenerateCACertificate()
	},
}

func init() {
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file path")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose output")

	startCmd.Flags().IntVarP(&port, "port", "p", 8080, "proxy port")
	startCmd.Flags().StringVar(&host, "host", "0.0.0.0", "proxy host")
	startCmd.Flags().StringVarP(&outputFile, "output", "o", "", "output file for captured traffic")
	startCmd.Flags().BoolVar(&analyze, "analyze", false, "enable real-time analysis")
	startCmd.Flags().BoolVar(&genaiOnly, "genai-highlight", false, "highlight GenAI traffic")
	startCmd.Flags().StringVar(&monitorTerms, "monitor", "", "comma-separated terms to monitor")

	analyzeCmd.Flags().StringVarP(&outputFile, "input", "i", "", "input file to analyze")
	analyzeCmd.Flags().BoolVar(&genaiOnly, "genai-only", false, "show only GenAI traffic")

	statsCmd.Flags().StringVarP(&outputFile, "input", "i", "", "input file for statistics")

	rootCmd.AddCommand(startCmd)
	rootCmd.AddCommand(analyzeCmd)
	rootCmd.AddCommand(statsCmd)
	rootCmd.AddCommand(generateCertCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}