package proxy

import (
	"bufio"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/http"
	"regexp"
	"strings"
	"time"

	"codex-traffic-proxy/internal/config"
	"codex-traffic-proxy/internal/logger"
	"codex-traffic-proxy/internal/parser"
	"codex-traffic-proxy/pkg/models"

	"github.com/elazarl/goproxy"
	"github.com/google/uuid"
)

type Proxy struct {
	config     *config.Config
	logger     *logger.Logger
	parser     *parser.Parser
	server     *http.Server
	requestLog map[string]*models.RequestInfo
}

func NewProxy(cfg *config.Config, log *logger.Logger) *Proxy {
	return &Proxy{
		config:     cfg,
		logger:     log,
		parser:     parser.NewParser(cfg, log),
		requestLog: make(map[string]*models.RequestInfo),
	}
}

func (p *Proxy) Start() error {
	proxy := goproxy.NewProxyHttpServer()
	proxy.Verbose = p.config.Proxy.Verbose

	// Set up request handlers
	proxy.OnRequest().DoFunc(p.handleRequest)
	proxy.OnResponse().DoFunc(p.handleResponse)

	// Handle CONNECT method for HTTPS
	proxy.OnRequest().HandleConnectFunc(p.handleConnect)

	addr := fmt.Sprintf("%s:%d", p.config.Proxy.ListenAddr, p.config.Proxy.Port)

	p.logger.Info("Starting proxy server", "address", addr)

	server := &http.Server{
		Addr:         addr,
		Handler:      proxy,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		TLSConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
	}

	p.server = server

	return server.ListenAndServe()
}

func (p *Proxy) handleRequest(req *http.Request, ctx *goproxy.ProxyCtx) (*http.Request, *http.Response) {
	requestID := uuid.New().String()

	// Create request info
	reqInfo := &models.RequestInfo{
		ID:        requestID,
		Method:    req.Method,
		URL:       req.URL.String(),
		Host:      req.Host,
		Path:      req.URL.Path,
		Headers:   make(map[string]string),
		Timestamp: time.Now(),
		Direction: "outbound",
	}

	// Extract headers
	for name, values := range req.Header {
		if len(values) > 0 {
			reqInfo.Headers[name] = values[0]
		}
	}

	// Store request info
	p.requestLog[requestID] = reqInfo

	p.logger.Info("Request intercepted",
		"id", requestID,
		"method", req.Method,
		"host", req.Host,
		"path", req.URL.Path,
	)

	// Parse the request if parser is enabled
	if p.config.Parser.Enabled {
		if err := p.parser.ParseRequest(req, reqInfo); err != nil {
			p.logger.Error("Failed to parse request", "error", err, "id", requestID)
		}
	}

	return req, nil
}

func (p *Proxy) handleResponse(resp *http.Response, ctx *goproxy.ProxyCtx) *http.Response {
	// Find the corresponding request
	var requestID string
	for id, reqInfo := range p.requestLog {
		if reqInfo.Host == resp.Request.Host && reqInfo.Path == resp.Request.URL.Path {
			requestID = id
			break
		}
	}

	if requestID == "" {
		requestID = "unknown"
	}

	// Update request info with response details
	if reqInfo, exists := p.requestLog[requestID]; exists {
		reqInfo.StatusCode = resp.StatusCode
		reqInfo.ResponseHeaders = make(map[string]string)
		for name, values := range resp.Header {
			if len(values) > 0 {
				reqInfo.ResponseHeaders[name] = values[0]
			}
		}
		reqInfo.ResponseTimestamp = time.Now()
		reqInfo.Duration = reqInfo.ResponseTimestamp.Sub(reqInfo.Timestamp)

		// Read response body for analysis
		if resp.Body != nil {
			bodyBytes, err := io.ReadAll(resp.Body)
			if err == nil {
				reqInfo.ResponseBody = string(bodyBytes)
				// Create a new ReadCloser for the body
				resp.Body = io.NopCloser(strings.NewReader(reqInfo.ResponseBody))
			}
		}

		p.logger.Info("Response intercepted",
			"id", requestID,
			"status", resp.StatusCode,
			"duration", reqInfo.Duration,
		)

		// Parse the response if parser is enabled
		if p.config.Parser.Enabled {
			if err := p.parser.ParseResponse(resp, reqInfo); err != nil {
				p.logger.Error("Failed to parse response", "error", err, "id", requestID)
			}
		}
	}

	return resp
}

func (p *Proxy) handleConnect(host string, ctx *goproxy.ProxyCtx) (*goproxy.ConnectAction, string) {
	p.logger.Info("HTTPS CONNECT request", "host", host)

	// For HTTPS, we'll just pass it through
	return goproxy.OkConnect, host
}

func (p *Proxy) Shutdown() error {
	if p.server != nil {
		p.logger.Info("Shutting down proxy server")
		return p.server.Close()
	}
	return nil
}