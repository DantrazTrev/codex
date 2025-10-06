package codex

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"
	
	"github.com/okulis-ai/lumeus-application/gocore/pkg/goproxy"
)

// IntegrationExample shows how to integrate the Codex parser with your proxy
type IntegrationExample struct {
	codexClient *CodexClient
}

// NewIntegration creates a new integration instance
func NewIntegration() *IntegrationExample {
	return &IntegrationExample{
		codexClient: NewCodexClient(),
	}
}

// IsCodexTraffic determines if the request/response is Codex-related
func (i *IntegrationExample) IsCodexTraffic(req *http.Request) bool {
	host := req.Host
	path := req.URL.Path
	
	// Check common Codex/OpenAI endpoints
	codexHosts := []string{
		"api.openai.com",
		"openai.com",
		"chat.openai.com",
		"chatgpt.com",
	}
	
	for _, h := range codexHosts {
		if strings.Contains(host, h) {
			return true
		}
	}
	
	// Check path patterns
	codexPaths := []string{
		"/v1/chat/completions",
		"/v1/completions",
		"/v1/engines",
		"/api/codex",
		"/backend-api",
		"/wham",
		"/code_tasks",
		"/conversation",
	}
	
	for _, p := range codexPaths {
		if strings.Contains(path, p) {
			return true
		}
	}
	
	// Check for API key headers (OpenAI style)
	authHeader := req.Header.Get("Authorization")
	if strings.HasPrefix(authHeader, "Bearer sk-") {
		return true
	}
	
	return false
}

// GetEndpointType categorizes the type of Codex endpoint
func (i *IntegrationExample) GetEndpointType(path string) string {
	switch {
	case strings.Contains(path, "/chat/completions"):
		return "CHAT_COMPLETION"
	case strings.Contains(path, "/completions"):
		return "COMPLETION"
	case strings.Contains(path, "/embeddings"):
		return "EMBEDDING"
	case strings.Contains(path, "/images"):
		return "IMAGE_GENERATION"
	case strings.Contains(path, "/code_tasks"):
		return "CODE_TASK"
	case strings.Contains(path, "/conversation"):
		return "CONVERSATION"
	case strings.Contains(path, "/backend-api"):
		return "CHATGPT_BACKEND"
	case strings.Contains(path, "/wham"):
		return "WHAM_API"
	default:
		return "UNKNOWN"
	}
}

// ExtractModelInfo extracts model information from the request
func (i *IntegrationExample) ExtractModelInfo(reqBytes []byte) (model string, isStreaming bool) {
	var request map[string]interface{}
	if err := json.Unmarshal(reqBytes, &request); err != nil {
		return "", false
	}
	
	// Extract model
	if m, exists := request["model"]; exists {
		if modelStr, ok := m.(string); ok {
			model = modelStr
		}
	}
	
	// Check if streaming
	if s, exists := request["stream"]; exists {
		if stream, ok := s.(bool); ok {
			isStreaming = stream
		}
	}
	
	return model, isStreaming
}

// LogTraffic logs Codex traffic with context
func (i *IntegrationExample) LogTraffic(req *http.Request, reqBytes []byte, isRequest bool) {
	endpointType := i.GetEndpointType(req.URL.Path)
	model, isStreaming := i.ExtractModelInfo(reqBytes)
	
	direction := "REQUEST"
	if !isRequest {
		direction = "RESPONSE"
	}
	
	log.Printf("[CODEX %s] Endpoint: %s, Type: %s, Model: %s, Streaming: %v",
		direction,
		req.URL.Path,
		endpointType,
		model,
		isStreaming,
	)
}

// Example proxy handler integration
func (i *IntegrationExample) ProxyHandler(ctx *goproxy.ProxyCtx, req *http.Request, reqBytes []byte) {
	// Check if this is Codex traffic
	if !i.IsCodexTraffic(req) {
		return
	}
	
	// Log the traffic
	i.LogTraffic(req, reqBytes, true)
	
	// Parse using the Codex client
	// Note: You'd pass your actual SaasAppCache here
	// i.codexClient.ParseRequest(ctx, req, reqBytes, "action", yourCache)
}

// Statistics tracking for Codex traffic
type CodexStats struct {
	TotalRequests    int64
	ChatCompletions  int64
	Completions      int64
	CodeTasks        int64
	ModelsUsed       map[string]int64
	TotalTokensUsed  int64
	StreamingRequests int64
}

// NewCodexStats creates a new statistics tracker
func NewCodexStats() *CodexStats {
	return &CodexStats{
		ModelsUsed: make(map[string]int64),
	}
}

// UpdateStats updates statistics based on parsed data
func (s *CodexStats) UpdateStats(endpointType string, model string, isStreaming bool, tokens int64) {
	s.TotalRequests++
	
	switch endpointType {
	case "CHAT_COMPLETION":
		s.ChatCompletions++
	case "COMPLETION":
		s.Completions++
	case "CODE_TASK":
		s.CodeTasks++
	}
	
	if model != "" {
		s.ModelsUsed[model]++
	}
	
	if isStreaming {
		s.StreamingRequests++
	}
	
	s.TotalTokensUsed += tokens
}

// GetSummary returns a summary of the statistics
func (s *CodexStats) GetSummary() map[string]interface{} {
	return map[string]interface{}{
		"total_requests":     s.TotalRequests,
		"chat_completions":   s.ChatCompletions,
		"completions":        s.Completions,
		"code_tasks":         s.CodeTasks,
		"models_used":        s.ModelsUsed,
		"total_tokens":       s.TotalTokensUsed,
		"streaming_requests": s.StreamingRequests,
	}
}