package parser

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/openai/codex/proxy-parser/pkg/storage"
	"go.uber.org/zap"
)

// Analyzer analyzes captured traffic
type Analyzer struct {
	logger *zap.Logger
}

// NewAnalyzer creates a new analyzer
func NewAnalyzer(logger *zap.Logger) *Analyzer {
	return &Analyzer{
		logger: logger,
	}
}

// AnalyzeOptions defines analysis options
type AnalyzeOptions struct {
	GenAIOnly bool
	Verbose   bool
	Filter    string
}

// AnalyzeFile analyzes traffic from a file
func (a *Analyzer) AnalyzeFile(filepath string, opts *AnalyzeOptions) error {
	stor := &storage.Storage{}
	records, err := stor.LoadFromFile(filepath)
	if err != nil {
		return fmt.Errorf("failed to load file: %w", err)
	}

	fmt.Printf("\n📊 Traffic Analysis Report\n")
	fmt.Printf("========================\n\n")
	fmt.Printf("Total records: %d\n\n", len(records))

	// Apply filters
	if opts.GenAIOnly {
		filtered := []storage.TrafficRecord{}
		for _, r := range records {
			for _, tag := range r.Tags {
				if tag == "genai" || tag == "genai-response" {
					filtered = append(filtered, r)
					break
				}
			}
		}
		records = filtered
		fmt.Printf("GenAI records: %d\n\n", len(records))
	}

	// Group by session
	sessions := a.groupBySessions(records)
	
	// Display sessions
	for sessionID, sessionRecords := range sessions {
		a.displaySession(sessionID, sessionRecords, opts)
	}

	// Display summary
	a.displaySummary(records)

	return nil
}

// groupBySessions groups records by session ID
func (a *Analyzer) groupBySessions(records []storage.TrafficRecord) map[string][]storage.TrafficRecord {
	sessions := make(map[string][]storage.TrafficRecord)
	
	for _, record := range records {
		sessions[record.ID] = append(sessions[record.ID], record)
	}
	
	return sessions
}

// displaySession displays a single session
func (a *Analyzer) displaySession(sessionID string, records []storage.TrafficRecord, opts *AnalyzeOptions) {
	if len(records) == 0 {
		return
	}

	// Find request and response
	var request, response *storage.TrafficRecord
	for i := range records {
		if records[i].Type == "request" {
			request = &records[i]
		} else if records[i].Type == "response" {
			response = &records[i]
		}
	}

	if request == nil {
		return
	}

	// Determine if this is GenAI traffic
	isGenAI := false
	var modelName string
	for _, tag := range request.Tags {
		if tag == "genai" {
			isGenAI = true
		}
		if strings.HasPrefix(tag, "model:") {
			modelName = strings.TrimPrefix(tag, "model:")
		}
	}

	// Format output with colors
	if isGenAI {
		color.Green("🤖 GenAI Request [Session: %s]", sessionID)
		if modelName != "" {
			color.Yellow("   Model: %s", modelName)
		}
	} else {
		fmt.Printf("📡 Request [Session: %s]\n", sessionID)
	}

	fmt.Printf("   %s %s\n", request.Method, request.URL)
	fmt.Printf("   Time: %s\n", request.Timestamp.Format(time.RFC3339))

	if opts.Verbose && request.Body != "" {
		a.displayBody("Request", request.Body, isGenAI)
	}

	if response != nil {
		if response.StatusCode >= 200 && response.StatusCode < 300 {
			color.Green("   ✓ Response: %d", response.StatusCode)
		} else if response.StatusCode >= 400 {
			color.Red("   ✗ Response: %d", response.StatusCode)
		} else {
			fmt.Printf("   Response: %d\n", response.StatusCode)
		}

		if opts.Verbose && response.Body != "" {
			a.displayBody("Response", response.Body, isGenAI)
		}
	}

	fmt.Println()
}

