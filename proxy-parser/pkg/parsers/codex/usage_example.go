package codex

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
)

// ExampleUsage demonstrates how to use the independent Codex parser
func ExampleUsage() {
	// Create parser instance
	parser := NewIndependentParser()
	
	// Set enforcement policy (strict, moderate, or development)
	parser.enforcement.SetPolicy("moderate")
	
	// Add custom rules if needed
	customRule := EnforcementRule{
		ID:       "custom_1",
		Name:     "Block Production Secrets",
		Type:     "code",
		Pattern:  `PROD_SECRET_[\w]+`,
		Action:   "block",
		Message:  "Production secrets not allowed",
		Fields:   []string{"code", "content"},
		Priority: 10,
	}
	parser.enforcement.AddRule(customRule)
	
	// Example: Parse a chat completion request
	chatRequest := map[string]interface{}{
		"model": "gpt-4",
		"messages": []map[string]interface{}{
			{
				"role":    "user",
				"content": "Write a Python function to connect to database with password='secret123'",
			},
		},
		"temperature": 0.7,
		"max_tokens":  2000,
		"stream":      true,
	}
	
	reqBody, _ := json.Marshal(chatRequest)
	req, _ := http.NewRequest("POST", "https://api.openai.com/v1/chat/completions", bytes.NewReader(reqBody))
	
	// Parse the request
	parsedReq, err := parser.ParseRequest(req, reqBody)
	if err != nil {
		log.Printf("Error parsing request: %v", err)
		return
	}
	
	// Check enforcements
	for _, enforcement := range parsedReq.Enforcements {
		switch enforcement.Action {
		case "block":
			fmt.Printf("🚫 BLOCKED: %s - %s\n", enforcement.RuleName, enforcement.Reason)
			// Block the request
			return
		case "redact":
			fmt.Printf("✂️ REDACTED: %s - %s\n", enforcement.RuleName, enforcement.Reason)
			fmt.Printf("   Original: %s\n", enforcement.OriginalContent)
			fmt.Printf("   Modified: %s\n", enforcement.ModifiedContent)
		case "warn":
			fmt.Printf("⚠️ WARNING: %s - %s\n", enforcement.RuleName, enforcement.Reason)
		case "log":
			fmt.Printf("📝 LOGGED: %s - %s\n", enforcement.RuleName, enforcement.Reason)
		}
	}
	
	// Display parsed information
	fmt.Printf("\n📊 Parsed Request:\n")
	fmt.Printf("  Type: %s\n", parsedReq.Type)
	fmt.Printf("  Model: %s\n", parsedReq.Model)
	fmt.Printf("  User Message: %s\n", parsedReq.UserMessage)
	fmt.Printf("  Stream: %v\n", parsedReq.Stream)
	fmt.Printf("  Max Tokens: %d\n", parsedReq.MaxTokens)
	
	// Example: Parse a streaming response
	streamResponse := `data: {"id":"chatcmpl-123","object":"chat.completion.chunk","created":1234567890,"model":"gpt-4","choices":[{"index":0,"delta":{"role":"assistant"},"finish_reason":null}]}

data: {"id":"chatcmpl-123","object":"chat.completion.chunk","created":1234567890,"model":"gpt-4","choices":[{"index":0,"delta":{"content":"Here's a Python"},"finish_reason":null}]}

data: {"id":"chatcmpl-123","object":"chat.completion.chunk","created":1234567890,"model":"gpt-4","choices":[{"index":0,"delta":{"content":" function:\n\n```python\ndef connect"},"finish_reason":null}]}

data: {"id":"chatcmpl-123","object":"chat.completion.chunk","created":1234567890,"model":"gpt-4","choices":[{"index":0,"delta":{"content":"_db():\n    password = 'secret123'\n    # connection code\n```"},"finish_reason":null}]}

data: {"id":"chatcmpl-123","object":"chat.completion.chunk","created":1234567890,"model":"gpt-4","choices":[{"index":0,"delta":{},"finish_reason":"stop"}]}

data: [DONE]`
	
	resp := &http.Response{
		Header: http.Header{
			"Content-Type": []string{"text/event-stream"},
		},
	}
	
	// Parse the response
	parsedResp, err := parser.ParseResponse(resp, []byte(streamResponse))
	if err != nil {
		log.Printf("Error parsing response: %v", err)
		return
	}
	
	// Check response enforcements
	for _, enforcement := range parsedResp.Enforcements {
		if enforcement.Action == "redact" {
			fmt.Printf("\n✂️ Response REDACTED: %s\n", enforcement.RuleName)
		}
	}
	
	fmt.Printf("\n📊 Parsed Response:\n")
	fmt.Printf("  Type: %s\n", parsedResp.Type)
	fmt.Printf("  Model: %s\n", parsedResp.Model)
	fmt.Printf("  Assistant Message: %s\n", parsedResp.AssistantMessage)
	fmt.Printf("  Code Extracted: %s\n", parsedResp.Code)
	fmt.Printf("  Is Streaming: %v\n", parsedResp.IsStreaming)
	fmt.Printf("  Finish Reason: %s\n", parsedResp.FinishReason)
	
	// Get statistics
	fmt.Printf("\n📈 Parser Statistics:\n")
	fmt.Printf("  Total Requests: %d\n", parser.stats.TotalRequests)
	fmt.Printf("  Chat Requests: %d\n", parser.stats.ChatRequests)
	fmt.Printf("  Blocked Requests: %d\n", parser.stats.BlockedRequests)
	fmt.Printf("  Redacted Content: %d\n", parser.stats.RedactedContent)
}

// ProxyIntegration shows how to integrate with a proxy server
func ProxyIntegration() {
	parser := NewIndependentParser()
	
	// Create a proxy handler
	proxyHandler := func(w http.ResponseWriter, r *http.Request) {
		// Read request body
		reqBody, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Failed to read request", http.StatusBadRequest)
			return
		}
		r.Body = io.NopCloser(bytes.NewReader(reqBody)) // Restore body
		
		// Parse request
		parsedReq, err := parser.ParseRequest(r, reqBody)
		if err != nil {
			log.Printf("Parse error: %v", err)
			// Continue anyway
		}
		
		// Check for blocking rules
		blocked := false
		for _, enforcement := range parsedReq.Enforcements {
			if enforcement.Action == "block" {
				blocked = true
				http.Error(w, fmt.Sprintf("Request blocked: %s", enforcement.Reason), http.StatusForbidden)
				log.Printf("BLOCKED request to %s: %s", r.URL.Path, enforcement.Reason)
				break
			}
		}
		
		if blocked {
			return
		}
		
		// Log parsed information
		log.Printf("REQUEST [%s] Model: %s, Type: %s, Tokens: %d",
			parsedReq.Type, parsedReq.Model, parsedReq.Endpoint, parsedReq.MaxTokens)
		
		// Forward request to upstream (simplified)
		// In real implementation, you'd forward to actual backend
		upstreamResp := &http.Response{
			StatusCode: 200,
			Header: http.Header{
				"Content-Type": []string{"application/json"},
			},
		}
		
		// Example response
		respData := map[string]interface{}{
			"id":      "chatcmpl-123",
			"object":  "chat.completion",
			"model":   "gpt-4",
			"choices": []map[string]interface{}{
				{
					"message": map[string]interface{}{
						"role":    "assistant",
						"content": "Here's the code you requested...",
					},
					"finish_reason": "stop",
				},
			},
			"usage": map[string]interface{}{
				"prompt_tokens":     100,
				"completion_tokens": 200,
				"total_tokens":      300,
			},
		}
		
		respBody, _ := json.Marshal(respData)
		
		// Parse response
		parsedResp, err := parser.ParseResponse(upstreamResp, respBody)
		if err != nil {
			log.Printf("Response parse error: %v", err)
		}
		
		// Apply response modifications if needed
		if len(parsedResp.Enforcements) > 0 {
			// Modify response based on enforcements
			for _, enforcement := range parsedResp.Enforcements {
				if enforcement.Action == "redact" {
					// Response was redacted, use modified content
					respData["choices"].([]map[string]interface{})[0]["message"].(map[string]interface{})["content"] = parsedResp.AssistantMessage
					respBody, _ = json.Marshal(respData)
				}
			}
		}
		
		// Log response information
		log.Printf("RESPONSE [%s] Tokens: %d, Finish: %s",
			parsedResp.Type,
			parsedResp.TokenUsage.TotalTokens,
			parsedResp.FinishReason)
		
		// Send response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		w.Write(respBody)
	}
	
	// Start proxy server
	http.HandleFunc("/", proxyHandler)
	log.Println("Proxy server starting on :8080...")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

// SaveParsedData shows how to save parsed data for analysis
func SaveParsedData(parsedReq *ParsedRequest, parsedResp *ParsedResponse) {
	// Create a combined record
	record := map[string]interface{}{
		"request":  parsedReq,
		"response": parsedResp,
		"metadata": map[string]interface{}{
			"timestamp": time.Now().Unix(),
			"session_id": generateID(),
			"enforcements_triggered": len(parsedReq.Enforcements) + len(parsedResp.Enforcements),
		},
	}
	
	// Save to file or database
	data, _ := json.MarshalIndent(record, "", "  ")
	fmt.Println(string(data))
	
	// You could save to file:
	// os.WriteFile("parsed_traffic.json", data, 0644)
	
	// Or send to database/monitoring system
}