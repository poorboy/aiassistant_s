package model

import "time"

type Setting struct {
	Key       string    `json:"key"`
	Value     string    `json:"value"`
	UpdatedAt time.Time `json:"updated_at"`
}

type MCPConnection struct {
	ID         string    `json:"id"`
	Name       string    `json:"name"`
	Type       string    `json:"type"` // stdio | sse
	Command    string    `json:"command"`
	Args       string    `json:"args"`        // JSON array string
	SSEURL     string    `json:"sse_url"`     // SSE mode only
	SSEHeaders string    `json:"sse_headers"` // JSON object string
	Status     string    `json:"status"`      // connected | disconnected
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

type RAGDocument struct {
	ID             string    `json:"id"`
	Filename       string    `json:"filename"`
	Filepath       string    `json:"filepath"`
	RAGVersion     string    `json:"rag_version"`
	Source         string    `json:"source"`
	ChunkCount     int       `json:"chunk_count"`
	Status         string    `json:"status"`
	BlenderVersion string    `json:"blender_version"`
	CreatedAt      time.Time `json:"created_at"`
}

type ChatMessage struct {
	ID        string `json:"id"`
	Role      string `json:"role"`
	Content   string `json:"content"`
	ToolCalls string `json:"tool_calls"`
	CreatedAt string `json:"created_at"`
}

type DeepSeekMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}
