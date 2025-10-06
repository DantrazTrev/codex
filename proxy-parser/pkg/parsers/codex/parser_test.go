package codex

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

func TestParseChatCompletionRequest(t *testing.T) {
	client := NewCodexClient()
	
	// Sample request similar to what Codex CLI sends
	request := CodexRequest{
		Model: "gpt-4",
		Messages: []Message{
			{
				Role:    "system",
				Content: "You are a helpful coding assistant.",
			},
			{
				Role:    "user",
				Content: "Write a Python function to calculate fibonacci",
			},
		},
		Temperature: 0.7,
		MaxTokens:   2000,
		Stream:      true,
	}
	
	reqBytes, _ := json.Marshal(request)
	
	// Create mock HTTP request
	req, _ := http.NewRequest("POST", "https://api.openai.com/v1/chat/completions", bytes.NewReader(reqBytes))
	
	// Mock cache object (you'd need to adapt this to your actual params.SaasAppCache)
	saasAppCache := &MockSaasAppCache{}
	
	// Parse the request
	client.ParseRequest(nil, req, reqBytes, "", saasAppCache)
	
	// Verify parsing
	if string(saasAppCache.BodyBytes) != "Write a Python function to calculate fibonacci" {
		t.Errorf("Expected user message to be extracted, got: %s", string(saasAppCache.BodyBytes))
	}
}

func TestParseStreamingResponse(t *testing.T) {
	client := NewCodexClient()
	
	// Sample SSE streaming response
	streamResponse := `data: {"id":"chatcmpl-123","object":"chat.completion.chunk","created":1234567890,"model":"gpt-4","choices":[{"index":0,"delta":{"role":"assistant"},"finish_reason":null}]}

data: {"id":"chatcmpl-123","object":"chat.completion.chunk","created":1234567890,"model":"gpt-4","choices":[{"index":0,"delta":{"content":"Here's"},"finish_reason":null}]}

data: {"id":"chatcmpl-123","object":"chat.completion.chunk","created":1234567890,"model":"gpt-4","choices":[{"index":0,"delta":{"content":" a Python"},"finish_reason":null}]}

data: {"id":"chatcmpl-123","object":"chat.completion.chunk","created":1234567890,"model":"gpt-4","choices":[{"index":0,"delta":{"content":" function"},"finish_reason":null}]}

data: {"id":"chatcmpl-123","object":"chat.completion.chunk","created":1234567890,"model":"gpt-4","choices":[{"index":0,"delta":{},"finish_reason":"stop"}]}

data: [DONE]`
	
	// Create mock response
	resp := &http.Response{
		Header: http.Header{
			"Content-Type": []string{"text/event-stream"},
		},
	}
	
	saasAppCache := &MockSaasAppCache{}
	
	// Parse the response
	client.ParseResponse(nil, resp, []byte(streamResponse), "", saasAppCache)
	
	// Verify extracted text
	expectedText := "Here's a Python function"
	if string(saasAppCache.BodyBytes) != expectedText {
		t.Errorf("Expected '%s', got '%s'", expectedText, string(saasAppCache.BodyBytes))
	}
}

func TestParseToolCallRequest(t *testing.T) {
	client := NewCodexClient()
	
	// Request with tool calls
	request := CodexRequest{
		Model: "gpt-4",
		Messages: []Message{
			{
				Role:    "user",
				Content: "What's the weather in San Francisco?",
			},
			{
				Role: "assistant",
				ToolCalls: []ToolCall{
					{
						Id:   "call_123",
						Type: "function",
						Function: FunctionCall{
							Name:      "get_weather",
							Arguments: `{"location": "San Francisco"}`,
						},
					},
				},
			},
		},
	}
	
	reqBytes, _ := json.Marshal(request)
	req, _ := http.NewRequest("POST", "https://api.openai.com/v1/chat/completions", bytes.NewReader(reqBytes))
	
	saasAppCache := &MockSaasAppCache{}
	client.ParseRequest(nil, req, reqBytes, "", saasAppCache)
	
	// Verify tool call was detected
	if saasAppCache.AccessType != AccessType_TOOL_CALL {
		t.Errorf("Expected TOOL_CALL access type, got: %s", saasAppCache.AccessType)
	}
}

// Mock implementation of SaasAppCache for testing
type MockSaasAppCache struct {
	BodyBytes             []byte
	AccessType            string
	FirewallPolicyApplied bool
	Metadata              map[string]interface{}
}

// Example of how to integrate this parser into your proxy
func ExampleUsage() {
	// In your proxy handler:
	/*
	codexClient := NewCodexClient()
	
	// In request handler
	if isCodexRequest(req) {
		codexClient.ParseRequest(ctx, req, reqBytes, action, cache)
	}
	
	// In response handler  
	if isCodexResponse(resp) {
		codexClient.ParseResponse(ctx, resp, respBytes, action, cache)
	}
	*/
}

// Helper to identify Codex requests
func isCodexRequest(req *http.Request) bool {
	host := req.Host
	path := req.URL.Path
	
	// Check for OpenAI API endpoints
	if host == "api.openai.com" || host == "openai.com" {
		return true
	}
	
	// Check for Codex-specific endpoints
	codexPaths := []string{
		"/v1/chat/completions",
		"/v1/completions", 
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