package codex

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strings"
	"time"
)

// IndependentParser is a standalone parser that doesn't depend on external packages
type IndependentParser struct {
	enforcement *EnforcementEngine
	stats       *ParserStats
}

// NewIndependentParser creates a new standalone parser
func NewIndependentParser() *IndependentParser {
	return &IndependentParser{
		enforcement: NewEnforcementEngine(),
		stats:       NewParserStats(),
	}
}

// ParsedRequest represents the extracted request data
type ParsedRequest struct {
	ID           string                 `json:"id"`
	Timestamp    time.Time              `json:"timestamp"`
	Type         string                 `json:"type"` // "chat", "edit", "task"
	Endpoint     string                 `json:"endpoint"`
	Model        string                 `json:"model"`
	UserMessage  string                 `json:"user_message"`
	Instruction  string                 `json:"instruction,omitempty"`
	Code         string                 `json:"code,omitempty"`
	FilePaths    []string               `json:"file_paths,omitempty"`
	Temperature  float64                `json:"temperature"`
	MaxTokens    int                    `json:"max_tokens"`
	Stream       bool                   `json:"stream"`
	Tools        []string               `json:"tools,omitempty"`
	RawData      map[string]interface{} `json:"raw_data"`
	Enforcements []EnforcementAction    `json:"enforcements,omitempty"`
}

// ParsedResponse represents the extracted response data
type ParsedResponse struct {
	ID                string              `json:"id"`
	Timestamp         time.Time           `json:"timestamp"`
	Type              string              `json:"type"`
	Model             string              `json:"model"`
	AssistantMessage  string              `json:"assistant_message"`
	Code              string              `json:"code,omitempty"`
	Edits             []FileEdit          `json:"edits,omitempty"`
	ToolCalls         []ExtractedToolCall `json:"tool_calls,omitempty"`
	TokenUsage        *TokenUsage         `json:"token_usage,omitempty"`
	FinishReason      string              `json:"finish_reason"`
	IsStreaming       bool                `json:"is_streaming"`
	RawData           interface{}         `json:"raw_data,omitempty"`
	Enforcements      []EnforcementAction `json:"enforcements,omitempty"`
}

// TokenUsage represents token consumption
type TokenUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// FileEdit represents a code edit
type FileEdit struct {
	Path      string `json:"path"`
	Operation string `json:"operation"` // "create", "modify", "delete"
	Content   string `json:"content"`
	LineStart int    `json:"line_start,omitempty"`
	LineEnd   int    `json:"line_end,omitempty"`
}

// ExtractedToolCall represents a tool/function call
type ExtractedToolCall struct {
	ID        string                 `json:"id"`
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments"`
}

// ParseRequest parses an HTTP request independently
func (p *IndependentParser) ParseRequest(req *http.Request, body []byte) (*ParsedRequest, error) {
	parsed := &ParsedRequest{
		ID:        generateID(),
		Timestamp: time.Now(),
		Endpoint:  req.URL.Path,
		RawData:   make(map[string]interface{}),
	}

	// Unmarshal body to detect type
	var rawRequest map[string]interface{}
	if err := json.Unmarshal(body, &rawRequest); err != nil {
		return nil, fmt.Errorf("invalid JSON: %w", err)
	}
	parsed.RawData = rawRequest

	// Detect and parse based on endpoint and content
	switch {
	case strings.Contains(req.URL.Path, "/chat/completions"):
		p.parseChatCompletionRequest(parsed, rawRequest)
	case strings.Contains(req.URL.Path, "/edits"):
		p.parseCodeEditRequest(parsed, rawRequest)
	case strings.Contains(req.URL.Path, "/code_tasks") || strings.Contains(req.URL.Path, "/api/codex"):
		p.parseCodeTaskRequest(parsed, rawRequest)
	default:
		parsed.Type = "unknown"
	}

	// Apply enforcement rules
	enforcements := p.enforcement.EnforceRequest(parsed)
	parsed.Enforcements = enforcements

	// Update statistics
	p.stats.RecordRequest(parsed)

	return parsed, nil
}

