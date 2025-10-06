package codex

// ExampleIntegrationWithYourProxy shows how to integrate this parser 
// with your existing proxy infrastructure that already has Claude parser

import (
	"log"
	"net/http"
	"strings"
	
	"github.com/okulis-ai/lumeus-application/gocore/pkg/goproxy"
	"github.com/okulis-ai/lumeus-application/gocore/pkg/params"
	// Assuming you have claude parser here
	// "github.com/okulis-ai/lumeus-application/gocore/pkg/parsers/claudecode"
)

// ProxyWithParsers shows how to use both Claude and Codex parsers
type ProxyWithParsers struct {
	codexClient  *CodexClient
	// claudeClient *claudecode.ClaudeCodeClient
}

// NewProxyWithParsers creates a new proxy with both parsers
func NewProxyWithParsers() *ProxyWithParsers {
	return &ProxyWithParsers{
		codexClient: NewCodexClient(),
		// claudeClient: claudecode.NewClaudeCodeClient(),
	}
}

// HandleRequest routes requests to appropriate parser
func (p *ProxyWithParsers) HandleRequest(ctx *goproxy.ProxyCtx, req *http.Request, reqBytes []byte, whiteListedAction string, saasAppCache *params.SaasAppCache) {
	host := req.Host
	path := req.URL.Path
	
	// Route to appropriate parser based on the request
	switch {
	case p.isCodexRequest(host, path):
		log.Printf("Routing to Codex parser: %s %s", req.Method, req.URL)
		p.codexClient.ParseRequest(ctx, req, reqBytes, whiteListedAction, saasAppCache)
		
	case p.isClaudeRequest(host, path):
		log.Printf("Routing to Claude parser: %s %s", req.Method, req.URL)
		// p.claudeClient.ParseRequest(ctx, req, reqBytes, whiteListedAction, saasAppCache)
		
	default:
		log.Printf("Unknown service, not parsing: %s %s", req.Method, req.URL)
		saasAppCache.BodyBytes = reqBytes
	}
}

// HandleResponse routes responses to appropriate parser
func (p *ProxyWithParsers) HandleResponse(ctx *goproxy.ProxyCtx, resp *http.Response, respBytes []byte, whiteListedAction string, saasAppCache *params.SaasAppCache) {
	// Determine which parser to use based on the original request
	// You might store this info in the context or cache
	
	reqHost := ctx.Req.Host
	reqPath := ctx.Req.URL.Path
	
	switch {
	case p.isCodexRequest(reqHost, reqPath):
		log.Printf("Parsing Codex response for: %s", ctx.Req.URL)
		p.codexClient.ParseResponse(ctx, resp, respBytes, whiteListedAction, saasAppCache)
		
	case p.isClaudeRequest(reqHost, reqPath):
		log.Printf("Parsing Claude response for: %s", ctx.Req.URL)
		// p.claudeClient.ParseResponse(ctx, resp, respBytes, whiteListedAction, saasAppCache)
		
	default:
		log.Printf("Unknown service response, not parsing")
		saasAppCache.BodyBytes = respBytes
	}
}

// isCodexRequest checks if this is a Codex/OpenAI request
func (p *ProxyWithParsers) isCodexRequest(host, path string) bool {
	// OpenAI API hosts
	openAIHosts := []string{
		"api.openai.com",
		"openai.com",
		"chat.openai.com",
		"chatgpt.com",
	}
	
	for _, h := range openAIHosts {
		if strings.Contains(host, h) {
			return true
		}
	}
	
	// Codex-specific paths (can be on any host if using proxy)
	codexPaths := []string{
		"/v1/chat/completions",
		"/v1/completions",
		"/v1/embeddings",
		"/v1/engines",
		"/api/codex",
		"/backend-api",
		"/wham",
	}
	
	for _, p := range codexPaths {
		if strings.Contains(path, p) {
			return true
		}
	}
	
	return false
}

// isClaudeRequest checks if this is a Claude/Anthropic request
func (p *ProxyWithParsers) isClaudeRequest(host, path string) bool {
	// Anthropic API hosts
	claudeHosts := []string{
		"api.anthropic.com",
		"anthropic.com",
		"claude.ai",
	}
	
	for _, h := range claudeHosts {
		if strings.Contains(host, h) {
			return true
		}
	}
	
	// Claude-specific paths
	claudePaths := []string{
		"/v1/messages",
		"/v1/complete",
		"/claude",
	}
	
	for _, p := range claudePaths {
		if strings.Contains(path, p) {
			return true
		}
	}
	
	return false
}

// Example of how to set up the proxy with parsers
func SetupProxyExample() {
	proxy := goproxy.NewProxyHttpServer()
	parsers := NewProxyWithParsers()
	
	// Handle requests
	proxy.OnRequest().DoFunc(
		func(r *http.Request, ctx *goproxy.ProxyCtx) (*http.Request, *http.Response) {
			// Read request body
			reqBytes := []byte{} // You'd actually read the body here
			
			// Create cache for this request
			cache := &params.SaasAppCache{
				Metadata: make(map[string]interface{}),
			}
			
			// Parse the request
			parsers.HandleRequest(ctx, r, reqBytes, "", cache)
			
			// Log what was extracted
			log.Printf("Extracted from request: %s", string(cache.BodyBytes))
			log.Printf("Access type: %s", cache.AccessType)
			
			return r, nil
		})
	
	// Handle responses
	proxy.OnResponse().DoFunc(
		func(resp *http.Response, ctx *goproxy.ProxyCtx) *http.Response {
			// Read response body
			respBytes := []byte{} // You'd actually read the body here
			
			// Get or create cache for this request
			cache := &params.SaasAppCache{
				Metadata: make(map[string]interface{}),
			}
			
			// Parse the response
			parsers.HandleResponse(ctx, resp, respBytes, "", cache)
			
			// Log what was extracted
			log.Printf("Extracted from response: %s", string(cache.BodyBytes))
			
			// Check token usage if available
			if cache.Metadata != nil {
				if tokens, ok := cache.Metadata["total_tokens"]; ok {
					log.Printf("Total tokens used: %v", tokens)
				}
			}
			
			return resp
		})
	
	// Start proxy
	log.Fatal(http.ListenAndServe(":8080", proxy))
}

// Unified statistics for both services
type UnifiedStats struct {
	CodexRequests  int64
	ClaudeRequests int64
	TotalTokens    int64
	ModelUsage     map[string]int64
}

// UpdateStatsFromCache updates statistics based on parsed cache
func (s *UnifiedStats) UpdateStatsFromCache(cache *params.SaasAppCache, service string) {
	switch service {
	case "codex":
		s.CodexRequests++
	case "claude":
		s.ClaudeRequests++
	}
	
	// Extract model from metadata
	if cache.Metadata != nil {
		if model, ok := cache.Metadata["model"].(string); ok {
			if s.ModelUsage == nil {
				s.ModelUsage = make(map[string]int64)
			}
			s.ModelUsage[model]++
		}
		
		// Extract token usage
		if tokens, ok := cache.Metadata["total_tokens"].(int); ok {
			s.TotalTokens += int64(tokens)
		}
	}
}