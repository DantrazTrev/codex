package parser

import (
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	"codex-proxy-parser/internal/config"
)

// TrafficParser parses HTTP traffic to identify GenAI and vibe coding requests
type TrafficParser struct {
	config *config.ParserConfig
	genAIRegex *regexp.Regexp
	vibeCodingRegex *regexp.Regexp
}

// ParsedData represents parsed request data
type ParsedData struct {
	Timestamp     time.Time `json:"timestamp"`
	Type          string    `json:"type"`
	Endpoint      string    `json:"endpoint"`
	Method        string    `json:"method"`
	Model         string    `json:"model,omitempty"`
	Tokens        int       `json:"tokens,omitempty"`
	IsGenAI       bool      `json:"is_genai"`
	IsVibeCoding  bool      `json:"is_vibe_coding"`
	RequestID     string    `json:"request_id,omitempty"`
	UserAgent     string    `json:"user_agent,omitempty"`
	ContentLength int64     `json:"content_length"`
	RawBody       string    `json:"raw_body,omitempty"`
}

// NewTrafficParser creates a new traffic parser
func NewTrafficParser(cfg *config.ParserConfig) *TrafficParser {
	// Compile regex patterns for GenAI and vibe coding detection
	genAIRegex := regexp.MustCompile(`(?i)(openai|anthropic|claude|gpt|dall-e|whisper|embedding|completion|chat)`)
	vibeCodingRegex := regexp.MustCompile(`(?i)(codex|cursor|github|copilot|tabnine|kite|intellicode|vibe|coding|assistant)`)

	return &TrafficParser{
		config: cfg,
		genAIRegex: genAIRegex,
		vibeCodingRegex: vibeCodingRegex,
	}
}

// ParseRequest parses an HTTP request to extract relevant information
func (p *TrafficParser) ParseRequest(r *http.Request, body []byte) (*ParsedData, error) {
	// Check if we should parse this request
	if !p.shouldParse(r) {
		return nil, nil
	}

	// Create parsed data structure
	parsed := &ParsedData{
		Timestamp:     time.Now(),
		Endpoint:      r.URL.Path,
		Method:        r.Method,
		UserAgent:     r.UserAgent(),
		ContentLength: int64(len(body)),
	}

	// Parse request body if it's JSON
	if r.Header.Get("Content-Type") == "application/json" && len(body) > 0 {
		if err := p.parseJSONBody(body, parsed); err != nil {
			return nil, fmt.Errorf("failed to parse JSON body: %w", err)
		}
	}

	// Detect GenAI and vibe coding patterns
	p.detectPatterns(r, body, parsed)

	// Store raw body if configured and within size limit
	if p.config.MaxBodySize > 0 && int64(len(body)) <= p.config.MaxBodySize {
		parsed.RawBody = string(body)
	}

	return parsed, nil
}

// shouldParse determines if a request should be parsed
func (p *TrafficParser) shouldParse(r *http.Request) bool {
	// Check if parsing is enabled
	if !p.config.Enabled {
		return false
	}

	// Check if we should parse GenAI requests
	if p.config.ParseGenAI && p.isGenAIRequest(r) {
		return true
	}

	// Check if we should parse vibe coding requests
	if p.config.ParseVibeCoding && p.isVibeCodingRequest(r) {
		return true
	}

	// Check against keywords
	for _, keyword := range p.config.Keywords {
		if strings.Contains(strings.ToLower(r.URL.Path), strings.ToLower(keyword)) ||
		   strings.Contains(strings.ToLower(r.Host), strings.ToLower(keyword)) {
			return true
		}
	}

	return false
}

// isGenAIRequest checks if a request is likely a GenAI request
func (p *TrafficParser) isGenAIRequest(r *http.Request) bool {
	// Check URL path
	if p.genAIRegex.MatchString(r.URL.Path) {
		return true
	}

	// Check host
	if p.genAIRegex.MatchString(r.Host) {
		return true
	}

	// Check User-Agent
	if p.genAIRegex.MatchString(r.UserAgent()) {
		return true
	}

	return false
}

