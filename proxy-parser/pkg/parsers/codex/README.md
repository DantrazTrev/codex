# Codex CLI Parser

This parser is designed to intercept and parse traffic from the Codex CLI, similar to your Claude Code parser. It handles various OpenAI API formats and Codex-specific backend endpoints.

## Features

- **Chat Completion Parsing**: Handles `/v1/chat/completions` requests and responses
- **Streaming SSE Support**: Parses Server-Sent Events for streaming responses
- **Tool/Function Calls**: Detects and extracts tool calls from messages
- **Multiple Content Types**: Supports text, multimodal (images), and tool results
- **Token Usage Tracking**: Extracts token consumption from responses
- **Codex Backend Support**: Handles `/api/codex`, `/backend-api`, `/wham` endpoints

## Usage

### Basic Integration

```go
import "github.com/yourorg/proxy-parser/pkg/parsers/codex"

// Create parser instance
codexClient := codex.NewCodexClient()

// In your proxy request handler
func handleRequest(ctx *goproxy.ProxyCtx, req *http.Request, reqBytes []byte) {
    if isCodexRequest(req) {
        codexClient.ParseRequest(ctx, req, reqBytes, action, saasAppCache)
    }
}

// In your proxy response handler
func handleResponse(ctx *goproxy.ProxyCtx, resp *http.Response, respBytes []byte) {
    if isCodexResponse(resp) {
        codexClient.ParseResponse(ctx, resp, respBytes, action, saasAppCache)
    }
}
```

### Access Types

The parser sets different access types based on the content:

- `AccessType_CHAT` - Regular chat messages
- `AccessType_TOOL_CALL` - Function/tool invocations
- `AccessType_TOOL_RESULT` - Results from tool execution
- `AccessType_COMPLETION` - Text completions
- `AccessType_CODE_TASK` - Codex code task operations
- `AccessType_CONVERSATION` - Conversation management
- `AccessType_CHAT_RESPONSE` - Assistant responses

### Extracted Data

For requests, the parser extracts:
- User messages and prompts
- Model selection (gpt-4, gpt-3.5-turbo, etc.)
- Temperature, max_tokens, and other parameters
- Tool/function calls

For responses, the parser extracts:
- Assistant messages (from both streaming and non-streaming)
- Token usage statistics
- Tool call results
- Finish reasons

## Endpoint Detection

The parser recognizes these endpoints:

### OpenAI API Endpoints
- `/v1/chat/completions` - Chat completions
- `/v1/completions` - Text completions
- `/v1/embeddings` - Embeddings
- `/v1/images` - Image generation

### Codex-Specific Endpoints
- `/api/codex/*` - Codex API calls
- `/backend-api/*` - ChatGPT backend
- `/wham/*` - WHAM API endpoints
- `/code_tasks` - Code task management
- `/conversation` - Conversation handling

## Streaming Response Handling

The parser fully supports SSE (Server-Sent Events) streaming:

```go
// Automatically detects and parses streaming responses
// Extracts text chunks and combines them
// Handles [DONE] markers and finish reasons
```

## Examples

### Parsing a Chat Request

```go
request := CodexRequest{
    Model: "gpt-4",
    Messages: []Message{
        {Role: "user", Content: "Write a function"},
    },
    Stream: true,
}

// Parser extracts:
// - User message: "Write a function"
// - Model: "gpt-4"
// - Stream mode: true
// - Sets AccessType to CHAT
```

### Parsing a Streaming Response

```go
// SSE stream like:
// data: {"choices":[{"delta":{"content":"Here's"}}]}
// data: {"choices":[{"delta":{"content":" the"}}]}
// data: {"choices":[{"delta":{"content":" function"}}]}
// data: [DONE]

// Parser combines to: "Here's the function"
// Sets AccessType to CHAT_RESPONSE
```

## Differences from Claude Code Parser

| Feature | Claude Code Parser | Codex Parser |
|---------|-------------------|--------------|
| Request Format | Anthropic Messages API | OpenAI Chat/Completions API |
| Streaming | SSE with specific format | SSE with OpenAI format |
| Tool Calls | tool_result content type | tool_calls array |
| Models | claude-3, etc. | gpt-4, gpt-3.5-turbo, etc. |
| Endpoints | Claude API | OpenAI + Codex backend |

## Testing

Run the included tests:

```bash
go test ./pkg/parsers/codex
```

## Integration with Your Proxy

Since you already have the Claude parser integrated, adding Codex is similar:

```go
// In your main proxy handler
switch {
case isClaudeTraffic(req):
    claudeClient.ParseRequest(ctx, req, reqBytes, action, cache)
case isCodexTraffic(req):
    codexClient.ParseRequest(ctx, req, reqBytes, action, cache)
}
```

## Token Counting

The parser extracts token usage from responses:
- `prompt_tokens` - Tokens in the input
- `completion_tokens` - Tokens in the output  
- `total_tokens` - Combined total

This is stored in the metadata field of your cache.