// displayBody displays request/response body with formatting
func (a *Analyzer) displayBody(label string, body string, isGenAI bool) {
	fmt.Printf("\n   %s Body:\n", label)
	
	// Try to parse as JSON for better formatting
	var jsonData interface{}
	if err := json.Unmarshal([]byte(body), &jsonData); err == nil {
		// Successfully parsed JSON
		if isGenAI {
			a.highlightGenAIContent(jsonData)
		} else {
			prettyJSON, _ := json.MarshalIndent(jsonData, "   ", "  ")
			fmt.Printf("%s\n", string(prettyJSON))
		}
	} else {
		// Not JSON or failed to parse
		lines := strings.Split(body, "\n")
		for _, line := range lines {
			if len(line) > 100 {
				fmt.Printf("   %s...\n", line[:100])
			} else {
				fmt.Printf("   %s\n", line)
			}
			if len(lines) > 10 {
				fmt.Printf("   ... (%d more lines)\n", len(lines)-10)
				break
			}
		}
	}
}

// highlightGenAIContent highlights important GenAI fields
func (a *Analyzer) highlightGenAIContent(data interface{}) {
	dataMap, ok := data.(map[string]interface{})
	if !ok {
		prettyJSON, _ := json.MarshalIndent(data, "   ", "  ")
		fmt.Printf("%s\n", string(prettyJSON))
		return
	}

	// Highlight important fields
	importantFields := []string{"model", "messages", "temperature", "max_tokens", "stream", "choices", "usage"}
	
	for _, field := range importantFields {
		if value, exists := dataMap[field]; exists {
			switch field {
			case "model":
				color.Yellow("   %s: %v", field, value)
			case "messages":
				if messages, ok := value.([]interface{}); ok {
					color.Cyan("   messages: [%d messages]", len(messages))
					for i, msg := range messages {
						if msgMap, ok := msg.(map[string]interface{}); ok {
							role := msgMap["role"]
							content := msgMap["content"]
							if contentStr, ok := content.(string); ok && len(contentStr) > 50 {
								content = contentStr[:50] + "..."
							}
							fmt.Printf("     [%d] role: %v, content: %v\n", i, role, content)
						}
					}
				}
			case "choices":
				if choices, ok := value.([]interface{}); ok {
					color.Green("   choices: [%d choices]", len(choices))
				}
			case "usage":
				color.Magenta("   usage: %v", value)
			default:
				fmt.Printf("   %s: %v\n", field, value)
			}
		}
	}

	// Show other fields
	for field, value := range dataMap {
		found := false
		for _, important := range importantFields {
			if field == important {
				found = true
				break
			}
		}
		if !found {
			fmt.Printf("   %s: %v\n", field, value)
		}
	}
}

// displaySummary displays analysis summary
func (a *Analyzer) displaySummary(records []storage.TrafficRecord) {
	fmt.Printf("\n📈 Summary Statistics\n")
	fmt.Printf("===================\n\n")

	// Count statistics
	requests := 0
	responses := 0
	genaiRequests := 0
	endpoints := make(map[string]int)
	models := make(map[string]int)
	statusCodes := make(map[int]int)

	for _, record := range records {
		if record.Type == "request" {
			requests++
			if record.URL != "" {
				// Simplify URL for grouping
				endpoint := extractPath(record.URL)
				endpoints[endpoint]++
			}
		} else if record.Type == "response" {
			responses++
			if record.StatusCode > 0 {
				statusCodes[record.StatusCode]++
			}
		}

		for _, tag := range record.Tags {
			if tag == "genai" {
				genaiRequests++
			}
			if strings.HasPrefix(tag, "model:") {
				model := strings.TrimPrefix(tag, "model:")
				models[model]++
			}
		}
	}

	fmt.Printf("Total Requests:  %d\n", requests)
	fmt.Printf("Total Responses: %d\n", responses)
	fmt.Printf("GenAI Requests:  %d (%.1f%%)\n", genaiRequests, float64(genaiRequests)/float64(requests)*100)
	
	if len(models) > 0 {
		fmt.Printf("\n🤖 AI Models Used:\n")
		for model, count := range models {
			fmt.Printf("   • %s: %d calls\n", model, count)
		}
	}

	if len(endpoints) > 0 {
		fmt.Printf("\n🌐 Top Endpoints:\n")
		top := getTopEndpoints(endpoints, 5)
		for _, ep := range top {
			fmt.Printf("   • %s: %d calls\n", ep.endpoint, ep.count)
		}
	}

	if len(statusCodes) > 0 {
		fmt.Printf("\n📊 Response Status Codes:\n")
		for code, count := range statusCodes {
			if code >= 200 && code < 300 {
				color.Green("   • %d: %d responses", code, count)
			} else if code >= 400 {
				color.Red("   • %d: %d responses", code, count)
			} else {
				fmt.Printf("   • %d: %d responses\n", code, count)
			}
		}
	}
}