// parseChatCompletionRequest extracts chat completion data
func (p *IndependentParser) parseChatCompletionRequest(parsed *ParsedRequest, data map[string]interface{}) {
	parsed.Type = "chat"

	// Extract model
	if model, ok := data["model"].(string); ok {
		parsed.Model = model
	}

	// Extract parameters
	if temp, ok := data["temperature"].(float64); ok {
		parsed.Temperature = temp
	}
	if maxTokens, ok := data["max_tokens"].(float64); ok {
		parsed.MaxTokens = int(maxTokens)
	}
	if stream, ok := data["stream"].(bool); ok {
		parsed.Stream = stream
	}

	// Extract messages
	if messages, ok := data["messages"].([]interface{}); ok && len(messages) > 0 {
		// Get the last user message
		for i := len(messages) - 1; i >= 0; i-- {
			if msg, ok := messages[i].(map[string]interface{}); ok {
				role, _ := msg["role"].(string)
				if role == "user" {
					parsed.UserMessage = extractMessageContent(msg["content"])
					
					// Check for code in the message
					if code := extractCodeFromMessage(parsed.UserMessage); code != "" {
						parsed.Code = code
					}
					break
				}
			}
		}
	}

	// Extract tools
	if tools, ok := data["tools"].([]interface{}); ok {
		for _, tool := range tools {
			if t, ok := tool.(map[string]interface{}); ok {
				if fn, ok := t["function"].(map[string]interface{}); ok {
					if name, ok := fn["name"].(string); ok {
						parsed.Tools = append(parsed.Tools, name)
					}
				}
			}
		}
	}
}

// parseCodeEditRequest extracts code edit data
func (p *IndependentParser) parseCodeEditRequest(parsed *ParsedRequest, data map[string]interface{}) {
	parsed.Type = "edit"

	if model, ok := data["model"].(string); ok {
		parsed.Model = model
	}
	if instruction, ok := data["instruction"].(string); ok {
		parsed.Instruction = instruction
		parsed.UserMessage = instruction // For unified access
	}
	if input, ok := data["input"].(string); ok {
		parsed.Code = input
	}
	if temp, ok := data["temperature"].(float64); ok {
		parsed.Temperature = temp
	}
}

// parseCodeTaskRequest extracts code task data
func (p *IndependentParser) parseCodeTaskRequest(parsed *ParsedRequest, data map[string]interface{}) {
	parsed.Type = "task"

	if content, ok := data["content"].(string); ok {
		parsed.UserMessage = content
	}
	if title, ok := data["title"].(string); ok {
		if parsed.UserMessage == "" {
			parsed.UserMessage = title
		}
	}
	
	// Extract file paths
	if files, ok := data["file_paths"].([]interface{}); ok {
		for _, f := range files {
			if path, ok := f.(string); ok {
				parsed.FilePaths = append(parsed.FilePaths, path)
			}
		}
	}

	// Extract code snippets
	if code, ok := data["code"].(string); ok {
		parsed.Code = code
	}
}

// ParseResponse parses an HTTP response independently
func (p *IndependentParser) ParseResponse(resp *http.Response, body []byte) (*ParsedResponse, error) {
	parsed := &ParsedResponse{
		ID:        generateID(),
		Timestamp: time.Now(),
	}

	contentType := resp.Header.Get("Content-Type")

	// Handle different response types
	switch {
	case strings.Contains(contentType, "text/event-stream"):
		// SSE streaming response
		p.parseStreamingResponse(parsed, body)
	case strings.Contains(contentType, "application/json"):
		// JSON response
		p.parseJSONResponse(parsed, body)
	default:
		parsed.Type = "unknown"
		parsed.RawData = string(body)
	}

	// Apply enforcement rules
	enforcements := p.enforcement.EnforceResponse(parsed)
	parsed.Enforcements = enforcements

	// Update statistics
	p.stats.RecordResponse(parsed)

	return parsed, nil
}

