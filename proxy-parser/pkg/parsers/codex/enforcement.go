package codex

import (
	"encoding/json"
	"fmt"
	"log"
	"regexp"
	"strings"
	"time"
)

// EnforcementEngine handles policy enforcement for Codex traffic
type EnforcementEngine struct {
	rules      []EnforcementRule
	policies   map[string]Policy
	redactors  map[string]Redactor
	validators map[string]Validator
}

// EnforcementRule defines a single enforcement rule
type EnforcementRule struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Type        string   `json:"type"` // "content", "model", "token", "tool", "pii", "code", "path"
	Pattern     string   `json:"pattern"`
	Regex       *regexp.Regexp
	Action      string   `json:"action"` // "allow", "block", "redact", "log", "warn"
	Message     string   `json:"message"`
	Fields      []string `json:"fields"`
	Priority    int      `json:"priority"`
	MaxValue    int      `json:"max_value,omitempty"` // For token limits
	AllowedList []string `json:"allowed_list,omitempty"`
	BlockedList []string `json:"blocked_list,omitempty"`
}

// Policy defines enforcement policies
type Policy struct {
	Name          string            `json:"name"`
	Description   string            `json:"description"`
	Rules         []string          `json:"rules"` // Rule IDs
	DefaultAction string            `json:"default_action"`
	Enabled       bool              `json:"enabled"`
	Metadata      map[string]string `json:"metadata"`
}

// Redactor handles content redaction
type Redactor struct {
	Type        string `json:"type"`
	Pattern     *regexp.Regexp
	Replacement string `json:"replacement"`
}

// Validator validates content against rules
type Validator struct {
	Type     string
	Validate func(content string) (bool, string)
}

// EnforcementAction represents an enforcement action taken
type EnforcementAction struct {
	RuleID           string `json:"rule_id"`
	RuleName         string `json:"rule_name"`
	Action           string `json:"action"`
	Reason           string `json:"reason"`
	OriginalContent  string `json:"original_content,omitempty"`
	ModifiedContent  string `json:"modified_content,omitempty"`
	Field            string `json:"field,omitempty"`
	Timestamp        int64  `json:"timestamp"`
}

// NewEnforcementEngine creates a new enforcement engine with default rules
func NewEnforcementEngine() *EnforcementEngine {
	engine := &EnforcementEngine{
		rules:      make([]EnforcementRule, 0),
		policies:   make(map[string]Policy),
		redactors:  make(map[string]Redactor),
		validators: make(map[string]Validator),
	}
	
	// Load default rules
	engine.loadDefaultRules()
	engine.loadDefaultPolicies()
	engine.initializeRedactors()
	engine.initializeValidators()
	
	return engine
}

