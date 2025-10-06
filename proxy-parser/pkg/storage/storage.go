package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"
)

// TrafficRecord represents a single traffic record
type TrafficRecord struct {
	ID         string            `json:"id"`
	Timestamp  time.Time         `json:"timestamp"`
	Type       string            `json:"type"` // "request" or "response"
	Method     string            `json:"method,omitempty"`
	URL        string            `json:"url,omitempty"`
	StatusCode int               `json:"status_code,omitempty"`
	Headers    map[string]string `json:"headers"`
	Body       string            `json:"body,omitempty"`
	Tags       []string          `json:"tags,omitempty"`
	Duration   time.Duration     `json:"duration,omitempty"`
}

// Storage handles traffic record storage
type Storage struct {
	mu       sync.Mutex
	file     *os.File
	encoder  *json.Encoder
	records  []TrafficRecord
	filepath string
}

// NewStorage creates a new storage instance
func NewStorage(filepath string) (*Storage, error) {
	s := &Storage{
		records:  make([]TrafficRecord, 0),
		filepath: filepath,
	}

	if filepath != "" {
		file, err := os.OpenFile(filepath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			return nil, fmt.Errorf("failed to open output file: %w", err)
		}
		s.file = file
		s.encoder = json.NewEncoder(file)
		s.encoder.SetIndent("", "  ")
	}

	return s, nil
}

// Store saves a traffic record
func (s *Storage) Store(record *TrafficRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.records = append(s.records, *record)

	if s.encoder != nil {
		if err := s.encoder.Encode(record); err != nil {
			return fmt.Errorf("failed to encode record: %w", err)
		}
	}

	return nil
}

// GetRecords returns all stored records
func (s *Storage) GetRecords() []TrafficRecord {
	s.mu.Lock()
	defer s.mu.Unlock()

	recordsCopy := make([]TrafficRecord, len(s.records))
	copy(recordsCopy, s.records)
	return recordsCopy
}

// LoadFromFile loads traffic records from a JSON file
func (s *Storage) LoadFromFile(filepath string) ([]TrafficRecord, error) {
	file, err := os.Open(filepath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	var records []TrafficRecord
	decoder := json.NewDecoder(file)

	// Read records one by one (handling newline-delimited JSON)
	for decoder.More() {
		var record TrafficRecord
		if err := decoder.Decode(&record); err != nil {
			// Try to continue reading if one record fails
			continue
		}
		records = append(records, record)
	}

	return records, nil
}

// Close closes the storage file
func (s *Storage) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.file != nil {
		// Write summary if we have records
		if len(s.records) > 0 {
			summary := s.generateSummary()
			summaryJSON, _ := json.MarshalIndent(summary, "", "  ")
			fmt.Fprintf(s.file, "\n// Summary:\n// %s\n", string(summaryJSON))
		}
		return s.file.Close()
	}
	return nil
}

// generateSummary creates a summary of the stored records
func (s *Storage) generateSummary() map[string]interface{} {
	summary := map[string]interface{}{
		"total_records": len(s.records),
		"requests":      0,
		"responses":     0,
		"genai_records": 0,
	}

	endpoints := make(map[string]int)
	models := make(map[string]int)

	for _, record := range s.records {
		if record.Type == "request" {
			summary["requests"] = summary["requests"].(int) + 1
			if record.URL != "" {
				endpoints[record.URL]++
			}
		} else if record.Type == "response" {
			summary["responses"] = summary["responses"].(int) + 1
		}

		// Count GenAI records
		for _, tag := range record.Tags {
			if tag == "genai" || tag == "genai-response" {
				summary["genai_records"] = summary["genai_records"].(int) + 1
			}
			if len(tag) > 6 && tag[:6] == "model:" {
				models[tag[6:]]++
			}
		}
	}

	summary["top_endpoints"] = getTopN(endpoints, 5)
	summary["models_used"] = models

	return summary
}

// getTopN returns the top N items from a frequency map
func getTopN(freq map[string]int, n int) []map[string]interface{} {
	type kv struct {
		Key   string
		Value int
	}

	var sorted []kv
	for k, v := range freq {
		sorted = append(sorted, kv{k, v})
	}

	// Simple bubble sort for top N
	for i := 0; i < len(sorted); i++ {
		for j := i + 1; j < len(sorted); j++ {
			if sorted[j].Value > sorted[i].Value {
				sorted[i], sorted[j] = sorted[j], sorted[i]
			}
		}
	}

	result := make([]map[string]interface{}, 0)
	for i := 0; i < n && i < len(sorted); i++ {
		result = append(result, map[string]interface{}{
			"endpoint": sorted[i].Key,
			"count":    sorted[i].Value,
		})
	}

	return result
}

// FilterRecords filters records based on criteria
func FilterRecords(records []TrafficRecord, filter FilterOptions) []TrafficRecord {
	filtered := make([]TrafficRecord, 0)

	for _, record := range records {
		if filter.Type != "" && record.Type != filter.Type {
			continue
		}

		if filter.GenAIOnly {
			hasGenAI := false
			for _, tag := range record.Tags {
				if tag == "genai" || tag == "genai-response" {
					hasGenAI = true
					break
				}
			}
			if !hasGenAI {
				continue
			}
		}

		if filter.Endpoint != "" && record.URL != "" {
			if !contains(record.URL, filter.Endpoint) {
				continue
			}
		}

		filtered = append(filtered, record)
	}

	return filtered
}

// FilterOptions defines filtering criteria
type FilterOptions struct {
	Type      string // "request" or "response"
	GenAIOnly bool
	Endpoint  string
	StartTime *time.Time
	EndTime   *time.Time
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && s[0:len(substr)] == substr
}