// parseStreamingResponse handles SSE streaming responses
func (p *IndependentParser) parseStreamingResponse(parsed *ParsedResponse, body []byte) {
	parsed.Type = "streaming"
	parsed.IsStreaming = true

	var contentBuilder strings.Builder
	var toolCalls []ExtractedToolCall
	var finishReason string

	scanner := bufio.NewScanner(bytes.NewReader(body))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		
		if !strings.HasPrefix(line, "data:") {
			continue
		}

		dataStr := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if dataStr == "" || dataStr == "[DONE]" {
			continue
		}

		// Parse chunk
		var chunk map[string]interface{}
		if err := json.Unmarshal([]byte(dataStr), &chunk); err != nil {
			continue
		}

		// Extract model if present
		if model, ok := chunk["model"].(string); ok && parsed.Model == "" {
			parsed.Model = model
		}

		// Extract choices
		if choices, ok := chunk["choices"].([]interface{}); ok {
			for _, choice := range choices {
				if c, ok := choice.(map[string]interface{}); ok {
					// Extract content
					if delta, ok := c["delta"].(map[string]interface{}); ok {
						if content, ok := delta["content"].(string); ok {
							contentBuilder.WriteString(content)
						}
						
						// Extract tool calls
						if tools, ok := delta["tool_calls"].([]interface{}); ok {
							for _, tool := range tools {
								toolCall := extractToolCall(tool)
								if toolCall != nil {
									toolCalls = append(toolCalls, *toolCall)
								}
							}
						}
					}

					// Extract finish reason
					if fr, ok := c["finish_reason"].(string); ok && fr != "" {
						finishReason = fr
					}
				}
			}
		}
	}

	parsed.AssistantMessage = contentBuilder.String()
	parsed.ToolCalls = toolCalls
	parsed.FinishReason = finishReason
	
	// Extract code from response
	if code := extractCodeFromMessage(parsed.AssistantMessage); code != "" {
		parsed.Code = code
	}
}

// parseJSONResponse handles JSON responses
func (p *IndependentParser) parseJSONResponse(parsed *ParsedResponse, body []byte) {
	var response map[string]interface{}
	if err := json.Unmarshal(body, &response); err != nil {
		parsed.Type = "error"
		parsed.RawData = string(body)
		return
	}

	parsed.RawData = response

	// Detect response type
	if object, ok := response["object"].(string); ok {
		switch object {
		case "chat.completion":
			p.parseChatCompletionResponse(parsed, response)
		case "edit":
			p.parseEditResponse(parsed, response)
		default:
			parsed.Type = object
		}
	} else if _, ok := response["choices"]; ok {
		// Likely a completion response
		p.parseChatCompletionResponse(parsed, response)
	} else if _, ok := response["edits"]; ok {
		// Code task response
		p.parseCodeTaskResponse(parsed, response)
	}
}

// parseChatCompletionResponse extracts chat completion response data
func (p *IndependentParser) parseChatCompletionResponse(parsed *ParsedResponse, data map[string]interface{}) {
	parsed.Type = "chat_completion"

	if model, ok := data["model"].(string); ok {
		parsed.Model = model
	}

	// Extract choices
	if choices, ok := data["choices"].([]interface{}); ok && len(choices) > 0 {
		if choice, ok := choices[0].(map[string]interface{}); ok {
			// Extract message
			if message, ok := choice["message"].(map[string]interface{}); ok {
				parsed.AssistantMessage = extractMessageContent(message["content"])
				
				// Extract tool calls
				if tools, ok := message["tool_calls"].([]interface{}); ok {
					for _, tool := range tools {
						if tc := extractToolCall(tool); tc != nil {
							parsed.ToolCalls = append(parsed.ToolCalls, *tc)
						}
					}
				}
			}

			// Extract finish reason
			if fr, ok := choice["finish_reason"].(string); ok {
				parsed.FinishReason = fr
			}
		}
	}

	// Extract usage
	if usage, ok := data["usage"].(map[string]interface{}); ok {
		parsed.TokenUsage = &TokenUsage{
			PromptTokens:     int(getFloat(usage, "prompt_tokens")),
			CompletionTokens: int(getFloat(usage, "completion_tokens")),
			TotalTokens:      int(getFloat(usage, "total_tokens")),
		}
	}

	// Extract code
	if code := extractCodeFromMessage(parsed.AssistantMessage); code != "" {
		parsed.Code = code
	}
}

// parseEditResponse extracts edit response data
func (p *IndependentParser) parseEditResponse(parsed *ParsedResponse, data map[string]interface{}) {
	parsed.Type = "edit"

	if choices, ok := data["choices"].([]interface{}); ok && len(choices) > 0 {
		if choice, ok := choices[0].(map[string]interface{}); ok {
			if text, ok := choice["text"].(string); ok {
				parsed.Code = text
				parsed.AssistantMessage = "Code edit completed"
			}
		}
	}

	// Extract usage
	if usage, ok := data["usage"].(map[string]interface{}); ok {
		parsed.TokenUsage = &TokenUsage{
			PromptTokens:     int(getFloat(usage, "prompt_tokens")),
			CompletionTokens: int(getFloat(usage, "completion_tokens")),
			TotalTokens:      int(getFloat(usage, "total_tokens")),
		}
	}
}