// loadDefaultRules loads standard enforcement rules
func (e *EnforcementEngine) loadDefaultRules() {
	defaultRules := []EnforcementRule{
		// Model restrictions
		{
			ID:          "model_restrict_1",
			Name:        "Restrict GPT-4 Usage",
			Type:        "model",
			Pattern:     "gpt-4",
			Action:      "log",
			Message:     "GPT-4 usage detected",
			Fields:      []string{"model"},
			Priority:    5,
			AllowedList: []string{"gpt-3.5-turbo", "gpt-4-turbo"},
		},
		
		// Token limits
		{
			ID:       "token_limit_1",
			Name:     "Max Token Limit",
			Type:     "token",
			Action:   "warn",
			Message:  "Token limit exceeded",
			MaxValue: 4000,
			Priority: 3,
		},
		
		// PII Detection
		{
			ID:      "pii_ssn",
			Name:    "SSN Detection",
			Type:    "pii",
			Pattern: `\b\d{3}-\d{2}-\d{4}\b`,
			Action:  "redact",
			Message: "SSN detected and redacted",
			Fields:  []string{"content", "user_message", "assistant_message"},
			Priority: 10,
		},
		{
			ID:      "pii_email",
			Name:    "Email Detection",
			Type:    "pii",
			Pattern: `\b[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Z|a-z]{2,}\b`,
			Action:  "log",
			Message: "Email address detected",
			Fields:  []string{"content", "user_message"},
			Priority: 7,
		},
		{
			ID:      "pii_phone",
			Name:    "Phone Number Detection",
			Type:    "pii",
			Pattern: `\b(?:\+?1[-.]?)?\(?[0-9]{3}\)?[-.]?[0-9]{3}[-.]?[0-9]{4}\b`,
			Action:  "redact",
			Message: "Phone number detected and redacted",
			Fields:  []string{"content", "user_message"},
			Priority: 8,
		},
		
		// Code patterns
		{
			ID:      "code_secret_1",
			Name:    "API Key Detection",
			Type:    "code",
			Pattern: `(api[_-]?key|apikey|secret[_-]?key|access[_-]?token)\s*[:=]\s*["'][\w-]{20,}["']`,
			Action:  "redact",
			Message: "Potential API key detected",
			Fields:  []string{"code", "content"},
			Priority: 10,
		},
		{
			ID:      "code_password",
			Name:    "Password Detection",
			Type:    "code",
			Pattern: `(password|passwd|pwd)\s*[:=]\s*["'][^"']+["']`,
			Action:  "redact",
			Message: "Password detected in code",
			Fields:  []string{"code"},
			Priority: 10,
		},
		{
			ID:      "code_exec",
			Name:    "Code Execution Detection",
			Type:    "code",
			Pattern: `(exec|eval|system|subprocess\.call|os\.system)\s*\(`,
			Action:  "warn",
			Message: "Potentially dangerous code execution detected",
			Fields:  []string{"code"},
			Priority: 6,
		},
		
		// File path restrictions
		{
			ID:          "path_system",
			Name:        "System Path Restriction",
			Type:        "path",
			Pattern:     `^/(etc|var|sys|proc|root)`,
			Action:      "block",
			Message:     "Access to system paths not allowed",
			Fields:      []string{"file_paths"},
			Priority:    9,
			BlockedList: []string{"/etc/passwd", "/etc/shadow", "/root"},
		},
		{
			ID:      "path_home",
			Name:    "Home Directory Warning",
			Type:    "path",
			Pattern: `~/\.ssh|~/\.aws|~/\.config`,
			Action:  "warn",
			Message: "Access to sensitive home directory files",
			Fields:  []string{"file_paths"},
			Priority: 7,
		},
		
		// Tool restrictions
		{
			ID:          "tool_restrict",
			Name:        "Tool Usage Restriction",
			Type:        "tool",
			Action:      "log",
			Message:     "Tool usage detected",
			Fields:      []string{"tools"},
			BlockedList: []string{"execute_command", "delete_file", "modify_system"},
			Priority:    5,
		},
		
		// Content filters
		{
			ID:      "content_profanity",
			Name:    "Profanity Filter",
			Type:    "content",
			Pattern: `\b(badword1|badword2|badword3)\b`, // Replace with actual patterns
			Action:  "redact",
			Message: "Inappropriate content detected",
			Fields:  []string{"content", "user_message", "assistant_message"},
			Priority: 6,
		},
		{
			ID:      "content_injection",
			Name:    "Injection Attack Detection",
			Type:    "content",
			Pattern: `(DROP TABLE|DELETE FROM|INSERT INTO|UPDATE SET|<script|javascript:|onerror=)`,
			Action:  "block",
			Message: "Potential injection attack detected",
			Fields:  []string{"content", "user_message", "code"},
			Priority: 10,
		},
	}
	
	// Compile regex patterns
	for i := range defaultRules {
		if defaultRules[i].Pattern != "" {
			defaultRules[i].Regex = regexp.MustCompile(defaultRules[i].Pattern)
		}
	}
	
	e.rules = defaultRules
}

// loadDefaultPolicies loads default enforcement policies
func (e *EnforcementEngine) loadDefaultPolicies() {
	e.policies["strict"] = Policy{
		Name:          "Strict Security",
		Description:   "Strict security policy with maximum enforcement",
		Rules:         []string{"pii_ssn", "pii_phone", "code_secret_1", "code_password", "path_system", "content_injection"},
		DefaultAction: "block",
		Enabled:       true,
	}
	
	e.policies["moderate"] = Policy{
		Name:          "Moderate Security",
		Description:   "Balanced security with warnings and logging",
		Rules:         []string{"pii_email", "code_exec", "path_home", "tool_restrict"},
		DefaultAction: "log",
		Enabled:       true,
	}
	
	e.policies["development"] = Policy{
		Name:          "Development Mode",
		Description:   "Minimal enforcement for development",
		Rules:         []string{"content_injection"},
		DefaultAction: "log",
		Enabled:       true,
	}
}