// isVibeCodingRequest checks if a request is likely a vibe coding request
func (p *TrafficParser) isVibeCodingRequest(r *http.Request) bool {
	// Check URL path
	if p.vibeCodingRegex.MatchString(r.URL.Path) {
		return true
	}

	// Check host
	if p.vibeCodingRegex.MatchString(r.Host) {
		return true
	}

	// Check User-Agent
	if p.vibeCodingRegex.MatchString(r.UserAgent()) {
		return true
	}

	return false
}

// parseJSONBody parses JSON request body to extract relevant information
func (p *TrafficParser) parseJSONBody(body []byte, parsed *ParsedData) error {
	var data map[string]interface{}
	if err := json.Unmarshal(body, &data); err != nil {
		return err
	}

	// Extract model information
	if model, ok := data["model"].(string); ok {
		parsed.Model = model
	}

	// Extract request ID
	if id, ok := data["request_id"].(string); ok {
		parsed.RequestID = id
	}

	// Determine request type based on endpoint and content
	parsed.Type = p.determineRequestType(parsed.Endpoint, data)

	// Estimate tokens (rough approximation)
	parsed.Tokens = p.estimateTokens(body)

	return nil
}

// determineRequestType determines the type of request based on endpoint and content
func (p *TrafficParser) determineRequestType(endpoint string, data map[string]interface{}) string {
	// Check for common AI API endpoints
	if strings.Contains(endpoint, "/v1/chat/completions") {
		return "chat_completion"
	}
	if strings.Contains(endpoint, "/v1/completions") {
		return "completion"
	}
	if strings.Contains(endpoint, "/v1/embeddings") {
		return "embedding"
	}
	if strings.Contains(endpoint, "/v1/images/generations") {
		return "image_generation"
	}
	if strings.Contains(endpoint, "/v1/audio/transcriptions") {
		return "audio_transcription"
	}
	if strings.Contains(endpoint, "/v1/audio/translations") {
		return "audio_translation"
	}

	// Check for Codex-specific endpoints
	if strings.Contains(endpoint, "/codex") || strings.Contains(endpoint, "/cursor") {
		return "codex_request"
	}

	// Default to generic API request
	return "api_request"
}

// estimateTokens provides a rough estimate of token count
func (p *TrafficParser) estimateTokens(body []byte) int {
	// Very rough estimation: ~4 characters per token for English text
	// This is not accurate but gives a ballpark figure
	return len(body) / 4
}

// detectPatterns detects GenAI and vibe coding patterns in the request
func (p *TrafficParser) detectPatterns(r *http.Request, body []byte, parsed *ParsedData) {
	// Check request body for patterns
	bodyStr := string(body)
	
	// Detect GenAI patterns
	if p.genAIRegex.MatchString(bodyStr) {
		parsed.IsGenAI = true
	}

	// Detect vibe coding patterns
	if p.vibeCodingRegex.MatchString(bodyStr) {
		parsed.IsVibeCoding = true
	}

	// Additional pattern detection based on content
	if strings.Contains(strings.ToLower(bodyStr), "temperature") ||
	   strings.Contains(strings.ToLower(bodyStr), "max_tokens") ||
	   strings.Contains(strings.ToLower(bodyStr), "prompt") {
		parsed.IsGenAI = true
	}

	if strings.Contains(strings.ToLower(bodyStr), "code") ||
	   strings.Contains(strings.ToLower(bodyStr), "function") ||
	   strings.Contains(strings.ToLower(bodyStr), "class") ||
	   strings.Contains(strings.ToLower(bodyStr), "variable") {
		parsed.IsVibeCoding = true
	}
}

// ParseResponse parses an HTTP response (for future use)
func (p *TrafficParser) ParseResponse(resp *http.Response, body []byte) (*ParsedData, error) {
	// This could be extended to parse response data as well
	// For now, we focus on request parsing
	return nil, fmt.Errorf("response parsing not implemented")
}