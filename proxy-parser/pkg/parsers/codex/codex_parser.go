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

// Request structures
type Message struct {
	Role       string      `json:"role"`
	Content    interface{} `json:"content"` // Can be string or array
	Name       string      `json:"name,omitempty"`
	ToolCalls  []ToolCall  `json:"tool_calls,omitempty"`
	ToolCallID string      `json:"tool_call_id,omitempty"`
}

type ToolCall struct {
	ID       string   `json:"id"`
	Type     string   `json:"type"`
	Function Function `json:"function"`
}

type Function struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

type ContentPart struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
}

type CodexRequest struct {
	Model       string    `json:"model"`
	Messages    []Message `json:"messages"`
	Temperature float64   `json:"temperature,omitempty"`
	MaxTokens   int       `json:"max_tokens,omitempty"`
	Stream      bool      `json:"stream,omitempty"`
}

type CodeEditRequest struct {
	Model       string  `json:"model"`
	Instruction string  `json:"instruction"`
	Input       string  `json:"input,omitempty"`
	Temperature float64 `json:"temperature,omitempty"`
}

// Response structures for streaming
type StreamData struct {
	ID      string         `json:"id"`
	Object  string         `json:"object"`
	Created int64          `json:"created"`
	Model   string         `json:"model"`
	Choices []StreamChoice `json:"choices"`
}

type StreamChoice struct {
	Index        int         `json:"index"`
	Delta        Delta       `json:"delta"`
	FinishReason *string     `json:"finish_reason"`
}

type Delta struct {
	Role      string     `json:"role,omitempty"`
	Content   string     `json:"content,omitempty"`
	ToolCalls []ToolCall `json:"tool_calls,omitempty"`
}

func (c *CodexClient) ParseRequest(ctx *goproxy.ProxyCtx, req *http.Request, reqBytes []byte, whiteListedAction string, saasAppCache *params.SaasAppCache) {
	path := req.URL.Path
	
	// Handle chat completions
	if strings.Contains(path, "/chat/completions") || strings.Contains(path, "/v1/chat/completions") {
		c.parseChatRequest(reqBytes, saasAppCache)
	} else if strings.Contains(path, "/edits") || strings.Contains(path, "/v1/edits") {
		c.parseEditRequest(reqBytes, saasAppCache)
	} else if strings.Contains(path, "/completions") || strings.Contains(path, "/v1/completions") {
		c.parseCompletionRequest(reqBytes, saasAppCache)
	} else {
		// Generic request, store raw
		saasAppCache.BodyBytes = reqBytes
		saasAppCache.AccessType = utils.AccessType_UNKNOWN
	}
}

func (c *CodexClient) parseChatRequest(reqBytes []byte, saasAppCache *params.SaasAppCache) {
	codexRequest := CodexRequest{}
	if err := json.Unmarshal(reqBytes, &codexRequest); err != nil {
		log.Printf("Error unmarshalling Codex chat request: %v", err)
		return
	}

	log.Printf("Processing chat request for model: %s", codexRequest.Model)
	
	// Extract the last user message
	if len(codexRequest.Messages) > 0 {
		lastMessage := codexRequest.Messages[len(codexRequest.Messages)-1]
		
		// Handle different content formats
		switch content := lastMessage.Content.(type) {
		case string:
			// Simple text content
			saasAppCache.BodyBytes = []byte(content)
			saasAppCache.AccessType = utils.AccessType_CHAT
			saasAppCache.FirewallPolicyApplied = true
			
		case []interface{}:
			// Array of content parts (multimodal)
			var textContent strings.Builder
			for _, part := range content {
				if partMap, ok := part.(map[string]interface{}); ok {
					if partMap["type"] == "text" {
						if text, ok := partMap["text"].(string); ok {
							textContent.WriteString(text)
						}
					}
				}
			}
			saasAppCache.BodyBytes = []byte(textContent.String())
			saasAppCache.AccessType = utils.AccessType_CHAT
			saasAppCache.FirewallPolicyApplied = true
		}
		
		// Check for tool calls
		if len(lastMessage.ToolCalls) > 0 {
			// Extract tool call info
			var toolInfo strings.Builder
			for _, tool := range lastMessage.ToolCalls {
				toolInfo.WriteString(tool.Function.Name)
				toolInfo.WriteString(": ")
				toolInfo.WriteString(tool.Function.Arguments)
				toolInfo.WriteString(" ")
			}
			saasAppCache.BodyBytes = []byte(toolInfo.String())
			saasAppCache.AccessType = utils.AccessType_TOOL_CALL
			saasAppCache.FirewallPolicyApplied = true
		}
		
		// Check for tool response
		if lastMessage.Role == "tool" || lastMessage.ToolCallID != "" {
			switch content := lastMessage.Content.(type) {
			case string:
				saasAppCache.BodyBytes = []byte(content)
				saasAppCache.AccessType = utils.AccessType_TOOL_RESULT
				saasAppCache.FirewallPolicyApplied = true
			}
		}
	}
}

