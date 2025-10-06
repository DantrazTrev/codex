# Independent Codex CLI Parser with Enforcement

This is a completely independent parser for Codex CLI traffic that includes protobuf definitions, parsing, and enforcement capabilities. It requires no external dependencies beyond standard Go libraries.

## Features

### 1. **Independent Parsing** 
- No dependency on external proxy libraries
- Standalone request/response parsing
- Works with any HTTP proxy or middleware

### 2. **Traffic Types Supported**
- **Chat Completions** (`/v1/chat/completions`)
- **Code Edits** (`/v1/edits`)
- **Code Tasks** (`/api/codex`, `/code_tasks`)
- **Streaming Responses** (Server-Sent Events)
- **Tool/Function Calls**

### 3. **Enforcement Engine**
Built-in enforcement with multiple rule types:
- **PII Detection & Redaction** (SSN, phone, email)
- **Secret Detection** (API keys, passwords)
- **Code Security** (dangerous patterns, injection)
- **Model Restrictions** (allowed/blocked models)
- **Token Limits** (max token enforcement)
- **File Path Filtering** (system path blocking)
- **Content Filtering** (profanity, inappropriate content)

### 4. **Protobuf Definitions**
Complete protobuf schema in `codex.proto` for:
- Request/Response messages
- Chat completions
- Code edits  
- Enforcement policies
- Parsed traffic records

## Usage

### Basic Parsing

```go
// Create parser
parser := NewIndependentParser()

// Parse request
req, _ := http.NewRequest("POST", "https://api.openai.com/v1/chat/completions", body)
parsedReq, err := parser.ParseRequest(req, bodyBytes)

// Parse response  
parsedResp, err := parser.ParseResponse(resp, respBytes)

// Access extracted data
fmt.Printf("User said: %s\n", parsedReq.UserMessage)
fmt.Printf("AI replied: %s\n", parsedResp.AssistantMessage)
fmt.Printf("Tokens used: %d\n", parsedResp.TokenUsage.TotalTokens)
```

### With Enforcement

```go
// Set policy level
parser.enforcement.SetPolicy("strict") // or "moderate", "development"

// Parse with enforcement
parsedReq, _ := parser.ParseRequest(req, body)

// Check enforcement actions
for _, action := range parsedReq.Enforcements {
    switch action.Action {
    case "block":
        // Reject the request
        return fmt.Errorf("Blocked: %s", action.Reason)
    case "redact":
        // Content was redacted
        fmt.Printf("Redacted: %s\n", action.RuleName)
    case "warn":
        // Log warning
        log.Printf("Warning: %s", action.Reason)
    }
}
```

### Custom Rules

```go
// Add custom enforcement rule
customRule := EnforcementRule{
    ID:       "custom_block_gpt4",
    Name:     "Block GPT-4 in Production",
    Type:     "model",
    Pattern:  "gpt-4",
    Action:   "block",
    Message:  "GPT-4 not allowed in production",
    Fields:   []string{"model"},
    Priority: 10,
}
parser.enforcement.AddRule(customRule)
```

## Enforcement Examples

### PII Redaction
**Input:** "My SSN is 123-45-6789"
**Output:** "My SSN is [SSN REDACTED]"

### Secret Detection
**Input:** `api_key = "sk-proj-abc123xyz"`
**Output:** `api_key=[REDACTED]`

### Path Blocking
**Input:** File path `/etc/passwd`
**Action:** Block with message "Access to system paths not allowed"

### Token Limiting
**Input:** `max_tokens: 5000`
**Action:** Warn/Block if exceeds limit (default 4000)

## Extracted Data Structure

### ParsedRequest
```go
type ParsedRequest struct {
    ID           string      // Unique request ID
    Type         string      // "chat", "edit", "task"
    Model        string      // AI model requested
    UserMessage  string      // Extracted user message
    Code         string      // Code snippets found
    FilePaths    []string    // File paths mentioned
    Temperature  float64     // Temperature setting
    MaxTokens    int         // Max tokens requested
    Stream       bool        // Streaming enabled
    Tools        []string    // Tools/functions available
    Enforcements []Action    // Enforcement actions taken
}
```

