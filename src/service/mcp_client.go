package service

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

var mcpLogger = log.Printf

type mcpRequest struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      int         `json:"id"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params,omitempty"`
}

type mcpResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      int             `json:"id"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

type mcpListToolsResult struct {
	Tools []MCPTool `json:"tools"`
}

type MCPTool struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	InputSchema interface{} `json:"input_schema"`
}

type MCPConnConfig struct {
	ID         string
	Name       string
	Type       string
	Command    string
	Args       []string
	SSEURL     string
	SSEHeaders map[string]string
}

type mcpClient struct {
	ID      string
	Config  MCPConnConfig
	Cmd     *exec.Cmd
	logFile *os.File
	session *mcpSession
	mu      sync.Mutex
}

type MCPManager struct {
	mu      sync.RWMutex
	clients map[string]*mcpClient
}

var Manager = NewMCPManager()

func NewMCPManager() *MCPManager {
	return &MCPManager{
		clients: make(map[string]*mcpClient),
	}
}

func (m *MCPManager) CreateConnection(config MCPConnConfig) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.clients[config.ID] = &mcpClient{
		ID:     config.ID,
		Config: config,
	}
	return nil
}

func mcpLogPath(name string) string {
	logDir := filepath.Join("data", "log")
	os.MkdirAll(logDir, 0755)
	return filepath.Join(logDir, fmt.Sprintf("mcp-%s-%s.log", name, time.Now().Format("20060102-150405")))
}

func (m *MCPManager) Connect(id string) error {
	m.mu.RLock()
	client, ok := m.clients[id]
	m.mu.RUnlock()
	if !ok {
		return fmt.Errorf("connection %s not found", id)
	}

	client.mu.Lock()
	defer client.mu.Unlock()

	if client.Config.Type == "stdio" {
		return m.connectStdio(client)
	} else if client.Config.Type == "sse" {
		mcpLogger("[MCP] Connect SSE: checking %s", client.Config.SSEURL)
		resp, err := http.Get(client.Config.SSEURL)
		if err != nil {
			mcpLogger("[MCP] Connect SSE: connection refused (%v), attempting to start process", err)
			if client.Config.Command == "" {
				return fmt.Errorf("sse endpoint %s unreachable and no command configured to start it", client.Config.SSEURL)
			}
			cmd := exec.Command(client.Config.Command, client.Config.Args...)
			logPath := mcpLogPath(client.ID)
			logFile, openErr := os.Create(logPath)
			if openErr != nil {
				mcpLogger("[MCP] Connect SSE: failed to create log file %s: %v", logPath, openErr)
			} else {
				cmd.Stdout = logFile
				cmd.Stderr = logFile
			}
			client.Cmd = cmd
			client.logFile = logFile
			if startErr := cmd.Start(); startErr != nil {
				mcpLogger("[MCP] Connect SSE: failed to start process: %v", startErr)
				return fmt.Errorf("sse unreachable (%w) and start process failed: %v", err, startErr)
			}
			mcpLogger("[MCP] Connect SSE: process started (PID=%d, log=%s)", cmd.Process.Pid, logPath)
			time.Sleep(2 * time.Second)
			return nil
		}
		resp.Body.Close()
		mcpLogger("[MCP] Connect SSE: endpoint reachable (status=%d)", resp.StatusCode)
		return nil
	}
	return fmt.Errorf("unknown connection type: %s", client.Config.Type)
}

func (m *MCPManager) connectStdio(client *mcpClient) error {
	cmd := exec.Command(client.Config.Command, client.Config.Args...)
	client.Cmd = cmd
	return cmd.Start()
}

// StartProcess 启动后台进程 (SSE 模式专用)
func (m *MCPManager) StartProcess(id string) error {
	m.mu.RLock()
	client, ok := m.clients[id]
	m.mu.RUnlock()
	if !ok {
		return fmt.Errorf("connection %s not found", id)
	}
	client.mu.Lock()
	defer client.mu.Unlock()

	if client.Config.Type != "sse" {
		return fmt.Errorf("StartProcess only for SSE type, got %s", client.Config.Type)
	}
	if client.Cmd != nil && client.Cmd.Process != nil {
		return fmt.Errorf("process already running (PID: %d)", client.Cmd.Process.Pid)
	}
	cmd := exec.Command(client.Config.Command, client.Config.Args...)
	logPath := mcpLogPath(client.ID)
	logFile, openErr := os.Create(logPath)
	if openErr != nil {
		mcpLogger("[MCP] StartProcess: failed to create log file %s: %v", logPath, openErr)
	} else {
		cmd.Stdout = logFile
		cmd.Stderr = logFile
	}
	client.Cmd = cmd
	client.logFile = logFile
	return cmd.Start()
}

// StopProcess 停止后台进程 (SSE 模式专用)
func (m *MCPManager) StopProcess(id string) error {
	m.mu.RLock()
	client, ok := m.clients[id]
	m.mu.RUnlock()
	if !ok {
		return fmt.Errorf("connection %s not found", id)
	}
	return m.killProcess(client)
}

func (m *MCPManager) killProcess(client *mcpClient) error {
	client.mu.Lock()
	defer client.mu.Unlock()
	if client.Cmd != nil && client.Cmd.Process != nil {
		_ = client.Cmd.Process.Kill()
		client.Cmd = nil
	}
	if client.logFile != nil {
		client.logFile.Close()
		client.logFile = nil
	}
	return nil
}

func (m *MCPManager) Disconnect(id string) error {
	m.mu.RLock()
	client, ok := m.clients[id]
	m.mu.RUnlock()
	if !ok {
		return fmt.Errorf("connection %s not found", id)
	}
	client.mu.Lock()
	defer client.mu.Unlock()
	if client.Config.Type == "stdio" && client.Cmd != nil && client.Cmd.Process != nil {
		return client.Cmd.Process.Kill()
	}
	return nil
}

type mcpSession struct {
	baseURL   string
	sessionID string
	reqID     int
	client    *http.Client
}

func newMCPSession(baseURL string) *mcpSession {
	return &mcpSession{
		baseURL: baseURL,
		client:  &http.Client{Timeout: 15 * time.Second},
	}
}

func (s *mcpSession) send(method string, params interface{}) (json.RawMessage, error) {
	s.reqID++
	req := jsonRPCRequest{
		JSONRPC: "2.0",
		ID:      s.reqID,
		Method:  method,
		Params:  params,
	}
	bodyBytes, _ := json.Marshal(req)
	mcpLogger("[MCP] send: POST %s id=%d method=%s", s.baseURL, s.reqID, method)

	httpReq, err := http.NewRequest("POST", s.baseURL, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json, text/event-stream")
	if s.sessionID != "" {
		httpReq.Header.Set("Mcp-Session-Id", s.sessionID)
	}

	resp, err := s.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("http post: %w", err)
	}
	defer resp.Body.Close()

	if newID := resp.Header.Get("Mcp-Session-Id"); newID != "" {
		s.sessionID = newID
		mcpLogger("[MCP] send: got session ID: %s", s.sessionID)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}
	mcpLogger("[MCP] send: status=%d body=%s", resp.StatusCode, string(body))

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("http %d: %s", resp.StatusCode, string(body))
	}

	var rpcResp jsonRPCResponse
	if err := json.Unmarshal(body, &rpcResp); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}
	if rpcResp.Error != nil {
		return nil, fmt.Errorf("mcp error %d: %s", rpcResp.Error.Code, rpcResp.Error.Message)
	}
	return rpcResp.Result, nil
}

func (s *mcpSession) initialize() error {
	params := map[string]any{
		"protocolVersion": "2025-11-25",
		"capabilities":    map[string]any{},
		"clientInfo": map[string]any{
			"name":    "aiassistant-server",
			"version": "1.0.0",
		},
	}
	result, err := s.send("initialize", params)
	if err != nil {
		return fmt.Errorf("initialize: %w", err)
	}
	var initRes struct {
		ProtocolVersion string `json:"protocolVersion"`
		ServerInfo      struct {
			Name    string `json:"name"`
			Version string `json:"version"`
		} `json:"serverInfo"`
	}
	if err := json.Unmarshal(result, &initRes); err != nil {
		return fmt.Errorf("parse initialize result: %w", err)
	}
	mcpLogger("[MCP] initialized: server=%s v%s protocol=%s", initRes.ServerInfo.Name, initRes.ServerInfo.Version, initRes.ProtocolVersion)
	return nil
}

func (s *mcpSession) notifyInitialized() {
	_, err := s.send("notifications/initialized", nil)
	if err != nil {
		mcpLogger("[MCP] notifyInitialized warning: %v", err)
	}
}

func (s *mcpSession) listTools() ([]MCPTool, error) {
	result, err := s.send("tools/list", nil)
	if err != nil {
		return nil, fmt.Errorf("tools/list: %w", err)
	}
	var toolsList struct {
		Tools []MCPTool `json:"tools"`
	}
	if err := json.Unmarshal(result, &toolsList); err != nil {
		return nil, fmt.Errorf("parse tools/list result: %w", err)
	}
	mcpLogger("[MCP] listTools: got %d tools", len(toolsList.Tools))
	for _, t := range toolsList.Tools {
		mcpLogger("[MCP] listTools:   %s - %s", t.Name, t.Description)
	}
	return toolsList.Tools, nil
}

type jsonRPCRequest struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      int         `json:"id,omitempty"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params,omitempty"`
}

type jsonRPCResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      int             `json:"id"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

func (m *MCPManager) ListTools(id string) ([]MCPTool, error) {
	m.mu.RLock()
	client, ok := m.clients[id]
	m.mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("connection %s not found", id)
	}
	return m.fetchTools(client)
}

func (m *MCPManager) fetchTools(client *mcpClient) ([]MCPTool, error) {
	var baseURL string
	if client.Config.Type == "stdio" {
		return m.fetchToolsStdio(client)
	}
	baseURL = client.Config.SSEURL
	baseURL = strings.TrimSuffix(baseURL, "/sse")
	baseURL = strings.TrimSuffix(baseURL, "/")
	baseURL += "/mcp"

	mcpLogger("[MCP] fetchTools: derived MCP endpoint %s from SSE URL %s", baseURL, client.Config.SSEURL)

	client.mu.Lock()
	if client.session == nil {
		session := newMCPSession(baseURL)
		if err := session.initialize(); err != nil {
			client.mu.Unlock()
			mcpLogger("[MCP] fetchTools: initialize failed: %v", err)
			return nil, err
		}
		session.notifyInitialized()
		time.Sleep(100 * time.Millisecond)
		client.session = session
	}
	client.mu.Unlock()

	tools, err := client.session.listTools()
	if err != nil {
		mcpLogger("[MCP] fetchTools: listTools failed: %v", err)
		return nil, err
	}
	return tools, nil
}

func (m *MCPManager) fetchToolsStdio(client *mcpClient) ([]MCPTool, error) {
	if client.Cmd == nil || client.Cmd.Process == nil {
		return nil, fmt.Errorf("stdio process not running for %s", client.ID)
	}

	stdin, err := client.Cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("stdin pipe: %w", err)
	}
	stdout, err := client.Cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("stdout pipe: %w", err)
	}

	// initialize
	initReq := jsonRPCRequest{JSONRPC: "2.0", ID: 1, Method: "initialize", Params: map[string]any{
		"protocolVersion": "0.1.0",
		"capabilities":    map[string]any{},
		"clientInfo":      map[string]any{"name": "aiassistant-server", "version": "1.0.0"},
	}}
	body, _ := json.Marshal(initReq)
	if _, err := stdin.Write(append(body, '\n')); err != nil {
		return nil, fmt.Errorf("write initialize: %w", err)
	}

	scanner := bufio.NewScanner(stdout)
	scanner.Scan()
	// ignore init response

	// notify initialized
	notifyReq := jsonRPCRequest{JSONRPC: "2.0", Method: "notifications/initialized"}
	body, _ = json.Marshal(notifyReq)
	stdin.Write(append(body, '\n'))

	time.Sleep(100 * time.Millisecond)

	// tools/list
	toolsReq := jsonRPCRequest{JSONRPC: "2.0", ID: 2, Method: "tools/list"}
	body, _ = json.Marshal(toolsReq)
	if _, err := stdin.Write(append(body, '\n')); err != nil {
		return nil, fmt.Errorf("write tools/list: %w", err)
	}

	done := make(chan struct{})
	var toolsList struct {
		Tools []MCPTool `json:"tools"`
	}
	var parseErr error

	go func() {
		defer close(done)
		for scanner.Scan() {
			line := scanner.Bytes()
			var rpcResp jsonRPCResponse
			if err := json.Unmarshal(line, &rpcResp); err != nil {
				continue
			}
			if rpcResp.ID == 2 {
				if rpcResp.Error != nil {
					parseErr = fmt.Errorf("mcp error %d: %s", rpcResp.Error.Code, rpcResp.Error.Message)
					return
				}
				if err := json.Unmarshal(rpcResp.Result, &toolsList); err != nil {
					parseErr = fmt.Errorf("parse tools: %w", err)
				}
				return
			}
		}
		parseErr = fmt.Errorf("no response from stdio mcp")
	}()

	select {
	case <-done:
	case <-time.After(10 * time.Second):
		return nil, fmt.Errorf("stdio timeout")
	}

	return toolsList.Tools, parseErr
}

func (m *MCPManager) RemoveConnection(id string) error {
	_ = m.killProcessByID(id)
	_ = m.Disconnect(id)
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.clients, id)
	return nil
}

func (m *MCPManager) killProcessByID(id string) error {
	m.mu.RLock()
	client, ok := m.clients[id]
	m.mu.RUnlock()
	if !ok {
		return nil
	}
	return m.killProcess(client)
}

func (m *MCPManager) CallTool(id, toolName string, args json.RawMessage) (map[string]interface{}, error) {
	return m.callToolOnClient(id, toolName, args)
}

func (m *MCPManager) CallToolByName(toolName string, argsJSON string, toolNameToConn map[string]string) map[string]interface{} {
	connID, ok := toolNameToConn[toolName]
	if !ok {
		return map[string]interface{}{"error": "no mcp connection known for tool: " + toolName}
	}
	result, err := m.callToolOnClientRaw(connID, toolName, argsJSON)
	if err != nil {
		return map[string]interface{}{"error": err.Error()}
	}
	return result
}

func (m *MCPManager) callToolOnClientRaw(id, toolName, argsJSON string) (map[string]interface{}, error) {
	var args json.RawMessage
	if argsJSON != "" {
		args = json.RawMessage(argsJSON)
	}
	return m.callToolOnClient(id, toolName, args)
}

func (m *MCPManager) callToolOnClient(id, toolName string, args json.RawMessage) (map[string]interface{}, error) {
	m.mu.RLock()
	client, ok := m.clients[id]
	m.mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("connection %s not found", id)
	}

	if client.Config.Type == "stdio" {
		return m.callToolStdio(client, toolName, args)
	}

	client.mu.Lock()
	session := client.session
	if session == nil {
		baseURL := client.Config.SSEURL
		baseURL = strings.TrimSuffix(baseURL, "/sse")
		baseURL = strings.TrimSuffix(baseURL, "/")
		baseURL += "/mcp"

		session = newMCPSession(baseURL)
		if err := session.initialize(); err != nil {
			client.mu.Unlock()
			return nil, err
		}
		session.notifyInitialized()
		time.Sleep(100 * time.Millisecond)
		client.session = session
	}
	client.mu.Unlock()

	argsStr := string(args)
	mcpLogger("[MCP] callToolOnClient: id=%s tool=%s args=%s", id, toolName, argsStr)

	var params interface{}
	if argsStr == "" || argsStr == "null" {
		params = map[string]interface{}{}
	} else {
		if err := json.Unmarshal(args, &params); err != nil {
			mcpLogger("[MCP] callToolOnClient: unmarshal args failed: %v, raw=%s", err, argsStr)
			params = map[string]interface{}{}
		}
	}
	mcpLogger("[MCP] callToolOnClient: params type=%T value=%v", params, params)

	result, err := session.send("tools/call", map[string]interface{}{
		"name":      toolName,
		"arguments": params,
	})
	if err != nil {
		return nil, err
	}

	var res map[string]interface{}
	json.Unmarshal(result, &res)
	return res, nil
}

func (m *MCPManager) callToolStdio(client *mcpClient, toolName string, args json.RawMessage) (map[string]interface{}, error) {
	// Simplified: return not implemented for stdio
	return map[string]interface{}{"status": "not_implemented", "tool": toolName}, nil
}

// GetProcessStatus 返回进程是否在运行 (SSE 模式)
func (m *MCPManager) GetProcessStatus(id string) (bool, int) {
	m.mu.RLock()
	client, ok := m.clients[id]
	m.mu.RUnlock()
	if !ok {
		return false, 0
	}
	client.mu.Lock()
	defer client.mu.Unlock()
	if client.Cmd != nil && client.Cmd.Process != nil {
		return true, client.Cmd.Process.Pid
	}
	return false, 0
}

// ShutdownAll 程序退出时清理所有子进程
func (m *MCPManager) ShutdownAll() {
	m.mu.RLock()
	ids := make([]string, 0, len(m.clients))
	for id := range m.clients {
		ids = append(ids, id)
	}
	m.mu.RUnlock()
	for _, id := range ids {
		_ = m.killProcessByID(id)
	}
}