func (c *CodexClient) parseEditRequest(reqBytes []byte, saasAppCache *params.SaasAppCache) {
	editRequest := CodeEditRequest{}
	if err := json.Unmarshal(reqBytes, &editRequest); err != nil {
		log.Printf("Error unmarshalling edit request: %v", err)
		return
	}

	log.Printf("Processing edit request for model: %s", editRequest.Model)
	
	// For edits, the instruction is the main content
	saasAppCache.BodyBytes = []byte(editRequest.Instruction)
	saasAppCache.AccessType = utils.AccessType_CODE_EDIT
	saasAppCache.FirewallPolicyApplied = true
	
	// If there's input code, append it
	if editRequest.Input != "" {
		combined := editRequest.Instruction + "\n\nCode:\n" + editRequest.Input
		saasAppCache.BodyBytes = []byte(combined)
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
			saasAppCache.AccessType = utils.AccessType_COMPLETION
			saasAppCache.FirewallPolicyApplied = true
		case []interface{}:
			// Multiple prompts
			var combined strings.Builder
			for _, item := range p {
				if str, ok := item.(string); ok {
					combined.WriteString(str)
					combined.WriteString(" ")
				}
			}
			saasAppCache.BodyBytes = []byte(combined.String())
			saasAppCache.AccessType = utils.AccessType_COMPLETION
			saasAppCache.FirewallPolicyApplied = true
		}
	}
}

