package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"sort"
	"strings"
	"time"
)

type RequestLog struct {
	Timestamp    time.Time `json:"timestamp"`
	Method       string    `json:"method"`
	URL          string    `json:"url"`
	Host         string    `json:"host"`
	Headers      map[string]string `json:"headers"`
	RequestBody  string    `json:"request_body,omitempty"`
	ResponseCode int       `json:"response_code,omitempty"`
	ResponseBody string    `json:"response_body,omitempty"`
	Duration     int64     `json:"duration_ms"`
	IsGenAI      bool      `json:"is_genai"`
	Provider     string    `json:"provider,omitempty"`
}

type AnalysisReport struct {
	Summary          Summary                    `json:"summary"`
	ProviderStats    map[string]ProviderStat    `json:"provider_stats"`
	RequestsByHour   map[string]int             `json:"requests_by_hour"`
	TokenUsage       TokenUsage                 `json:"token_usage"`
	ErrorAnalysis    ErrorAnalysis              `json:"error_analysis"`
	TopEndpoints     []EndpointStat             `json:"top_endpoints"`
	PerformanceStats PerformanceStats           `json:"performance_stats"`
}

type Summary struct {
	TotalRequests    int     `json:"total_requests"`
	GenAIRequests    int     `json:"genai_requests"`
	UniqueProviders  int     `json:"unique_providers"`
	AvgResponseTime  float64 `json:"avg_response_time_ms"`
	ErrorRate        float64 `json:"error_rate_percent"`
	TimeRange        string  `json:"time_range"`
}

type ProviderStat struct {
	RequestCount    int     `json:"request_count"`
	AvgResponseTime float64 `json:"avg_response_time_ms"`
	ErrorCount      int     `json:"error_count"`
	ErrorRate       float64 `json:"error_rate_percent"`
	Endpoints       []string `json:"endpoints"`
}

type TokenUsage struct {
	TotalInputTokens  int `json:"total_input_tokens"`
	TotalOutputTokens int `json:"total_output_tokens"`
	TotalTokens       int `json:"total_tokens"`
	EstimatedCost     float64 `json:"estimated_cost_usd"`
}

type ErrorAnalysis struct {
	TotalErrors     int                    `json:"total_errors"`
	ErrorsByCode    map[string]int         `json:"errors_by_code"`
	ErrorsByProvider map[string]int        `json:"errors_by_provider"`
	CommonErrors    []string               `json:"common_errors"`
}

type EndpointStat struct {
	Endpoint     string  `json:"endpoint"`
	RequestCount int     `json:"request_count"`
	AvgDuration  float64 `json:"avg_duration_ms"`
}

type PerformanceStats struct {
	MinResponseTime float64 `json:"min_response_time_ms"`
	MaxResponseTime float64 `json:"max_response_time_ms"`
	P50ResponseTime float64 `json:"p50_response_time_ms"`
	P95ResponseTime float64 `json:"p95_response_time_ms"`
	P99ResponseTime float64 `json:"p99_response_time_ms"`
}

func main() {
	var inputFile = flag.String("input", "traffic.log", "Input traffic log file")
	var outputFile = flag.String("output", "analysis.json", "Output analysis file")
	var format = flag.String("format", "json", "Output format: json, csv, summary")
	var filterProvider = flag.String("provider", "", "Filter by provider")
	var filterGenAI = flag.Bool("genai-only", false, "Only analyze GenAI requests")
	flag.Parse()

	// Read and parse log file
	requests, err := parseLogFile(*inputFile)
	if err != nil {
		log.Fatalf("Failed to parse log file: %v", err)
	}

	// Apply filters
	if *filterGenAI {
		requests = filterGenAIRequests(requests)
	}
	if *filterProvider != "" {
		requests = filterByProvider(requests, *filterProvider)
	}

	fmt.Printf("Parsed %d requests\n", len(requests))

	// Generate analysis
	report := generateAnalysis(requests)

	// Output results
	switch *format {
	case "json":
		outputJSON(report, *outputFile)
	case "csv":
		outputCSV(requests, *outputFile)
	case "summary":
		outputSummary(report)
	default:
		log.Fatalf("Unknown format: %s", *format)
	}

	fmt.Printf("Analysis complete. Results written to %s\n", *outputFile)
}

func parseLogFile(filename string) ([]RequestLog, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var requests []RequestLog
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) == "" {
			continue
		}

		var req RequestLog
		if err := json.Unmarshal([]byte(line), &req); err != nil {
			fmt.Printf("Warning: Failed to parse line: %s\n", line)
			continue
		}

		requests = append(requests, req)
	}

	return requests, scanner.Err()
}

func filterGenAIRequests(requests []RequestLog) []RequestLog {
	var filtered []RequestLog
	for _, req := range requests {
		if req.IsGenAI {
			filtered = append(filtered, req)
		}
	}
	return filtered
}

