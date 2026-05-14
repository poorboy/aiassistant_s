package service

import (
	"log"
	"time"

	"aiass/src/config"
	"aiass/src/database"
)

func InitLogDir() {
	cfg := config.Load()
	LogDir = cfg.LogDir
}

var mcpManagerService = Manager

func GetMCPManager() *MCPManager {
	return mcpManagerService
}

func InitMCPManagerFromDB() error {
	rows, err := database.DB.Query("SELECT id, name, type, command, args, sse_url FROM mcp_connections")
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var id, name, mtype, command, args, sseURL string
		if err := rows.Scan(&id, &name, &mtype, &command, &args, &sseURL); err != nil {
			continue
		}
		cfg := MCPConnConfig{
			ID:      id,
			Name:    name,
			Type:    mtype,
			Command: command,
			Args:    parseArgs(args),
			SSEURL:  sseURL,
		}
		Manager.CreateConnection(cfg)
	}
	return nil
}

func AutoConnectAll() {
	for _, id := range []string{"blender", "gimp"} {
		if err := Manager.Connect(id); err != nil {
			log.Printf("[MCP] AutoConnect %s failed: %v", id, err)
		} else {
			log.Printf("[MCP] AutoConnect %s succeeded", id)
		}
		time.Sleep(500 * time.Millisecond)
	}
}

func parseArgs(args string) []string {
	if args == "" {
		return nil
	}
	var result []string
	result = append(result, args)
	return result
}