// initializeRedactors sets up content redactors
func (e *EnforcementEngine) initializeRedactors() {
	e.redactors["ssn"] = Redactor{
		Type:        "ssn",
		Pattern:     regexp.MustCompile(`\b\d{3}-\d{2}-\d{4}\b`),
		Replacement: "[SSN REDACTED]",
	}
	
	e.redactors["email"] = Redactor{
		Type:        "email",
		Pattern:     regexp.MustCompile(`\b[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Z|a-z]{2,}\b`),
		Replacement: "[EMAIL REDACTED]",
	}
	
	e.redactors["phone"] = Redactor{
		Type:        "phone",
		Pattern:     regexp.MustCompile(`\b(?:\+?1[-.]?)?\(?[0-9]{3}\)?[-.]?[0-9]{3}[-.]?[0-9]{4}\b`),
		Replacement: "[PHONE REDACTED]",
	}
	
	e.redactors["apikey"] = Redactor{
		Type:        "apikey",
		Pattern:     regexp.MustCompile(`(api[_-]?key|apikey|secret[_-]?key|access[_-]?token)\s*[:=]\s*["'][\w-]{20,}["']`),
		Replacement: "$1=[REDACTED]",
	}
	
	e.redactors["password"] = Redactor{
		Type:        "password",
		Pattern:     regexp.MustCompile(`(password|passwd|pwd)\s*[:=]\s*["'][^"']+["']`),
		Replacement: "$1=[REDACTED]",
	}
}

// initializeValidators sets up content validators
func (e *EnforcementEngine) initializeValidators() {
	e.validators["model"] = Validator{
		Type: "model",
		Validate: func(content string) (bool, string) {
			allowedModels := []string{"gpt-3.5-turbo", "gpt-4-turbo", "gpt-4o"}
			for _, allowed := range allowedModels {
				if content == allowed {
					return true, ""
				}
			}
			return false, fmt.Sprintf("Model %s is not in allowed list", content)
		},
	}
	
	e.validators["token_limit"] = Validator{
		Type: "token_limit",
		Validate: func(content string) (bool, string) {
			// Parse token count from content
			var tokens int
			if err := json.Unmarshal([]byte(content), &tokens); err == nil {
				if tokens > 4000 {
					return false, fmt.Sprintf("Token count %d exceeds limit of 4000", tokens)
				}
			}
			return true, ""
		},
	}
}

// EnforceRequest applies enforcement rules to a request
func (e *EnforcementEngine) EnforceRequest(req *ParsedRequest) []EnforcementAction {
	actions := make([]EnforcementAction, 0)
	
	// Apply rules in priority order
	for _, rule := range e.getSortedRules() {
		action := e.applyRuleToRequest(rule, req)
		if action != nil {
			actions = append(actions, *action)
			
			// If blocked, stop processing
			if action.Action == "block" {
				break
			}
		}
	}
	
	return actions
}

// EnforceResponse applies enforcement rules to a response
func (e *EnforcementEngine) EnforceResponse(resp *ParsedResponse) []EnforcementAction {
	actions := make([]EnforcementAction, 0)
	
	for _, rule := range e.getSortedRules() {
		action := e.applyRuleToResponse(rule, resp)
		if action != nil {
			actions = append(actions, *action)
			
			// Apply redaction if needed
			if action.Action == "redact" && action.ModifiedContent != "" {
				e.applyRedaction(resp, action)
			}
		}
	}
	
	return actions
}

// applyRuleToRequest applies a single rule to a request
func (e *EnforcementEngine) applyRuleToRequest(rule EnforcementRule, req *ParsedRequest) *EnforcementAction {
	var content string
	var field string
	
	// Check each field specified in the rule
	for _, f := range rule.Fields {
		switch f {
		case "model":
			content = req.Model
			field = f
		case "content", "user_message":
			content = req.UserMessage
			field = f
		case "code":
			content = req.Code
			field = f
		case "file_paths":
			content = strings.Join(req.FilePaths, " ")
			field = f
		case "tools":
			content = strings.Join(req.Tools, " ")
			field = f
		}
		
		if content != "" && e.matchesRule(rule, content) {
			return &EnforcementAction{
				RuleID:          rule.ID,
				RuleName:        rule.Name,
				Action:          rule.Action,
				Reason:          rule.Message,
				OriginalContent: truncate(content, 100),
				Field:           field,
				Timestamp:       getCurrentTimestamp(),
			}
		}
	}
	
	// Check token limits
	if rule.Type == "token" && rule.MaxValue > 0 {
		if req.MaxTokens > rule.MaxValue {
			return &EnforcementAction{
				RuleID:    rule.ID,
				RuleName:  rule.Name,
				Action:    rule.Action,
				Reason:    fmt.Sprintf("Max tokens %d exceeds limit %d", req.MaxTokens, rule.MaxValue),
				Field:     "max_tokens",
				Timestamp: getCurrentTimestamp(),
			}
		}
	}
	
	return nil
}

