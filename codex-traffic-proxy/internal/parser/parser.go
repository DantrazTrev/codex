package parser

import (
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	"codex-traffic-proxy/internal/config"
	"codex-traffic-proxy/internal/logger"
	"codex-traffic-proxy/pkg/models"
)

type Parser struct {
	config *config.Config
	logger *logger.Logger
}

func NewParser(cfg *config.Config, log *logger.Logger) *Parser {
	return &Parser{
		config: cfg,
		logger: log,
	}
}

func (p *Parser) ParseRequest(req *http.Request, reqInfo *models.RequestInfo) error {
	// Extract API keys from headers and URL
	if p.config.Parser.ExtractAPIKeys {
		p.extractAPIKeys(req, reqInfo)
	}

	// Extract endpoints
	if p.config.Parser.ExtractEndpoints {
		p.extractEndpoints(req, reqInfo)
	}

	// Extract tokens from various sources
	if p.config.Parser.ExtractTokens {
		p.extractTokens(req, reqInfo)
	}

	// Analyze request patterns
	p.analyzeRequestPatterns(req, reqInfo)

	return nil
}

func (p *Parser) ParseResponse(resp *http.Response, reqInfo *models.RequestInfo) error {
	// Parse response body for insights
	if reqInfo.ResponseBody != "" {
		p.parseResponseBody(reqInfo)
	}

	// Analyze response patterns
	p.analyzeResponsePatterns(resp, reqInfo)

	return nil
}

func (p *Parser) extractAPIKeys(req *http.Request, reqInfo *models.RequestInfo) {
	// Check Authorization header
	if auth := req.Header.Get("Authorization"); auth != "" {
		if strings.HasPrefix(auth, "Bearer ") {
			token := strings.TrimPrefix(auth, "Bearer ")
			reqInfo.Tokens = append(reqInfo.Tokens, models.TokenInfo{
				Type:      "Bearer",
				Value:     token,
				Location:  "Authorization header",
				Pattern:   "Bearer [token]",
				Timestamp: time.Now(),
			})
		}
	}

	// Check for API keys in query parameters
	if req.URL.RawQuery != "" {
		query := req.URL.Query()
		for key, values := range query {
			for _, value := range values {
				if p.matchesSensitivePattern(value) {
					reqInfo.APIKeys = append(reqInfo.APIKeys, models.APIKeyInfo{
						Key:       value,
						Location:  fmt.Sprintf("query parameter: %s", key),
						Timestamp: time.Now(),
					})
				}
			}
		}
	}

	// Check for API keys in headers
	for name, values := range req.Header {
		for _, value := range values {
			if p.matchesSensitivePattern(value) {
				reqInfo.APIKeys = append(reqInfo.APIKeys, models.APIKeyInfo{
					Key:       value,
					Location:  fmt.Sprintf("header: %s", name),
					Timestamp: time.Now(),
				})
			}
		}
	}
}

func (p *Parser) extractEndpoints(req *http.Request, reqInfo *models.RequestInfo) {
	endpoint := models.EndpointInfo{
		URL:       req.URL.String(),
		Host:      req.Host,
		Path:      req.URL.Path,
		Method:    req.Method,
		Timestamp: time.Now(),
	}

	// Determine service type based on host
	switch {
	case strings.Contains(req.Host, "api.openai.com"):
		endpoint.Service = "openai"
	case strings.Contains(req.Host, "chatgpt.com"):
		endpoint.Service = "chatgpt"
	case strings.Contains(req.Host, "backend-api"):
		endpoint.Service = "chatgpt-backend"
	case strings.Contains(req.Host, "codex"):
		endpoint.Service = "codex"
	default:
		endpoint.Service = "unknown"
	}

	reqInfo.Endpoints = append(reqInfo.Endpoints, endpoint)
}

func (p *Parser) extractTokens(req *http.Request, reqInfo *models.RequestInfo) {
	// Extract from various token sources
	if cookie := req.Header.Get("Cookie"); cookie != "" {
		p.extractTokensFromCookie(cookie, reqInfo)
	}

	// Check for tokens in request body (if present)
	if req.Body != nil {
		// Note: This would need to read the body, but goproxy handles this differently
		// For now, we'll rely on header extraction
	}
}

func (p *Parser) extractTokensFromCookie(cookie string, reqInfo *models.RequestInfo) {
	// Common token patterns in cookies
	tokenPatterns := map[string]string{
		"session":       "__Secure-next-auth.session-token",
		"csrf":         "csrf-token",
		"access_token": "access_token",
	}

	for tokenType, cookieName := range tokenPatterns {
		if strings.Contains(cookie, cookieName) {
			// Extract token value (simplified - would need proper cookie parsing)
			parts := strings.Split(cookie, cookieName+"=")
			if len(parts) > 1 {
				tokenValue := strings.Split(parts[1], ";")[0]
				reqInfo.Tokens = append(reqInfo.Tokens, models.TokenInfo{
					Type:      tokenType,
					Value:     tokenValue,
					Location:  "Cookie",
					Pattern:   fmt.Sprintf("%s=...", cookieName),
					Timestamp: time.Now(),
				})
			}
		}
	}
}

