package main

import (
	"bufio"
	"crypto/tls"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

type ProxyServer struct {
	port     int
	logFile  string
	logger   *logrus.Logger
	logWriter *os.File
}

type RequestLog struct {
	Timestamp    time.Time `json:"timestamp"`
	Method       string    `json:"method"`
	URL          string    `json:"url"`
	Host         string    `json:"host"`
	Headers      map[string]string `json:"headers"`
	Body         string    `json:"body,omitempty"`
	ResponseCode int       `json:"response_code,omitempty"`
	ResponseBody string    `json:"response_body,omitempty"`
	Duration     int64     `json:"duration_ms"`
	IsGenAI      bool      `json:"is_genai"`
	Provider     string    `json:"provider,omitempty"`
}

func main() {
	var port = flag.Int("port", 8080, "Proxy server port")
	var logFile = flag.String("log-file", "traffic.log", "Traffic log file")
	var verbose = flag.Bool("verbose", false, "Verbose logging")
	flag.Parse()

	proxy := &ProxyServer{
		port:    *port,
		logFile: *logFile,
		logger:  logrus.New(),
	}

	if *verbose {
		proxy.logger.SetLevel(logrus.DebugLevel)
	}

	// Open log file
	var err error
	proxy.logWriter, err = os.OpenFile(*logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		log.Fatalf("Failed to open log file: %v", err)
	}
	defer proxy.logWriter.Close()

	proxy.logger.Infof("Starting proxy server on port %d", *port)
	proxy.logger.Infof("Logging traffic to: %s", *logFile)

	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", *port),
		Handler: proxy,
	}

	log.Fatal(server.ListenAndServe())
}

func (p *ProxyServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()
	
	if r.Method == http.MethodConnect {
		p.handleHTTPS(w, r, startTime)
	} else {
		p.handleHTTP(w, r, startTime)
	}
}

func (p *ProxyServer) handleHTTP(w http.ResponseWriter, r *http.Request, startTime time.Time) {
	// Create request log entry
	reqLog := &RequestLog{
		Timestamp: startTime,
		Method:    r.Method,
		URL:       r.URL.String(),
		Host:      r.Host,
		Headers:   make(map[string]string),
	}

	// Copy headers
	for name, values := range r.Header {
		reqLog.Headers[name] = strings.Join(values, ", ")
	}

	// Read and log request body for POST/PUT requests
	var bodyBytes []byte
	if r.Method == "POST" || r.Method == "PUT" || r.Method == "PATCH" {
		bodyBytes, _ = io.ReadAll(r.Body)
		r.Body.Close()
		reqLog.Body = string(bodyBytes)
	}

	// Detect GenAI providers
	reqLog.IsGenAI, reqLog.Provider = p.detectGenAIProvider(r.Host, r.URL.Path)

	// Create new request for forwarding
	targetURL := r.URL
	if targetURL.Scheme == "" {
		targetURL.Scheme = "http"
	}
	if targetURL.Host == "" {
		targetURL.Host = r.Host
	}

	proxyReq, err := http.NewRequest(r.Method, targetURL.String(), nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Copy headers
	for name, values := range r.Header {
		for _, value := range values {
			proxyReq.Header.Add(name, value)
		}
	}

	// Add body if present
	if len(bodyBytes) > 0 {
		proxyReq.Body = io.NopCloser(strings.NewReader(string(bodyBytes)))
		proxyReq.ContentLength = int64(len(bodyBytes))
	}

	// Make the request
	client := &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}

	resp, err := client.Do(proxyReq)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	// Read response body
	respBody, _ := io.ReadAll(resp.Body)
	reqLog.ResponseCode = resp.StatusCode
	if reqLog.IsGenAI {
		reqLog.ResponseBody = string(respBody)
	}

	// Copy response headers
	for name, values := range resp.Header {
		for _, value := range values {
			w.Header().Add(name, value)
		}
	}

	// Write response
	w.WriteHeader(resp.StatusCode)
	w.Write(respBody)

	// Calculate duration and log
	reqLog.Duration = time.Since(startTime).Milliseconds()
	p.logRequest(reqLog)
}