// GenerateStats generates detailed statistics
func (a *Analyzer) GenerateStats(filepath string) error {
	stor := &storage.Storage{}
	records, err := stor.LoadFromFile(filepath)
	if err != nil {
		return fmt.Errorf("failed to load file: %w", err)
	}

	stats := a.calculateStatistics(records)
	
	// Output as JSON
	statsJSON, err := json.MarshalIndent(stats, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal stats: %w", err)
	}

	fmt.Println(string(statsJSON))
	return nil
}

// calculateStatistics calculates detailed statistics
func (a *Analyzer) calculateStatistics(records []storage.TrafficRecord) map[string]interface{} {
	stats := make(map[string]interface{})
	
	// Time-based analysis
	if len(records) > 0 {
		firstTime := records[0].Timestamp
		lastTime := records[len(records)-1].Timestamp
		duration := lastTime.Sub(firstTime)
		
		stats["time_range"] = map[string]interface{}{
			"start":    firstTime,
			"end":      lastTime,
			"duration": duration.String(),
		}
	}

	// Request/Response counts
	requests := 0
	responses := 0
	for _, r := range records {
		if r.Type == "request" {
			requests++
		} else if r.Type == "response" {
			responses++
		}
	}
	
	stats["counts"] = map[string]interface{}{
		"total_records": len(records),
		"requests":      requests,
		"responses":     responses,
	}

	// GenAI analysis
	genaiStats := a.analyzeGenAITraffic(records)
	if len(genaiStats) > 0 {
		stats["genai"] = genaiStats
	}

	return stats
}

// analyzeGenAITraffic analyzes GenAI-specific traffic
func (a *Analyzer) analyzeGenAITraffic(records []storage.TrafficRecord) map[string]interface{} {
	genaiStats := make(map[string]interface{})
	
	modelUsage := make(map[string]int)
	totalTokens := 0
	streamingRequests := 0
	
	for _, record := range records {
		for _, tag := range record.Tags {
			if strings.HasPrefix(tag, "model:") {
				model := strings.TrimPrefix(tag, "model:")
				modelUsage[model]++
			}
			if tag == "streaming" {
				streamingRequests++
			}
		}
		
		// Try to extract token usage from response
		if record.Type == "response" && record.Body != "" {
			var data map[string]interface{}
			if err := json.Unmarshal([]byte(record.Body), &data); err == nil {
				if usage, ok := data["usage"].(map[string]interface{}); ok {
					if total, ok := usage["total_tokens"].(float64); ok {
						totalTokens += int(total)
					}
				}
			}
		}
	}
	
	if len(modelUsage) > 0 {
		genaiStats["models"] = modelUsage
	}
	if totalTokens > 0 {
		genaiStats["total_tokens"] = totalTokens
	}
	if streamingRequests > 0 {
		genaiStats["streaming_requests"] = streamingRequests
	}
	
	return genaiStats
}

// Helper types and functions

type endpointCount struct {
	endpoint string
	count    int
}

func getTopEndpoints(endpoints map[string]int, n int) []endpointCount {
	// Convert map to slice for sorting
	var counts []endpointCount
	for ep, count := range endpoints {
		counts = append(counts, endpointCount{ep, count})
	}
	
	// Sort by count (descending)
	for i := 0; i < len(counts); i++ {
		for j := i + 1; j < len(counts); j++ {
			if counts[j].count > counts[i].count {
				counts[i], counts[j] = counts[j], counts[i]
			}
		}
	}
	
	// Return top N
	if len(counts) > n {
		return counts[:n]
	}
	return counts
}

func extractPath(urlStr string) string {
	// Simple URL path extraction
	parts := strings.Split(urlStr, "?")
	path := parts[0]
	
	// Remove protocol and host if present
	if idx := strings.Index(path, "://"); idx != -1 {
		path = path[idx+3:]
		if idx := strings.Index(path, "/"); idx != -1 {
			path = path[idx:]
		}
	}
	
	return path
}