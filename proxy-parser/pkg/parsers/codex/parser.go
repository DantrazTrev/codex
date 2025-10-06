package codex

import (
	"bufio"
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"github.com/okulis-ai/lumeus-application/gocore/pkg/goproxy"
	"github.com/okulis-ai/lumeus-application/gocore/pkg/params"
	"github.com/okulis-ai/lumeus-application/gocore/pkg/proxy/utils"
)

type CodexClient struct{}

func NewCodexClient() *CodexClient {
	return &CodexClient{}
}

// OpenAI/Codex request structures
type Message struct {
	Role       string      `json:"role"`
	Content    interface{} `json:"content"` // Can be string or array of content blocks
	Name       string      `json:"name,omitempty"`
	ToolCalls  []ToolCall  `json:"tool_calls,omitempty"`
	ToolCallId string      `json:"tool_call_id,omitempty"`
}

type ContentBlock struct {
	Type     string    `json:"type"` // "text", "image_url"
	Text     string    `json:"text,omitempty"`
	ImageURL *ImageURL `json:"image_url,omitempty"`
}

type ImageURL struct {
	URL    string `json:"url"`
	Detail string `json:"detail,omitempty"`
}

type ToolCall struct {
	Id       string       `json:"id"`
	Type     string       `json:"type"`
	Function FunctionCall `json:"function"`
}

type FunctionCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

type CodexRequest struct {
	Model            string         `json:"model"`
	Messages         []Message      `json:"messages"`
	Temperature      float64        `json:"temperature,omitempty"`
	MaxTokens        int            `json:"max_tokens,omitempty"`
	Stream           bool           `json:"stream,omitempty"`
	Tools            []interface{}  `json:"tools,omitempty"`
	ToolChoice       interface{}    `json:"tool_choice,omitempty"`
	ResponseFormat   interface{}    `json:"response_format,omitempty"`
	FrequencyPenalty float64        `json:"frequency_penalty,omitempty"`
	PresencePenalty  float64        `json:"presence_penalty,omitempty"`
}

// Response structures for streaming
type StreamChunk struct {
	Id                string        `json:"id"`
	Object            string        `json:"object"`
	Created           int64         `json:"created"`
	Model             string        `json:"model"`
	SystemFingerprint string        `json:"system_fingerprint,omitempty"`
	Choices           []StreamChoice `json:"choices"`
}

type StreamChoice struct {
	Index        int          `json:"index"`
	Delta        DeltaContent `json:"delta"`
	Logprobs     interface{}  `json:"logprobs"`
	FinishReason *string      `json:"finish_reason"`
}

type DeltaContent struct {
	Role      string     `json:"role,omitempty"`
	Content   string     `json:"content,omitempty"`
	ToolCalls []ToolCall `json:"tool_calls,omitempty"`
}

// Non-streaming response
type CodexResponse struct {
	Id                string         `json:"id"`
	Object            string         `json:"object"`
	Created           int64          `json:"created"`
	Model             string         `json:"model"`
	SystemFingerprint string         `json:"system_fingerprint,omitempty"`
	Choices           []Choice       `json:"choices"`
	Usage             *Usage         `json:"usage,omitempty"`
}

type Choice struct {
	Index        int      `json:"index"`
	Message      Message  `json:"message"`
	Logprobs     interface{} `json:"logprobs"`
	FinishReason string   `json:"finish_reason"`
}

type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

func (c *CodexClient) ParseRequest(ctx *goproxy.ProxyCtx, req *http.Request, reqBytes []byte, whiteListedAction string, saasAppCache *params.SaasAppCache) {
	log.Printf("Parsing request for Codex: %s", req.URL.Path)

	// Detect endpoint type
	path := req.URL.Path
	
	// Handle different Codex endpoints
	if strings.Contains(path, "/chat/completions") || strings.Contains(path, "/v1/chat/completions") {
		c.parseChatCompletionRequest(reqBytes, saasAppCache)
	} else if strings.Contains(path, "/completions") || strings.Contains(path, "/v1/completions") {
		c.parseCompletionRequest(reqBytes, saasAppCache)
	} else if strings.Contains(path, "/api/codex") || strings.Contains(path, "/backend-api") || strings.Contains(path, "/wham") {
		// Handle Codex-specific backend endpoints
		c.parseCodexBackendRequest(reqBytes, path, saasAppCache)
	} else {
		log.Printf("Unknown endpoint type: %s", path)
		saasAppCache.BodyBytes = reqBytes
		saasAppCache.AccessType = utils.AccessType_UNKNOWN
	}
}

