package proxy

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/elazarl/goproxy"
	"github.com/openai/codex/proxy-parser/pkg/config"
	"github.com/openai/codex/proxy-parser/pkg/storage"
	"go.uber.org/zap"
)

// Server represents the proxy server
type Server struct {
	config     *config.Config
	logger     *zap.Logger
	proxy      *goproxy.ProxyHttpServer
	storage    *storage.Storage
	mu         sync.RWMutex
	statistics *Statistics
}

// Statistics holds traffic statistics
type Statistics struct {
	TotalRequests   int64
	TotalResponses  int64
	GenAIRequests   int64
	BytesSent       int64
	BytesReceived   int64
	Endpoints       map[string]int64
	StatusCodes     map[int]int64
	ModelCalls      map[string]int64
}

// NewServer creates a new proxy server
func NewServer(cfg *config.Config, logger *zap.Logger) (*Server, error) {
	proxyServer := goproxy.NewProxyHttpServer()
	proxyServer.Verbose = cfg.Verbose

	// Enable HTTPS interception if certificate exists
	if certExists() {
		setupHTTPS(proxyServer)
	}

	stor, err := storage.NewStorage(cfg.OutputFile)
	if err != nil {
		return nil, fmt.Errorf("failed to create storage: %w", err)
	}

	server := &Server{
		config:  cfg,
		logger:  logger,
		proxy:   proxyServer,
		storage: stor,
		statistics: &Statistics{
			Endpoints:   make(map[string]int64),
			StatusCodes: make(map[int]int64),
			ModelCalls:  make(map[string]int64),
		},
	}

	// Set up request/response handlers
	server.setupHandlers()

	return server, nil
}

// setupHandlers configures the proxy handlers
func (s *Server) setupHandlers() {
	// Handle HTTP requests
	s.proxy.OnRequest().DoFunc(func(req *http.Request, ctx *goproxy.ProxyCtx) (*http.Request, *http.Response) {
		s.handleRequest(req, ctx)
		return req, nil
	})

	// Handle HTTP responses
	s.proxy.OnResponse().DoFunc(func(resp *http.Response, ctx *goproxy.ProxyCtx) *http.Response {
		s.handleResponse(resp, ctx)
		return resp
	})

	// Handle HTTPS CONNECT
	s.proxy.OnRequest().HandleConnect(goproxy.AlwaysMitm)
}

// handleRequest processes incoming requests
func (s *Server) handleRequest(req *http.Request, ctx *goproxy.ProxyCtx) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.statistics.TotalRequests++

	// Create request record
	record := &storage.TrafficRecord{
		ID:        fmt.Sprintf("%d", ctx.Session),
		Timestamp: time.Now(),
		Type:      "request",
		Method:    req.Method,
		URL:       req.URL.String(),
		Headers:   flattenHeaders(req.Header),
	}

	// Capture request body if present
	if req.Body != nil && req.ContentLength > 0 {
		bodyBytes, err := io.ReadAll(req.Body)
		if err == nil {
			record.Body = string(bodyBytes)
			// Restore body for actual request
			req.Body = io.NopCloser(bytes.NewReader(bodyBytes))
			s.statistics.BytesSent += int64(len(bodyBytes))
		}
	}

	// Track endpoint
	endpoint := extractEndpoint(req.URL)
	s.statistics.Endpoints[endpoint]++

	// Check if this is GenAI traffic
	if s.isGenAITraffic(req, record.Body) {
		s.statistics.GenAIRequests++
		record.Tags = append(record.Tags, "genai")
		
		// Extract model information
		if model := s.extractModel(req, record.Body); model != "" {
			s.statistics.ModelCalls[model]++
			record.Tags = append(record.Tags, "model:"+model)
		}
	}

	// Log based on configuration
	if s.config.Verbose || (s.config.GenAIOnly && s.isGenAITraffic(req, record.Body)) {
		s.logger.Info("Request intercepted",
			zap.String("method", req.Method),
			zap.String("url", req.URL.String()),
			zap.String("endpoint", endpoint),
			zap.Strings("tags", record.Tags),
		)
	}

	// Store the record
	if err := s.storage.Store(record); err != nil {
		s.logger.Error("Failed to store request", zap.Error(err))
	}

	// Real-time analysis if enabled
	if s.config.Analyze {
		s.analyzeTraffic(record)
	}
}