// applyRuleToResponse applies a single rule to a response
func (e *EnforcementEngine) applyRuleToResponse(rule EnforcementRule, resp *ParsedResponse) *EnforcementAction {
	var content string
	var field string
	
	for _, f := range rule.Fields {
		switch f {
		case "assistant_message", "content":
			content = resp.AssistantMessage
			field = f
		case "code":
			content = resp.Code
			field = f
		}
		
		if content != "" && e.matchesRule(rule, content) {
			action := &EnforcementAction{
				RuleID:          rule.ID,
				RuleName:        rule.Name,
				Action:          rule.Action,
				Reason:          rule.Message,
				OriginalContent: truncate(content, 100),
				Field:           field,
				Timestamp:       getCurrentTimestamp(),
			}
			
			// Apply redaction if needed
			if rule.Action == "redact" {
				if redactor, exists := e.redactors[strings.ToLower(rule.Type)]; exists {
					action.ModifiedContent = redactor.Pattern.ReplaceAllString(content, redactor.Replacement)
				}
			}
			
			return action
		}
	}
	
	return nil
}

// matchesRule checks if content matches a rule
func (e *EnforcementEngine) matchesRule(rule EnforcementRule, content string) bool {
	// Check regex pattern
	if rule.Regex != nil {
		return rule.Regex.MatchString(content)
	}
	
	// Check blocked list
	for _, blocked := range rule.BlockedList {
		if strings.Contains(content, blocked) {
			return true
		}
	}
	
	// Check if NOT in allowed list (if list exists)
	if len(rule.AllowedList) > 0 {
		found := false
		for _, allowed := range rule.AllowedList {
			if strings.Contains(content, allowed) {
				found = true
				break
			}
		}
		if !found {
			return true // Not in allowed list = matches rule for blocking
		}
	}
	
	return false
}

// applyRedaction applies redaction to response content
func (e *EnforcementEngine) applyRedaction(resp *ParsedResponse, action *EnforcementAction) {
	switch action.Field {
	case "assistant_message", "content":
		resp.AssistantMessage = action.ModifiedContent
	case "code":
		resp.Code = action.ModifiedContent
	}
}

// getSortedRules returns rules sorted by priority
func (e *EnforcementEngine) getSortedRules() []EnforcementRule {
	// Simple bubble sort by priority (higher priority first)
	sorted := make([]EnforcementRule, len(e.rules))
	copy(sorted, e.rules)
	
	for i := 0; i < len(sorted); i++ {
		for j := i + 1; j < len(sorted); j++ {
			if sorted[j].Priority > sorted[i].Priority {
				sorted[i], sorted[j] = sorted[j], sorted[i]
			}
		}
	}
	
	return sorted
}

// AddRule adds a custom rule
func (e *EnforcementEngine) AddRule(rule EnforcementRule) error {
	if rule.Pattern != "" {
		regex, err := regexp.Compile(rule.Pattern)
		if err != nil {
			return fmt.Errorf("invalid regex pattern: %w", err)
		}
		rule.Regex = regex
	}
	
	e.rules = append(e.rules, rule)
	return nil
}

// SetPolicy sets the active policy
func (e *EnforcementEngine) SetPolicy(policyName string) error {
	if _, exists := e.policies[policyName]; !exists {
		return fmt.Errorf("policy %s not found", policyName)
	}
	
	// Enable only rules in this policy
	activeRules := e.policies[policyName].Rules
	for i := range e.rules {
		e.rules[i].Action = "disabled"
		for _, ruleID := range activeRules {
			if e.rules[i].ID == ruleID {
				// Re-enable the rule with its original action
				e.rules[i].Action = e.getOriginalAction(e.rules[i].ID)
				break
			}
		}
	}
	
	log.Printf("Enforcement policy set to: %s", policyName)
	return nil
}

// getOriginalAction gets the original action for a rule
func (e *EnforcementEngine) getOriginalAction(ruleID string) string {
	// This would typically be stored, but for now return defaults
	actionMap := map[string]string{
		"pii_ssn":           "redact",
		"pii_phone":         "redact",
		"code_secret_1":     "redact",
		"code_password":     "redact",
		"content_injection": "block",
		"path_system":       "block",
	}
	
	if action, exists := actionMap[ruleID]; exists {
		return action
	}
	return "log"
}

// Helper functions

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

func getCurrentTimestamp() int64 {
	return time.Now().Unix()
}