func (c *CodexClient) parseChatCompletionRequest(reqBytes []byte, saasAppCache *params.SaasAppCache) {
	var codexRequest CodexRequest
	if err := json.Unmarshal(reqBytes, &codexRequest); err != nil {
		log.Printf("Error unmarshalling Codex chat completion request: %v", err)
		return
	}

	log.Printf("Processing chat completion for model: %s", codexRequest.Model)
	
	// Extract the last user message
	if len(codexRequest.Messages) > 0 {
		lastMessage := codexRequest.Messages[len(codexRequest.Messages)-1]
		
		// Handle different content formats
		switch content := lastMessage.Content.(type) {
		case string:
			// Simple string content
			saasAppCache.BodyBytes = []byte(content)
			saasAppCache.AccessType = utils.AccessType_CHAT
		case []interface{}:
			// Array of content blocks (multimodal)
			var textContent strings.Builder
			for _, block := range content {
				if blockMap, ok := block.(map[string]interface{}); ok {
					if blockType, exists := blockMap["type"]; exists && blockType == "text" {
						if text, exists := blockMap["text"].(string); exists {
							textContent.WriteString(text)
							textContent.WriteString(" ")
						}
					}
				}
			}
			saasAppCache.BodyBytes = []byte(textContent.String())
			saasAppCache.AccessType = utils.AccessType_CHAT
		}
		
		// Check for tool calls
		if len(lastMessage.ToolCalls) > 0 {
			toolInfo, _ := json.Marshal(lastMessage.ToolCalls)
			saasAppCache.BodyBytes = toolInfo
			saasAppCache.AccessType = utils.AccessType_TOOL_CALL
		}
		
		saasAppCache.FirewallPolicyApplied = true
		
		// Store model information
		saasAppCache.Metadata = map[string]interface{}{
			"model":       codexRequest.Model,
			"stream":      codexRequest.Stream,
			"temperature": codexRequest.Temperature,
			"max_tokens":  codexRequest.MaxTokens,
		}
	}
}

func (c *CodexClient) parseCompletionRequest(reqBytes []byte, saasAppCache *params.SaasAppCache) {
	var request map[string]interface{}
	if err := json.Unmarshal(reqBytes, &request); err != nil {
		log.Printf("Error unmarshalling completion request: %v", err)
		return
	}
	
	// Extract prompt
	if prompt, exists := request["prompt"]; exists {
		switch p := prompt.(type) {
		case string:
			saasAppCache.BodyBytes = []byte(p)
		case []interface{}:
			prompts, _ := json.Marshal(p)
			saasAppCache.BodyBytes = prompts
		}
		saasAppCache.AccessType = utils.AccessType_COMPLETION
		saasAppCache.FirewallPolicyApplied = true
	}
}

func (c *CodexClient) parseCodexBackendRequest(reqBytes []byte, path string, saasAppCache *params.SaasAppCache) {
	// Parse generic JSON request
	var request map[string]interface{}
	if err := json.Unmarshal(reqBytes, &request); err != nil {
		log.Printf("Error unmarshalling Codex backend request: %v", err)
		saasAppCache.BodyBytes = reqBytes
		return
	}
	
	// Determine access type based on endpoint
	if strings.Contains(path, "/tasks") || strings.Contains(path, "/code_tasks") {
		saasAppCache.AccessType = utils.AccessType_CODE_TASK
		
		// Extract task content if present
		if taskContent, exists := request["content"]; exists {
			if content, ok := taskContent.(string); ok {
				saasAppCache.BodyBytes = []byte(content)
			}
		} else if message, exists := request["message"]; exists {
			if msg, ok := message.(string); ok {
				saasAppCache.BodyBytes = []byte(msg)
			}
		}
	} else if strings.Contains(path, "/conversation") {
		saasAppCache.AccessType = utils.AccessType_CONVERSATION
		
		// Extract conversation message
		if messages, exists := request["messages"]; exists {
			msgBytes, _ := json.Marshal(messages)
			saasAppCache.BodyBytes = msgBytes
		}
	} else {
		saasAppCache.AccessType = utils.AccessType_UNKNOWN
		saasAppCache.BodyBytes = reqBytes
	}
	
	saasAppCache.FirewallPolicyApplied = true
}