// handleResponse processes responses
func (s *Server) handleResponse(resp *http.Response, ctx *goproxy.ProxyCtx) {
	if resp == nil {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.statistics.TotalResponses++
	s.statistics.StatusCodes[resp.StatusCode]++

	// Create response record
	record := &storage.TrafficRecord{
		ID:         fmt.Sprintf("%d", ctx.Session),
		Timestamp:  time.Now(),
		Type:       "response",
		StatusCode: resp.StatusCode,
		Headers:    flattenHeaders(resp.Header),
	}

	// Capture response body if present
	if resp.Body != nil {
		bodyBytes, err := io.ReadAll(resp.Body)
		if err == nil {
			record.Body = string(bodyBytes)
			// Restore body for actual response
			resp.Body = io.NopCloser(bytes.NewReader(bodyBytes))
			s.statistics.BytesReceived += int64(len(bodyBytes))

			// Check for streaming responses (SSE)
			if strings.Contains(resp.Header.Get("Content-Type"), "text/event-stream") {
				record.Tags = append(record.Tags, "streaming")
			}
		}
	}

	// Check if response contains GenAI data
	if s.isGenAIResponse(resp, record.Body) {
		record.Tags = append(record.Tags, "genai-response")
	}

	// Log based on configuration
	if s.config.Verbose {
		s.logger.Info("Response intercepted",
			zap.Int("status", resp.StatusCode),
			zap.String("content-type", resp.Header.Get("Content-Type")),
			zap.Strings("tags", record.Tags),
		)
	}

	// Store the record
	if err := s.storage.Store(record); err != nil {
		s.logger.Error("Failed to store response", zap.Error(err))
	}
}

// isGenAITraffic checks if the request is related to GenAI
func (s *Server) isGenAITraffic(req *http.Request, body string) bool {
	urlStr := req.URL.String()
	
	// Check common GenAI endpoints
	genAIPatterns := []string{
		"/api/codex",
		"/backend-api",
		"/wham",
		"/completions",
		"/chat/completions",
		"/v1/engines",
		"openai.com",
		"anthropic.com",
		"chat.openai.com",
		"chatgpt.com",
	}

	for _, pattern := range genAIPatterns {
		if strings.Contains(urlStr, pattern) {
			return true
		}
	}

	// Check request body for model references
	if body != "" {
		modelKeywords := []string{
			"model", "gpt-4", "gpt-3.5", "claude",
			"temperature", "max_tokens", "messages",
			"prompt", "completion", "stream",
		}
		
		bodyLower := strings.ToLower(body)
		for _, keyword := range modelKeywords {
			if strings.Contains(bodyLower, keyword) {
				return true
			}
		}
	}

	// Check Authorization header for OpenAI/Anthropic patterns
	auth := req.Header.Get("Authorization")
	if strings.HasPrefix(auth, "Bearer sk-") || strings.Contains(auth, "anthropic") {
		return true
	}

	return false
}

// isGenAIResponse checks if the response contains GenAI data
func (s *Server) isGenAIResponse(resp *http.Response, body string) bool {
	if body == "" {
		return false
	}

	// Check for common GenAI response patterns
	patterns := []string{
		`"model":`, `"choices":`, `"completion":`,
		`"usage":`, `"tokens":`, `"finish_reason":`,
		`"content":`, `"role":`, `"assistant"`,
	}

	for _, pattern := range patterns {
		if strings.Contains(body, pattern) {
			return true
		}
	}

	return false
}

// extractModel attempts to extract the model name from request
func (s *Server) extractModel(req *http.Request, body string) string {
	if body == "" {
		return ""
	}

	// Try to parse as JSON
	var data map[string]interface{}
	if err := json.Unmarshal([]byte(body), &data); err == nil {
		// Look for model field
		if model, ok := data["model"].(string); ok {
			return model
		}
		
		// Check nested structures
		if messages, ok := data["messages"].([]interface{}); ok && len(messages) > 0 {
			if msg, ok := messages[0].(map[string]interface{}); ok {
				if model, ok := msg["model"].(string); ok {
					return model
				}
			}
		}
	}

	// Check URL path for model indicators
	urlPath := req.URL.Path
	if strings.Contains(urlPath, "gpt-4") {
		return "gpt-4"
	} else if strings.Contains(urlPath, "gpt-3.5") {
		return "gpt-3.5-turbo"
	}

	return ""
}

// analyzeTraffic performs real-time analysis
func (s *Server) analyzeTraffic(record *storage.TrafficRecord) {
	if record.Type == "request" && len(record.Tags) > 0 {
		// Log interesting findings
		for _, tag := range record.Tags {
			if strings.HasPrefix(tag, "model:") {
				model := strings.TrimPrefix(tag, "model:")
				s.logger.Info("AI Model detected",
					zap.String("model", model),
					zap.String("url", record.URL),
				)
			}
		}
	}
}

// extractEndpoint extracts the endpoint path from URL
func extractEndpoint(u *url.URL) string {
	path := u.Path
	if path == "" {
		path = "/"
	}
	
	// Simplify path by removing IDs and parameters
	parts := strings.Split(path, "/")
	simplified := []string{}
	
	for _, part := range parts {
		if part == "" {
			continue
		}
		// Replace UUIDs and numeric IDs with placeholders
		if len(part) > 20 || isNumeric(part) {
			simplified = append(simplified, "{id}")
		} else {
			simplified = append(simplified, part)
		}
	}
	
	return "/" + strings.Join(simplified, "/")
}

// isNumeric checks if a string is numeric
func isNumeric(s string) bool {
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return len(s) > 0
}

// flattenHeaders converts http.Header to a simple map
func flattenHeaders(headers http.Header) map[string]string {
	flat := make(map[string]string)
	for key, values := range headers {
		// Skip sensitive headers
		if isSensitiveHeader(key) {
			flat[key] = "[REDACTED]"
			continue
		}
		flat[key] = strings.Join(values, ", ")
	}
	return flat
}

// isSensitiveHeader checks if a header should be redacted
func isSensitiveHeader(header string) bool {
	sensitive := []string{
		"authorization", "cookie", "set-cookie",
		"x-api-key", "api-key", "x-auth-token",
	}
	
	headerLower := strings.ToLower(header)
	for _, s := range sensitive {
		if strings.Contains(headerLower, s) {
			return true
		}
	}
	return false
}

// Start starts the proxy server
func (s *Server) Start() error {
	addr := fmt.Sprintf("%s:%d", s.config.Host, s.config.Port)
	
	s.logger.Info("Proxy server starting",
		zap.String("address", addr),
		zap.String("output", s.config.OutputFile),
	)

	// Start statistics reporter
	go s.reportStatistics()

	return http.ListenAndServe(addr, s.proxy)
}

// reportStatistics periodically reports statistics
func (s *Server) reportStatistics() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		s.mu.RLock()
		stats := s.statistics
		s.mu.RUnlock()

		s.logger.Info("Traffic statistics",
			zap.Int64("total_requests", stats.TotalRequests),
			zap.Int64("total_responses", stats.TotalResponses),
			zap.Int64("genai_requests", stats.GenAIRequests),
			zap.Int64("bytes_sent", stats.BytesSent),
			zap.Int64("bytes_received", stats.BytesReceived),
		)
	}
}

// GetStatistics returns current statistics
func (s *Server) GetStatistics() *Statistics {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	// Create a copy to avoid race conditions
	statsCopy := &Statistics{
		TotalRequests:  s.statistics.TotalRequests,
		TotalResponses: s.statistics.TotalResponses,
		GenAIRequests:  s.statistics.GenAIRequests,
		BytesSent:      s.statistics.BytesSent,
		BytesReceived:  s.statistics.BytesReceived,
		Endpoints:      make(map[string]int64),
		StatusCodes:    make(map[int]int64),
		ModelCalls:     make(map[string]int64),
	}
	
	for k, v := range s.statistics.Endpoints {
		statsCopy.Endpoints[k] = v
	}
	for k, v := range s.statistics.StatusCodes {
		statsCopy.StatusCodes[k] = v
	}
	for k, v := range s.statistics.ModelCalls {
		statsCopy.ModelCalls[k] = v
	}
	
	return statsCopy
}