func (p *ProxyServer) handleHTTPS(w http.ResponseWriter, r *http.Request, startTime time.Time) {
	// Log HTTPS CONNECT request
	reqLog := &RequestLog{
		Timestamp: startTime,
		Method:    r.Method,
		URL:       r.URL.String(),
		Host:      r.Host,
		Headers:   make(map[string]string),
	}

	for name, values := range r.Header {
		reqLog.Headers[name] = strings.Join(values, ", ")
	}

	reqLog.IsGenAI, reqLog.Provider = p.detectGenAIProvider(r.Host, "")

	// Establish connection to target server
	destConn, err := net.DialTimeout("tcp", r.Host, 10*time.Second)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	defer destConn.Close()

	// Send 200 Connection Established
	w.WriteHeader(http.StatusOK)

	// Hijack the connection
	hijacker, ok := w.(http.Hijacker)
	if !ok {
		http.Error(w, "Hijacking not supported", http.StatusInternalServerError)
		return
	}

	clientConn, _, err := hijacker.Hijack()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer clientConn.Close()

	// Start proxying data
	go func() {
		io.Copy(destConn, clientConn)
	}()
	io.Copy(clientConn, destConn)

	reqLog.Duration = time.Since(startTime).Milliseconds()
	reqLog.ResponseCode = 200
	p.logRequest(reqLog)
}

func (p *ProxyServer) detectGenAIProvider(host, path string) (bool, string) {
	host = strings.ToLower(host)
	
	genaiProviders := map[string]string{
		"api.openai.com":           "OpenAI",
		"api.anthropic.com":        "Anthropic",
		"generativelanguage.googleapis.com": "Google",
		"api.cohere.ai":           "Cohere",
		"api.ai21.com":            "AI21",
		"api.together.xyz":        "Together",
		"api.replicate.com":       "Replicate",
		"api.huggingface.co":      "HuggingFace",
		"openai.azure.com":        "Azure OpenAI",
		"api.mistral.ai":          "Mistral",
		"api.perplexity.ai":       "Perplexity",
	}

	for domain, provider := range genaiProviders {
		if strings.Contains(host, domain) {
			return true, provider
		}
	}

	// Check for common AI API patterns
	if strings.Contains(host, "openai") || 
	   strings.Contains(host, "anthropic") ||
	   strings.Contains(host, "claude") ||
	   strings.Contains(host, "gpt") ||
	   strings.Contains(path, "/chat/completions") ||
	   strings.Contains(path, "/completions") ||
	   strings.Contains(path, "/v1/messages") {
		return true, "Unknown AI Provider"
	}

	return false, ""
}

func (p *ProxyServer) logRequest(reqLog *RequestLog) {
	// Log to console
	if reqLog.IsGenAI {
		p.logger.WithFields(logrus.Fields{
			"method":   reqLog.Method,
			"host":     reqLog.Host,
			"provider": reqLog.Provider,
			"duration": reqLog.Duration,
			"status":   reqLog.ResponseCode,
		}).Info("GenAI API Request")
	} else {
		p.logger.WithFields(logrus.Fields{
			"method":   reqLog.Method,
			"host":     reqLog.Host,
			"duration": reqLog.Duration,
			"status":   reqLog.ResponseCode,
		}).Debug("HTTP Request")
	}

	// Write to log file in JSON format
	if p.logWriter != nil {
		logLine := fmt.Sprintf(`{"timestamp":"%s","method":"%s","url":"%s","host":"%s","response_code":%d,"duration_ms":%d,"is_genai":%t,"provider":"%s"}`,
			reqLog.Timestamp.Format(time.RFC3339),
			reqLog.Method,
			reqLog.URL,
			reqLog.Host,
			reqLog.ResponseCode,
			reqLog.Duration,
			reqLog.IsGenAI,
			reqLog.Provider,
		)
		
		if reqLog.IsGenAI && reqLog.Body != "" {
			// Escape quotes in JSON body
			escapedBody := strings.ReplaceAll(reqLog.Body, `"`, `\"`)
			logLine = strings.TrimSuffix(logLine, "}") + fmt.Sprintf(`,"request_body":"%s"}`, escapedBody)
		}
		
		fmt.Fprintln(p.logWriter, logLine)
	}
}