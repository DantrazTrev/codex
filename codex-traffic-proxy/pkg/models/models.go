package models

import (
	"time"
)

type RequestInfo struct {
	ID                string                 `json:"id"`
	Method            string                 `json:"method"`
	URL               string                 `json:"url"`
	Host              string                 `json:"host"`
	Path              string                 `json:"path"`
	Headers           map[string]string      `json:"headers"`
	ResponseHeaders   map[string]string      `json:"response_headers,omitempty"`
	StatusCode        int                    `json:"status_code,omitempty"`
	ResponseBody      string                 `json:"response_body,omitempty"`
	ResponseJSON      interface{}            `json:"response_json,omitempty"`
	Timestamp         time.Time              `json:"timestamp"`
	ResponseTimestamp time.Time              `json:"response_timestamp,omitempty"`
	Duration          time.Duration          `json:"duration,omitempty"`
	Direction         string                 `json:"direction"`
	APIKeys           []APIKeyInfo           `json:"api_keys,omitempty"`
	Tokens            []TokenInfo            `json:"tokens,omitempty"`
	Endpoints         []EndpointInfo         `json:"endpoints,omitempty"`
	Patterns          []string               `json:"patterns,omitempty"`
	ContentTypes      []string               `json:"content_types,omitempty"`
	Models            []string               `json:"models,omitempty"`
	FinishReasons     []string               `json:"finish_reasons,omitempty"`
	Usage             UsageInfo              `json:"usage,omitempty"`
}

type APIKeyInfo struct {
	Key       string    `json:"key"`
	Location  string    `json:"location"`
	Timestamp time.Time `json:"timestamp"`
}

type TokenInfo struct {
	Type      string    `json:"type"`
	Value     string    `json:"value"`
	Location  string    `json:"location"`
	Pattern   string    `json:"pattern"`
	Timestamp time.Time `json:"timestamp"`
}

type EndpointInfo struct {
	URL       string    `json:"url"`
	Host      string    `json:"host"`
	Path      string    `json:"path"`
	Method    string    `json:"method"`
	Service   string    `json:"service"`
	Timestamp time.Time `json:"timestamp"`
}

type UsageInfo struct {
	PromptTokens     int       `json:"prompt_tokens,omitempty"`
	CompletionTokens int       `json:"completion_tokens,omitempty"`
	TotalTokens      int       `json:"total_tokens,omitempty"`
	Timestamp        time.Time `json:"timestamp"`
}

type SessionSummary struct {
	SessionID       string                 `json:"session_id"`
	StartTime       time.Time              `json:"start_time"`
	EndTime         time.Time              `json:"end_time"`
	TotalRequests   int                    `json:"total_requests"`
	TotalDuration   time.Duration          `json:"total_duration"`
	APIKeys         []APIKeyInfo           `json:"api_keys"`
	Tokens          []TokenInfo            `json:"tokens"`
	Endpoints       []EndpointInfo         `json:"endpoints"`
	Models          []string               `json:"models"`
	TotalUsage      UsageInfo              `json:"total_usage"`
	ErrorCount      int                    `json:"error_count"`
	RateLimitCount  int                    `json:"rate_limit_count"`
	Metadata        map[string]interface{} `json:"metadata,omitempty"`
}

type TrafficStats struct {
	TotalRequests     int                    `json:"total_requests"`
	TotalErrors       int                    `json:"total_errors"`
	TotalRateLimited  int                    `json:"total_rate_limited"`
	UniqueHosts       int                    `json:"unique_hosts"`
	UniqueAPIKeys     int                    `json:"unique_api_keys"`
	UniqueTokens      int                    `json:"unique_tokens"`
	TotalTokensUsed   UsageInfo              `json:"total_tokens_used"`
	ServiceBreakdown  map[string]int         `json:"service_breakdown"`
	PatternBreakdown  map[string]int         `json:"pattern_breakdown"`
	TimeRange         TimeRange              `json:"time_range"`
}

type TimeRange struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
}