// parseCodeTaskResponse extracts code task response data
func (p *IndependentParser) parseCodeTaskResponse(parsed *ParsedResponse, data map[string]interface{}) {
	parsed.Type = "task"

	if result, ok := data["result"].(map[string]interface{}); ok {
		if code, ok := result["code"].(string); ok {
			parsed.Code = code
		}
		if explanation, ok := result["explanation"].(string); ok {
			parsed.AssistantMessage = explanation
		}

		// Extract file edits
		if edits, ok := result["edits"].([]interface{}); ok {
			for _, edit := range edits {
				if e, ok := edit.(map[string]interface{}); ok {
					fileEdit := FileEdit{
						Path:      getString(e, "path"),
						Operation: getString(e, "operation"),
						Content:   getString(e, "content"),
					}
					parsed.Edits = append(parsed.Edits, fileEdit)
				}
			}
		}
	}
}

// Helper functions

func extractMessageContent(content interface{}) string {
	switch c := content.(type) {
	case string:
		return c
	case []interface{}:
		var texts []string
		for _, item := range c {
			if m, ok := item.(map[string]interface{}); ok {
				if m["type"] == "text" {
					if text, ok := m["text"].(string); ok {
						texts = append(texts, text)
					}
				}
			}
		}
		return strings.Join(texts, " ")
	default:
		return ""
	}
}

func extractCodeFromMessage(message string) string {
	// Extract code blocks from markdown
	codeBlockRegex := regexp.MustCompile("```(?:\\w+)?\\n([\\s\\S]*?)```")
	matches := codeBlockRegex.FindStringSubmatch(message)
	if len(matches) > 1 {
		return matches[1]
	}
	return ""
}

func extractToolCall(tool interface{}) *ExtractedToolCall {
	if t, ok := tool.(map[string]interface{}); ok {
		tc := &ExtractedToolCall{
			ID: getString(t, "id"),
		}
		
		if fn, ok := t["function"].(map[string]interface{}); ok {
			tc.Name = getString(fn, "name")
			if args, ok := fn["arguments"].(string); ok {
				var argMap map[string]interface{}
				if err := json.Unmarshal([]byte(args), &argMap); err == nil {
					tc.Arguments = argMap
				}
			}
		}
		
		if tc.Name != "" {
			return tc
		}
	}
	return nil
}

func getString(m map[string]interface{}, key string) string {
	if val, ok := m[key].(string); ok {
		return val
	}
	return ""
}

func getFloat(m map[string]interface{}, key string) float64 {
	if val, ok := m[key].(float64); ok {
		return val
	}
	return 0
}

func generateID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}

// ParserStats tracks parsing statistics
type ParserStats struct {
	TotalRequests    int64
	TotalResponses   int64
	ChatRequests     int64
	EditRequests     int64
	TaskRequests     int64
	StreamingCount   int64
	TotalTokens      int64
	BlockedRequests  int64
	RedactedContent  int64
	ModelUsage       map[string]int64
}

// NewParserStats creates new statistics tracker
func NewParserStats() *ParserStats {
	return &ParserStats{
		ModelUsage: make(map[string]int64),
	}
}

// RecordRequest updates request statistics
func (s *ParserStats) RecordRequest(req *ParsedRequest) {
	s.TotalRequests++
	
	switch req.Type {
	case "chat":
		s.ChatRequests++
	case "edit":
		s.EditRequests++
	case "task":
		s.TaskRequests++
	}
	
	if req.Model != "" {
		s.ModelUsage[req.Model]++
	}
	
	for _, e := range req.Enforcements {
		if e.Action == "block" {
			s.BlockedRequests++
		} else if e.Action == "redact" {
			s.RedactedContent++
		}
	}
}

// RecordResponse updates response statistics
func (s *ParserStats) RecordResponse(resp *ParsedResponse) {
	s.TotalResponses++
	
	if resp.IsStreaming {
		s.StreamingCount++
	}
	
	if resp.TokenUsage != nil {
		s.TotalTokens += int64(resp.TokenUsage.TotalTokens)
	}
	
	for _, e := range resp.Enforcements {
		if e.Action == "redact" {
			s.RedactedContent++
		}
	}
}