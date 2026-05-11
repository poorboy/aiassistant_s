package handler

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"aiass/src/database"
	"aiass/src/service"

	"github.com/labstack/echo/v4"
)

var mcpManager = service.GetMCPManager()

func ListMCPConnections(c echo.Context) error {
	rows, err := database.DB.Query(
		"SELECT id, name, type, command, args, sse_url, status, created_at, updated_at FROM mcp_connections ORDER BY name ASC",
	)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	defer rows.Close()

	type MCPConn struct {
		ID        string `json:"id"`
		Name      string `json:"name"`
		Type      string `json:"type"`
		Command   string `json:"command"`
		Args      string `json:"args"`
		SSEURL    string `json:"sse_url"`
		Status    string `json:"status"`
		CreatedAt string `json:"created_at"`
		UpdatedAt string `json:"updated_at"`
	}
	var conns []MCPConn
	for rows.Next() {
		var conn MCPConn
		if err := rows.Scan(&conn.ID, &conn.Name, &conn.Type, &conn.Command, &conn.Args, &conn.SSEURL, &conn.Status, &conn.CreatedAt, &conn.UpdatedAt); err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}
		conns = append(conns, conn)
	}
	if conns == nil {
		conns = []MCPConn{}
	}
	return c.JSON(http.StatusOK, conns)
}

func ConnectMCP(c echo.Context) error {
	id := c.Param("id")
	log.Printf("[MCP] ConnectMCP: id=%s", id)

	var sseURL, connType, command, args string
	err := database.DB.QueryRow("SELECT sse_url, type, command, args FROM mcp_connections WHERE id=?", id).Scan(&sseURL, &connType, &command, &args)
	if err != nil {
		log.Printf("[MCP] ConnectMCP: connection %s not found in DB", id)
		return c.JSON(http.StatusNotFound, map[string]string{"error": "connection not found"})
	}
	log.Printf("[MCP] ConnectMCP: loaded config type=%s command=%s sse_url=%s", connType, command, sseURL)

	cfg := service.MCPConnConfig{
		ID:      id,
		Type:    connType,
		Command: command,
		Args:    parseArgs(args),
		SSEURL:  sseURL,
	}
	if err := mcpManager.CreateConnection(cfg); err != nil {
		log.Printf("[MCP] ConnectMCP: CreateConnection failed: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	log.Printf("[MCP] ConnectMCP: connection created in manager")
	if err := mcpManager.Connect(id); err != nil {
		log.Printf("[MCP] ConnectMCP: Connect failed: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	log.Printf("[MCP] ConnectMCP: connected successfully")
	database.DB.Exec("UPDATE mcp_connections SET status='connected', updated_at=CURRENT_TIMESTAMP WHERE id=?", id)
	return c.JSON(http.StatusOK, map[string]string{"status": "connected"})
}

func DisconnectMCP(c echo.Context) error {
	id := c.Param("id")
	mcpManager.Disconnect(id)
	mcpManager.RemoveConnection(id)
	database.DB.Exec("UPDATE mcp_connections SET status='disconnected', updated_at=CURRENT_TIMESTAMP WHERE id=?", id)
	return c.JSON(http.StatusOK, map[string]string{"status": "disconnected"})
}

func ListMCPTools(c echo.Context) error {
	id := c.Param("id")
	tools, err := mcpManager.ListTools(id)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	if tools == nil {
		tools = []service.MCPTool{}
	}
	return c.JSON(http.StatusOK, tools)
}

func StartMCPProcess(c echo.Context) error {
	id := c.Param("id")

	var sseURL, command, args string
	err := database.DB.QueryRow("SELECT sse_url, command, args FROM mcp_connections WHERE id=?", id).Scan(&sseURL, &command, &args)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "connection not found"})
	}

	cfg := service.MCPConnConfig{
		ID:      id,
		Type:    "sse",
		Command: command,
		Args:    parseArgs(args),
		SSEURL:  sseURL,
	}
	mcpManager.CreateConnection(cfg)
	if err := mcpManager.StartProcess(id); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	running, pid := mcpManager.GetProcessStatus(id)
	return c.JSON(http.StatusOK, map[string]interface{}{"status": "started", "pid": pid, "running": running})
}

func StopMCPProcess(c echo.Context) error {
	id := c.Param("id")
	mcpManager.StopProcess(id)
	return c.JSON(http.StatusOK, map[string]string{"status": "stopped"})
}

func GetMCPProcessStatus(c echo.Context) error {
	id := c.Param("id")
	running, pid := mcpManager.GetProcessStatus(id)
	return c.JSON(http.StatusOK, map[string]interface{}{"running": running, "pid": pid})
}

type callToolRequest struct {
	ToolName string          `json:"tool_name"`
	Args     json.RawMessage `json:"args"`
}

func CallMCPTool(c echo.Context) error {
	id := c.Param("id")
	var req callToolRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
	}
	result, err := mcpManager.CallTool(id, req.ToolName, req.Args)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, result)
}

func GetMCPLogs(c echo.Context) error {
	id := c.Param("id")
	logDir := filepath.Join("data", "log")
	pattern := filepath.Join(logDir, "mcp-"+id+"-*.log")
	matches, err := filepath.Glob(pattern)
	if err != nil || len(matches) == 0 {
		return c.JSON(http.StatusOK, map[string]interface{}{"lines": []string{}})
	}
	sort.Sort(sort.Reverse(sort.StringSlice(matches)))
	latest := matches[0]
	data, err := os.ReadFile(latest)
	if err != nil {
		return c.JSON(http.StatusOK, map[string]interface{}{"lines": []string{}})
	}
	content := strings.TrimSpace(string(data))
	var lines []string
	if content != "" {
		lines = strings.Split(content, "\n")
	}
	// Keep last 200 lines
	if len(lines) > 200 {
		lines = lines[len(lines)-200:]
	}
	return c.JSON(http.StatusOK, map[string]interface{}{"lines": lines, "file": filepath.Base(latest)})
}

func parseArgs(args string) []string {
	if args == "" {
		return nil
	}
	var result []string
	result = append(result, args)
	return result
}