func filterByProvider(requests []RequestLog, provider string) []RequestLog {
	var filtered []RequestLog
	for _, req := range requests {
		if strings.EqualFold(req.Provider, provider) {
			filtered = append(filtered, req)
		}
	}
	return filtered
}

func generateAnalysis(requests []RequestLog) AnalysisReport {
	if len(requests) == 0 {
		return AnalysisReport{}
	}

	report := AnalysisReport{
		ProviderStats:  make(map[string]ProviderStat),
		RequestsByHour: make(map[string]int),
	}

	// Basic stats
	var totalDuration int64
	var genaiCount int
	var errorCount int
	var durations []int64
	providers := make(map[string]bool)
	endpointCounts := make(map[string]int)
	endpointDurations := make(map[string][]int64)
	errorsByCode := make(map[string]int)
	errorsByProvider := make(map[string]int)

	var minTime, maxTime time.Time
	if len(requests) > 0 {
		minTime = requests[0].Timestamp
		maxTime = requests[0].Timestamp
	}

	for _, req := range requests {
		// Update time range
		if req.Timestamp.Before(minTime) {
			minTime = req.Timestamp
		}
		if req.Timestamp.After(maxTime) {
			maxTime = req.Timestamp
		}

		// Basic counters
		totalDuration += req.Duration
		durations = append(durations, req.Duration)
		
		if req.IsGenAI {
			genaiCount++
			providers[req.Provider] = true
		}

		if req.ResponseCode >= 400 {
			errorCount++
			errorsByCode[fmt.Sprintf("%d", req.ResponseCode)]++
			if req.Provider != "" {
				errorsByProvider[req.Provider]++
			}
		}

		// Requests by hour
		hourKey := req.Timestamp.Format("2006-01-02 15:00")
		report.RequestsByHour[hourKey]++

		// Endpoint stats
		endpoint := extractEndpoint(req.URL)
		endpointCounts[endpoint]++
		endpointDurations[endpoint] = append(endpointDurations[endpoint], req.Duration)

		// Provider stats
		if req.IsGenAI && req.Provider != "" {
			stat := report.ProviderStats[req.Provider]
			stat.RequestCount++
			if req.ResponseCode >= 400 {
				stat.ErrorCount++
			}
			stat.Endpoints = appendUnique(stat.Endpoints, endpoint)
			report.ProviderStats[req.Provider] = stat
		}
	}

	// Calculate averages and percentiles
	sort.Slice(durations, func(i, j int) bool { return durations[i] < durations[j] })

	report.Summary = Summary{
		TotalRequests:   len(requests),
		GenAIRequests:   genaiCount,
		UniqueProviders: len(providers),
		AvgResponseTime: float64(totalDuration) / float64(len(requests)),
		ErrorRate:       float64(errorCount) / float64(len(requests)) * 100,
		TimeRange:       fmt.Sprintf("%s to %s", minTime.Format(time.RFC3339), maxTime.Format(time.RFC3339)),
	}

	// Provider stats calculations
	for provider, stat := range report.ProviderStats {
		if stat.RequestCount > 0 {
			stat.ErrorRate = float64(stat.ErrorCount) / float64(stat.RequestCount) * 100
			// Calculate average response time for this provider
			var providerDuration int64
			var providerCount int
			for _, req := range requests {
				if req.Provider == provider {
					providerDuration += req.Duration
					providerCount++
				}
			}
			stat.AvgResponseTime = float64(providerDuration) / float64(providerCount)
		}
		report.ProviderStats[provider] = stat
	}

	// Top endpoints
	type endpointPair struct {
		endpoint string
		count    int
	}
	var endpointPairs []endpointPair
	for endpoint, count := range endpointCounts {
		endpointPairs = append(endpointPairs, endpointPair{endpoint, count})
	}
	sort.Slice(endpointPairs, func(i, j int) bool {
		return endpointPairs[i].count > endpointPairs[j].count
	})

	for i, pair := range endpointPairs {
		if i >= 10 { // Top 10
			break
		}
		avgDuration := float64(0)
		if len(endpointDurations[pair.endpoint]) > 0 {
			var sum int64
			for _, d := range endpointDurations[pair.endpoint] {
				sum += d
			}
			avgDuration = float64(sum) / float64(len(endpointDurations[pair.endpoint]))
		}
		report.TopEndpoints = append(report.TopEndpoints, EndpointStat{
			Endpoint:     pair.endpoint,
			RequestCount: pair.count,
			AvgDuration:  avgDuration,
		})
	}

	// Performance stats
	if len(durations) > 0 {
		report.PerformanceStats = PerformanceStats{
			MinResponseTime: float64(durations[0]),
			MaxResponseTime: float64(durations[len(durations)-1]),
			P50ResponseTime: float64(durations[len(durations)*50/100]),
			P95ResponseTime: float64(durations[len(durations)*95/100]),
			P99ResponseTime: float64(durations[len(durations)*99/100]),
		}
	}

	// Error analysis
	report.ErrorAnalysis = ErrorAnalysis{
		TotalErrors:      errorCount,
		ErrorsByCode:     errorsByCode,
		ErrorsByProvider: errorsByProvider,
	}

	// Token usage estimation (simplified)
	report.TokenUsage = estimateTokenUsage(requests)

	return report
}

