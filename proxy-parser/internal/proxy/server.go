package proxy

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"codex-proxy-parser/internal/config"
	"codex-proxy-parser/internal/parser"

	"github.com/sirupsen/logrus"
)

// Server represents the proxy server
type Server struct {
	config *config.Config
	client *http.Client
	parser *parser.TrafficParser
}

// NewServer creates a new proxy server
func NewServer(cfg *config.Config) *Server {
	// Create HTTP client with configuration
	client := &http.Client{
		Timeout: time.Duration(cfg.Proxy.Timeout) * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: cfg.Proxy.SkipTLSVerify,
			},
		},
	}

	// Create traffic parser
	trafficParser := parser.NewTrafficParser(&cfg.Parser)

	return &Server{
		config: cfg,
		client: client,
		parser: trafficParser,
	}
}

// ServeHTTP implements the http.Handler interface
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Log incoming request
	logrus.WithFields(logrus.Fields{
		"method": r.Method,
		"url":    r.URL.String(),
		"host":   r.Host,
		"remote": r.RemoteAddr,
	}).Info("Incoming request")

	// Handle different HTTP methods
	switch r.Method {
	case http.MethodConnect:
		s.handleCONNECT(w, r)
	case http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
		s.handleHTTP(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleCONNECT handles CONNECT method for HTTPS tunneling
func (s *Server) handleCONNECT(w http.ResponseWriter, r *http.Request) {
	logrus.WithField("target", r.URL.Host).Info("CONNECT request")
	
	// For now, we'll reject CONNECT requests as we're focusing on HTTP traffic
	// In a production environment, you might want to implement proper HTTPS tunneling
	http.Error(w, "CONNECT method not supported", http.StatusMethodNotAllowed)
}

// handleHTTP handles regular HTTP requests
func (s *Server) handleHTTP(w http.ResponseWriter, r *http.Request) {
	// Read request body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		logrus.WithError(err).Error("Failed to read request body")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Parse traffic if parser is enabled
	if s.config.Parser.Enabled {
		parsedData, err := s.parser.ParseRequest(r, body)
		if err != nil {
			logrus.WithError(err).Warn("Failed to parse request")
		} else if parsedData != nil {
			logrus.WithFields(logrus.Fields{
				"type":        parsedData.Type,
				"endpoint":    parsedData.Endpoint,
				"model":       parsedData.Model,
				"tokens":      parsedData.Tokens,
				"is_genai":    parsedData.IsGenAI,
				"is_vibe_coding": parsedData.IsVibeCoding,
			}).Info("Parsed request")
		}
	}

	// Create new request to target server
	targetURL, err := s.buildTargetURL(r.URL)
	if err != nil {
		logrus.WithError(err).Error("Failed to build target URL")
		http.Error(w, "Bad Gateway", http.StatusBadGateway)
		return
	}

	// Create new request
	req, err := http.NewRequestWithContext(r.Context(), r.Method, targetURL, bytes.NewReader(body))
	if err != nil {
		logrus.WithError(err).Error("Failed to create request")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Copy headers
	s.copyHeaders(r.Header, req.Header)

	// Add custom headers from config
	for key, value := range s.config.Proxy.Headers {
		req.Header.Set(key, value)
	}

	// Execute request
	resp, err := s.client.Do(req)
	if err != nil {
		logrus.WithError(err).Error("Failed to execute request")
		http.Error(w, "Bad Gateway", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	// Copy response headers
	s.copyHeaders(resp.Header, w.Header())

	// Set status code
	w.WriteHeader(resp.StatusCode)

	// Copy response body
	if _, err := io.Copy(w, resp.Body); err != nil {
		logrus.WithError(err).Error("Failed to copy response body")
	}
}

// buildTargetURL builds the target URL for the request
func (s *Server) buildTargetURL(originalURL *url.URL) (string, error) {
	// Parse target URL
	target, err := url.Parse(s.config.Proxy.TargetURL)
	if err != nil {
		return "", fmt.Errorf("invalid target URL: %w", err)
	}

	// Build new URL
	newURL := &url.URL{
		Scheme:   target.Scheme,
		Host:     target.Host,
		Path:     originalURL.Path,
		RawQuery: originalURL.RawQuery,
		Fragment: originalURL.Fragment,
	}

	return newURL.String(), nil
}

// copyHeaders copies headers from source to destination
func (s *Server) copyHeaders(src, dst http.Header) {
	for key, values := range src {
		// Skip hop-by-hop headers
		if isHopByHopHeader(key) {
			continue
		}
		for _, value := range values {
			dst.Add(key, value)
		}
	}
}

// isHopByHopHeader checks if a header is a hop-by-hop header
func isHopByHopHeader(header string) bool {
	hopByHopHeaders := []string{
		"Connection",
		"Keep-Alive",
		"Proxy-Authenticate",
		"Proxy-Authorization",
		"Te",
		"Trailers",
		"Transfer-Encoding",
		"Upgrade",
	}
	
	headerLower := strings.ToLower(header)
	for _, h := range hopByHopHeaders {
		if headerLower == strings.ToLower(h) {
			return true
		}
	}
	return false
}