func (c *CodexClient) ParseResponse(ctx *goproxy.ProxyCtx, resp *http.Response, respBytes []byte, whiteListedAction string, saasAppCache *params.SaasAppCache) {
	contentType := resp.Header.Get("Content-Type")
	
	// Handle Server-Sent Events (streaming responses)
	if strings.Contains(contentType, "text/event-stream") {
		c.parseStreamingResponse(respBytes, saasAppCache)
	} else if strings.Contains(contentType, "application/json") {
		c.parseJSONResponse(respBytes, saasAppCache)
	} else {
		log.Printf("Unknown content type: %s", contentType)
		saasAppCache.BodyBytes = respBytes
	}
}

func (c *CodexClient) parseStreamingResponse(respBytes []byte, saasAppCache *params.SaasAppCache) {
	var responseText strings.Builder
	var toolCalls []ToolCall
	
	scanner := bufio.NewScanner(bytes.NewReader(respBytes))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		
		// Look for SSE data lines
		if strings.HasPrefix(line, "data:") {
			dataStr := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
			
			// Skip empty data or [DONE] marker
			if dataStr == "" || dataStr == "[DONE]" {
				continue
			}
			
			// Parse the chunk
			var chunk StreamChunk
			if err := json.Unmarshal([]byte(dataStr), &chunk); err != nil {
				log.Printf("Error unmarshalling stream chunk: %v", err)
				continue
			}
			
			// Extract content from choices
			for _, choice := range chunk.Choices {
				// Append text content
				if choice.Delta.Content != "" {
					responseText.WriteString(choice.Delta.Content)
				}
				
				// Collect tool calls
				if len(choice.Delta.ToolCalls) > 0 {
					toolCalls = append(toolCalls, choice.Delta.ToolCalls...)
				}
				
				// Log finish reason if present
				if choice.FinishReason != nil && *choice.FinishReason != "" {
					log.Printf("Stream finished with reason: %s", *choice.FinishReason)
				}
			}
		}
	}
	
	if err := scanner.Err(); err != nil {
		log.Printf("Error reading streaming response: %v", err)
		return
	}
	
	// Store extracted content
	extractedText := responseText.String()
	if extractedText != "" {
		log.Printf("Extracted streaming response text (first 200 chars): %.200s", extractedText)
		saasAppCache.BodyBytes = []byte(extractedText)
		saasAppCache.AccessType = utils.AccessType_CHAT_RESPONSE
	}
	
	// Store tool calls if any
	if len(toolCalls) > 0 {
		toolCallsJSON, _ := json.Marshal(toolCalls)
		saasAppCache.Metadata = map[string]interface{}{
			"tool_calls": string(toolCallsJSON),
		}
	}
}

func (c *CodexClient) parseJSONResponse(respBytes []byte, saasAppCache *params.SaasAppCache) {
	// Try to parse as chat completion response
	var response CodexResponse
	if err := json.Unmarshal(respBytes, &response); err != nil {
		// If it fails, treat as generic JSON
		log.Printf("Response is not a standard completion format")
		saasAppCache.BodyBytes = respBytes
		return
	}
	
	// Extract the assistant's message
	if len(response.Choices) > 0 {
		message := response.Choices[0].Message
		
		// Handle content
		switch content := message.Content.(type) {
		case string:
			saasAppCache.BodyBytes = []byte(content)
		default:
			// Marshal complex content
			contentBytes, _ := json.Marshal(content)
			saasAppCache.BodyBytes = contentBytes
		}
		
		saasAppCache.AccessType = utils.AccessType_CHAT_RESPONSE
		
		// Store usage information if available
		if response.Usage != nil {
			saasAppCache.Metadata = map[string]interface{}{
				"model":              response.Model,
				"prompt_tokens":      response.Usage.PromptTokens,
				"completion_tokens":  response.Usage.CompletionTokens,
				"total_tokens":       response.Usage.TotalTokens,
			}
			log.Printf("Token usage - Prompt: %d, Completion: %d, Total: %d",
				response.Usage.PromptTokens, 
				response.Usage.CompletionTokens,
				response.Usage.TotalTokens)
		}
	}
}

// Additional access types for Codex
const (
	AccessType_CODE_TASK     = "CODE_TASK"
	AccessType_CONVERSATION  = "CONVERSATION" 
	AccessType_TOOL_CALL     = "TOOL_CALL"
	AccessType_COMPLETION    = "COMPLETION"
	AccessType_CHAT_RESPONSE = "CHAT_RESPONSE"
	AccessType_UNKNOWN       = "UNKNOWN"
)