func extractEndpoint(url string) string {
	// Simple endpoint extraction - remove query params and normalize
	if idx := strings.Index(url, "?"); idx != -1 {
		url = url[:idx]
	}
	
	// Common AI API endpoints
	if strings.Contains(url, "/chat/completions") {
		return "/chat/completions"
	}
	if strings.Contains(url, "/completions") {
		return "/completions"
	}
	if strings.Contains(url, "/v1/messages") {
		return "/v1/messages"
	}
	
	return url
}

func appendUnique(slice []string, item string) []string {
	for _, existing := range slice {
		if existing == item {
			return slice
		}
	}
	return append(slice, item)
}

func estimateTokenUsage(requests []RequestLog) TokenUsage {
	// Simplified token estimation based on request/response body lengths
	var inputChars, outputChars int
	
	for _, req := range requests {
		if req.IsGenAI {
			inputChars += len(req.RequestBody)
			outputChars += len(req.ResponseBody)
		}
	}
	
	// Rough estimation: ~4 characters per token
	inputTokens := inputChars / 4
	outputTokens := outputChars / 4
	
	// Rough cost estimation (OpenAI GPT-4 pricing as baseline)
	inputCost := float64(inputTokens) * 0.00003  // $0.03 per 1K tokens
	outputCost := float64(outputTokens) * 0.00006 // $0.06 per 1K tokens
	
	return TokenUsage{
		TotalInputTokens:  inputTokens,
		TotalOutputTokens: outputTokens,
		TotalTokens:       inputTokens + outputTokens,
		EstimatedCost:     inputCost + outputCost,
	}
}

func outputJSON(report AnalysisReport, filename string) {
	file, err := os.Create(filename)
	if err != nil {
		log.Fatalf("Failed to create output file: %v", err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(report); err != nil {
		log.Fatalf("Failed to write JSON: %v", err)
	}
}

func outputCSV(requests []RequestLog, filename string) {
	file, err := os.Create(filename)
	if err != nil {
		log.Fatalf("Failed to create output file: %v", err)
	}
	defer file.Close()

	// CSV header
	fmt.Fprintln(file, "timestamp,method,host,provider,response_code,duration_ms,is_genai")

	for _, req := range requests {
		fmt.Fprintf(file, "%s,%s,%s,%s,%d,%d,%t\n",
			req.Timestamp.Format(time.RFC3339),
			req.Method,
			req.Host,
			req.Provider,
			req.ResponseCode,
			req.Duration,
			req.IsGenAI,
		)
	}
}

func outputSummary(report AnalysisReport) {
	fmt.Println("\n=== TRAFFIC ANALYSIS SUMMARY ===")
	fmt.Printf("Total Requests: %d\n", report.Summary.TotalRequests)
	fmt.Printf("GenAI Requests: %d (%.1f%%)\n", 
		report.Summary.GenAIRequests, 
		float64(report.Summary.GenAIRequests)/float64(report.Summary.TotalRequests)*100)
	fmt.Printf("Unique Providers: %d\n", report.Summary.UniqueProviders)
	fmt.Printf("Average Response Time: %.1f ms\n", report.Summary.AvgResponseTime)
	fmt.Printf("Error Rate: %.1f%%\n", report.Summary.ErrorRate)
	fmt.Printf("Time Range: %s\n", report.Summary.TimeRange)

	fmt.Println("\n=== PROVIDER BREAKDOWN ===")
	for provider, stats := range report.ProviderStats {
		fmt.Printf("%s: %d requests, %.1f ms avg, %.1f%% errors\n",
			provider, stats.RequestCount, stats.AvgResponseTime, stats.ErrorRate)
	}

	fmt.Println("\n=== TOP ENDPOINTS ===")
	for i, endpoint := range report.TopEndpoints {
		if i >= 5 { // Top 5
			break
		}
		fmt.Printf("%s: %d requests, %.1f ms avg\n",
			endpoint.Endpoint, endpoint.RequestCount, endpoint.AvgDuration)
	}

	fmt.Println("\n=== TOKEN USAGE ESTIMATE ===")
	fmt.Printf("Input Tokens: %d\n", report.TokenUsage.TotalInputTokens)
	fmt.Printf("Output Tokens: %d\n", report.TokenUsage.TotalOutputTokens)
	fmt.Printf("Estimated Cost: $%.4f\n", report.TokenUsage.EstimatedCost)
}