func (p *Parser) analyzeRequestPatterns(req *http.Request, reqInfo *models.RequestInfo) {
	// Analyze for common patterns
	patterns := map[string]string{
		"chat_completion": "/chat/completions",
		"conversation":    "/conversation",
		"backend_api":     "/backend-api",
		"wham":           "/wham/",
		"codex_api":      "/api/codex",
	}

	for patternName, pattern := range patterns {
		if strings.Contains(req.URL.Path, pattern) {
			reqInfo.Patterns = append(reqInfo.Patterns, patternName)
		}
	}

	// Check for streaming requests
	if strings.Contains(req.Header.Get("Accept"), "text/event-stream") {
		reqInfo.Patterns = append(reqInfo.Patterns, "streaming")
	}

	// Check for WebSocket upgrades
	if req.Header.Get("Upgrade") == "websocket" {
		reqInfo.Patterns = append(reqInfo.Patterns, "websocket")
	}
}

func (p *Parser) parseResponseBody(reqInfo *models.RequestInfo) {
	body := reqInfo.ResponseBody

	// Try to parse as JSON for structured analysis
	var jsonData interface{}
	if err := json.Unmarshal([]byte(body), &jsonData); err == nil {
		reqInfo.ResponseJSON = jsonData
		p.analyzeJSONResponse(jsonData, reqInfo)
	}
}

func (p *Parser) analyzeJSONResponse(data interface{}, reqInfo *models.RequestInfo) {
	// Analyze JSON response for common fields
	jsonMap, ok := data.(map[string]interface{})
	if !ok {
		return
	}

	// Look for usage information
	if usage, exists := jsonMap["usage"]; exists {
		reqInfo.Usage = p.extractUsageInfo(usage)
	}

	// Look for token information
	if choices, exists := jsonMap["choices"]; exists {
		if choicesArray, ok := choices.([]interface{}); ok {
			for _, choice := range choicesArray {
				if choiceMap, ok := choice.(map[string]interface{}); ok {
					if finishReason, exists := choiceMap["finish_reason"]; exists {
						reqInfo.FinishReasons = append(reqInfo.FinishReasons, fmt.Sprintf("%v", finishReason))
					}
				}
			}
		}
	}

	// Look for model information
	if model, exists := jsonMap["model"]; exists {
		reqInfo.Models = append(reqInfo.Models, fmt.Sprintf("%v", model))
	}
}

func (p *Parser) extractUsageInfo(usage interface{}) models.UsageInfo {
	usageMap, ok := usage.(map[string]interface{})
	if !ok {
		return models.UsageInfo{}
	}

	usageInfo := models.UsageInfo{
		Timestamp: time.Now(),
	}

	if promptTokens, exists := usageMap["prompt_tokens"]; exists {
		if tokens, ok := promptTokens.(float64); ok {
			usageInfo.PromptTokens = int(tokens)
		}
	}

	if completionTokens, exists := usageMap["completion_tokens"]; exists {
		if tokens, ok := completionTokens.(float64); ok {
			usageInfo.CompletionTokens = int(tokens)
		}
	}

	if totalTokens, exists := usageMap["total_tokens"]; exists {
		if tokens, ok := totalTokens.(float64); ok {
			usageInfo.TotalTokens = int(tokens)
		}
	}

	return usageInfo
}

func (p *Parser) analyzeResponsePatterns(resp *http.Response, reqInfo *models.RequestInfo) {
	// Analyze response headers for patterns
	contentType := resp.Header.Get("Content-Type")
	if contentType != "" {
		reqInfo.ContentTypes = append(reqInfo.ContentTypes, contentType)
	}

	// Check for streaming responses
	if strings.Contains(contentType, "text/event-stream") {
		reqInfo.Patterns = append(reqInfo.Patterns, "streaming_response")
	}

	// Check for rate limiting
	if resp.StatusCode == 429 {
		reqInfo.Patterns = append(reqInfo.Patterns, "rate_limited")
	}

	// Check for server errors
	if resp.StatusCode >= 500 {
		reqInfo.Patterns = append(reqInfo.Patterns, "server_error")
	}
}

func (p *Parser) matchesSensitivePattern(value string) bool {
	for _, pattern := range p.config.Parser.SensitivePatterns {
		matched, err := regexp.MatchString(pattern, value)
		if err == nil && matched {
			return true
		}
	}
	return false
}