### ParsedResponse
```go
type ParsedResponse struct {
    ID               string      // Unique response ID  
    Model            string      // Model that responded
    AssistantMessage string      // AI response text
    Code             string      // Generated code
    ToolCalls        []ToolCall  // Function calls made
    TokenUsage       *Usage      // Token consumption
    IsStreaming      bool        // Was streamed
    Enforcements     []Action    // Enforcement actions
}
```

## Policies

### Strict Security
- Blocks injection attacks
- Redacts all PII
- Blocks system path access
- Redacts secrets and passwords

### Moderate Security
- Logs sensitive operations
- Warns on dangerous code
- Monitors tool usage

### Development Mode
- Minimal blocking
- Mainly logging
- Allows most operations

## Integration Examples

### With HTTP Proxy

```go
func proxyHandler(w http.ResponseWriter, r *http.Request) {
    parser := NewIndependentParser()
    
    // Parse and enforce request
    body, _ := io.ReadAll(r.Body)
    parsed, _ := parser.ParseRequest(r, body)
    
    // Block if needed
    for _, e := range parsed.Enforcements {
        if e.Action == "block" {
            http.Error(w, e.Reason, 403)
            return
        }
    }
    
    // Forward to upstream...
    // Parse response...
    // Apply redactions...
}
```

### With Existing System

Since this parser is independent, integrate it anywhere:

```go
// Your existing code
requestData := getRequestSomehow()
responseData := getResponseSomehow()

// Add parsing
parser := NewIndependentParser()
parsedReq, _ := parser.ParseRequest(httpReq, requestData)
parsedResp, _ := parser.ParseResponse(httpResp, responseData)

// Use extracted data
logToYourSystem(parsedReq, parsedResp)
enforceYourPolicies(parsedReq.Enforcements)
```

## Statistics

The parser tracks:
- Total requests/responses
- Requests by type (chat/edit/task)
- Token usage
- Models used
- Blocked requests
- Redacted content
- Streaming vs non-streaming

Access via: `parser.stats`

## Files

- `codex.proto` - Protobuf definitions
- `independent_parser.go` - Main parser implementation
- `enforcement.go` - Enforcement engine
- `usage_example.go` - Complete examples

## No External Dependencies

This parser uses only Go standard library:
- `encoding/json` - JSON parsing
- `regexp` - Pattern matching
- `bufio` - Stream parsing
- `net/http` - HTTP types
- `time` - Timestamps

## Testing

```go
// Test chat parsing
chatReq := map[string]interface{}{
    "model": "gpt-4",
    "messages": []map[string]interface{}{
        {"role": "user", "content": "Hello"},
    },
}
body, _ := json.Marshal(chatReq)
parsed, _ := parser.ParseRequest(req, body)
assert(parsed.UserMessage == "Hello")
assert(parsed.Model == "gpt-4")
```

## Comparison with Claude Parser

| Feature | Claude Parser | Codex Parser |
|---------|--------------|--------------|
| Independence | Requires goproxy | Standalone |
| Enforcement | No | Yes, built-in |
| Protobuf | No | Yes |
| PII Detection | No | Yes |
| Secret Redaction | No | Yes |
| Model Restrictions | No | Yes |
| Token Limits | No | Yes |
| Statistics | Basic | Comprehensive |

## Performance

- Regex patterns are pre-compiled
- Streaming parsing is buffered
- Rules sorted by priority
- Early exit on blocking

## Security Features

1. **Input Validation** - All JSON parsed safely
2. **PII Protection** - Automatic redaction
3. **Secret Scanning** - API keys, passwords
4. **Injection Prevention** - SQL, XSS detection
5. **Path Traversal** - System path blocking
6. **Token Limits** - Prevent abuse
7. **Model Control** - Restrict expensive models

This independent parser gives you complete control over Codex CLI traffic with built-in security and enforcement.