func (c *CodexClient) ParseResponse(ctx *goproxy.ProxyCtx, resp *http.Response, respBytes []byte, whiteListedAction string, saasAppCache *params.SaasAppCache) {
	log.Printf("Parsing Codex response, Content-Type: %s", resp.Header.Get("Content-Type"))
	
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
	// Use a strings.Builder to efficiently build the response text
	var responseText strings.Builder
	var toolCalls []ToolCall

	// Create a scanner to read the response bytes
	scanner := bufio.NewScanner(bytes.NewReader(respBytes))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		
		// Look for lines starting with "data:"
		if strings.HasPrefix(line, "data:") {
			// Extract the JSON payload after "data:"
			dataStr := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
			
			// Skip empty data or [DONE] marker
			if dataStr == "" || dataStr == "[DONE]" {
				continue
			}

			// Unmarshal the data JSON
			var streamData StreamData
			err := json.Unmarshal([]byte(dataStr), &streamData)
			if err != nil {
				log.Printf("Error unmarshalling stream data: %v", err)
				continue
			}

			// Check for content in choices
			for _, choice := range streamData.Choices {
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

	// Check for scanner errors
	if err := scanner.Err(); err != nil {
		log.Printf("Error reading streaming response: %v", err)
		return
	}

	extractedText := responseText.String()
	if extractedText == "" && len(toolCalls) == 0 {
		log.Printf("No response text or tool calls extracted")
	} else {
		if extractedText != "" {
			log.Printf("Extracted response text: %s", extractedText)
			saasAppCache.BodyBytes = []byte(extractedText)
		}
		
		// If we have tool calls, handle them
		if len(toolCalls) > 0 {
			var toolInfo strings.Builder
			for _, tool := range toolCalls {
				toolInfo.WriteString(tool.Function.Name)
				toolInfo.WriteString(": ")
				toolInfo.WriteString(tool.Function.Arguments)
				toolInfo.WriteString(" ")
			}
			if extractedText == "" {
				// Only tool calls, no text
				saasAppCache.BodyBytes = []byte(toolInfo.String())
			} else {
				// Both text and tool calls
				combined := extractedText + "\n[Tools: " + toolInfo.String() + "]"
				saasAppCache.BodyBytes = []byte(combined)
			}
		}
	}
}

func (c *CodexClient) parseJSONResponse(respBytes []byte, saasAppCache *params.SaasAppCache) {
	var response map[string]interface{}
	if err := json.Unmarshal(respBytes, &response); err != nil {
		log.Printf("Error unmarshalling JSON response: %v", err)
		saasAppCache.BodyBytes = respBytes
		return
	}

	// Check if it's a chat completion response
	if choices, exists := response["choices"]; exists {
		if choicesArray, ok := choices.([]interface{}); ok && len(choicesArray) > 0 {
			if choice, ok := choicesArray[0].(map[string]interface{}); ok {
				// Extract message content
				if message, exists := choice["message"]; exists {
					if msg, ok := message.(map[string]interface{}); ok {
						// Extract content
						if content, exists := msg["content"]; exists {
							if contentStr, ok := content.(string); ok {
								saasAppCache.BodyBytes = []byte(contentStr)
								log.Printf("Extracted message content: %s", contentStr)
							}
						}
						
						// Extract tool calls
						if toolCalls, exists := msg["tool_calls"]; exists {
							if tools, ok := toolCalls.([]interface{}); ok && len(tools) > 0 {
								var toolInfo strings.Builder
								for _, tool := range tools {
									if t, ok := tool.(map[string]interface{}); ok {
										if function, exists := t["function"]; exists {
											if fn, ok := function.(map[string]interface{}); ok {
												if name, _ := fn["name"].(string); name != "" {
													toolInfo.WriteString(name)
													if args, _ := fn["arguments"].(string); args != "" {
														toolInfo.WriteString(": ")
														toolInfo.WriteString(args)
													}
													toolInfo.WriteString(" ")
												}
											}
										}
									}
								}
								if toolInfo.Len() > 0 {
									existing := string(saasAppCache.BodyBytes)
									if existing != "" {
										saasAppCache.BodyBytes = []byte(existing + "\n[Tools: " + toolInfo.String() + "]")
									} else {
										saasAppCache.BodyBytes = []byte(toolInfo.String())
									}
								}
							}
						}
					}
				}
				
				// For edit responses
				if text, exists := choice["text"]; exists {
					if textStr, ok := text.(string); ok {
						saasAppCache.BodyBytes = []byte(textStr)
						log.Printf("Extracted edit text: %s", textStr)
					}
				}
			}
		}
	}
	
	// For completion responses
	if text, exists := response["text"]; exists {
		if textStr, ok := text.(string); ok {
			saasAppCache.BodyBytes = []byte(textStr)
		}
	}
}

// Additional access types for Codex
const (
	AccessType_CHAT        = "CHAT"
	AccessType_TOOL_CALL   = "TOOL_CALL"
	AccessType_TOOL_RESULT = "TOOL_RESULT"
	AccessType_CODE_EDIT   = "CODE_EDIT"
	AccessType_COMPLETION  = "COMPLETION"
	AccessType_UNKNOWN     = "UNKNOWN"
)