package handler

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"aiass/src/database"
	"aiass/src/service"

	"github.com/labstack/echo/v4"
)

func ChatStream(c echo.Context) error {
	conversationID := c.QueryParam("conversation_id")
	message := c.QueryParam("message")
	promptID := c.QueryParam("prompt_id")
	enabledToolsParam := c.QueryParam("enabled_tools")

	if conversationID == "" || message == "" {
		return c.String(http.StatusBadRequest, "conversation_id and message required")
	}

	enabledConnIDs := map[string]bool{}
	for _, id := range strings.Split(enabledToolsParam, ",") {
		id = strings.TrimSpace(id)
		if id != "" {
			enabledConnIDs[id] = true
		}
	}

	now := time.Now().UnixNano()
	msgID := fmt.Sprintf("msg_%d", now)
	database.DB.Exec("INSERT INTO chat_history (id, conversation_id, role, content) VALUES (?, ?, 'user', ?)",
		msgID, conversationID, message)

	var msgCount int
	database.DB.QueryRow("SELECT message_count FROM conversations WHERE id=?", conversationID).Scan(&msgCount)
	if msgCount == 0 {
		title := message
		if len([]rune(title)) > 30 {
			title = string([]rune(title)[:30]) + "..."
		}
		database.DB.Exec("UPDATE conversations SET title=?, message_count=message_count+1, updated_at=CURRENT_TIMESTAMP WHERE id=?",
			title, conversationID)
	} else {
		database.DB.Exec("UPDATE conversations SET message_count=message_count+1, updated_at=CURRENT_TIMESTAMP WHERE id=?", conversationID)
	}

	settings := loadChatSettings()

	c.Response().Header().Set(echo.HeaderContentType, "text/event-stream")
	c.Response().Header().Set("Cache-Control", "no-cache")
	c.Response().Header().Set("Connection", "keep-alive")

	flusher, ok := c.Response().Writer.(http.Flusher)
	if !ok {
		return c.String(http.StatusInternalServerError, "streaming not supported")
	}

	if settings.apiKey == "" {
		c.Response().Write([]byte("event: message\ndata: {\"type\":\"text\",\"content\":\"错误: 请先在设置中配置 DeepSeek API Key\"}\n\n"))
		flusher.Flush()
		c.Response().Write([]byte("event: message\ndata: {\"type\":\"done\",\"message\":\"done\"}\n\n"))
		flusher.Flush()
		return nil
	}

	client := service.NewDeepSeekClientFromSettings(settings.apiKey, settings.baseURL, settings.model)
	mcpManager := service.GetMCPManager()

	connIDs := []string{"blender", "gimp"}
	if len(enabledConnIDs) > 0 {
		connIDs = nil
		for id := range enabledConnIDs {
			connIDs = append(connIDs, id)
		}
	}

	var tools []service.DeepSeekTool
	toolNameToConn := make(map[string]string)
	for _, connID := range connIDs {
		mcpTools, err := mcpManager.ListTools(connID)
		if err != nil {
			continue
		}
		for _, mt := range mcpTools {
			schema, _ := mt.InputSchema.(map[string]interface{})
			tools = append(tools, service.DeepSeekTool{
				Type: "function",
				Function: service.DeepSeekFunc{
					Name:        mt.Name,
					Description: mt.Description,
					Parameters:  schema,
				},
			})
			toolNameToConn[mt.Name] = connID
		}
	}

	// Build system messages describing available MCP capabilities per connection
	var systemMsgs []service.DeepSeekMessage
	if len(tools) > 0 {
		connTools := map[string][]string{}
		for name, connID := range toolNameToConn {
			connTools[connID] = append(connTools[connID], name)
		}

		for _, connID := range connIDs {
			names, ok := connTools[connID]
			if !ok || len(names) == 0 {
				continue
			}
			var prompt string
			switch connID {
			case "blender":
				prompt = "你是 Blender 3D 创作助手。你可以调用以下 Blender MCP 工具来执行 3D 建模、场景操作、材质设置、渲染等任务。\n当你接收到 3D 建模、渲染、场景创建、Blender 相关请求时，必须使用这些工具来完成任务，不要只给建议。\n可用工具: " + strings.Join(names, ", ") + "\n根据需要选择合适的工具并调用。如果工具返回的结果不完整，可以继续调用其他工具来完成整个工作流。"
			case "gimp":
				prompt = "你是 GIMP 图像编辑助手。你可以调用以下 GIMP MCP 工具来执行图像处理、绘画、海报设计、图片编辑等任务。\n当你接收到图像处理、绘画、设计海报、图片编辑、GIMP 相关请求时，必须使用这些工具来完成任务，不要只给建议。\n可用工具: " + strings.Join(names, ", ") + "\n根据需要选择合适的工具并调用。如果工具返回的结果不完整，可以继续调用其他工具来完成整个工作流。"
			default:
				continue
			}
			systemMsgs = append(systemMsgs, service.DeepSeekMessage{Role: "system", Content: prompt})
		}
	}

	// Load user-selected prompt from DB
	if promptID != "" {
		var title, content string
		err := database.DB.QueryRow("SELECT title, content FROM prompts WHERE id=?", promptID).Scan(&title, &content)
		if err == nil && content != "" {
			systemMsgs = append(systemMsgs, service.DeepSeekMessage{Role: "system", Content: content})
		}
	}

	messages := loadConversationHistory(conversationID)
	allMsgs := append(systemMsgs, messages...)
	allMsgs = append(allMsgs, service.DeepSeekMessage{Role: "user", Content: message})

	var fullContent strings.Builder
	maxLoops := 10
	for loop := 0; loop < maxLoops; loop++ {
		var toolCalls []service.DeepSeekToolCall
		_, err := client.ChatStreamWithTools(allMsgs, tools, func(chunk string, calls []service.DeepSeekToolCall) {
			if chunk != "" {
				fullContent.WriteString(chunk)
				escaped := strings.ReplaceAll(chunk, "\n", "\\n")
				c.Response().Write([]byte(fmt.Sprintf("event: message\ndata: {\"type\":\"text\",\"content\":\"%s\"}\n\n", escaped)))
				flusher.Flush()
			}
			if len(calls) > 0 {
				toolCalls = append(toolCalls, calls...)
			}
		})

		if err != nil {
			errorMsg := strings.ReplaceAll(err.Error(), "\n", "\\n")
			c.Response().Write([]byte(fmt.Sprintf("event: message\ndata: {\"type\":\"text\",\"content\":\"错误: %s\"}\n\n", errorMsg)))
			flusher.Flush()
			break
		}

		if len(toolCalls) == 0 {
			break
		}

		assistantMsg := service.DeepSeekMessage{Role: "assistant", Content: fullContent.String(), ToolCalls: toolCalls}
		allMsgs = append(allMsgs, assistantMsg)

		for _, tc := range toolCalls {
			// Skip if tool name is empty (likely broken stream from DeepSeek)
			if tc.Function.Name == "" {
				continue
			}
			
			c.Response().Write([]byte(fmt.Sprintf("event: message\ndata: {\"type\":\"tool_start\",\"tool\":\"%s\",\"args\":%s}\n\n", tc.Function.Name, tc.Function.Arguments)))
			flusher.Flush()

			result := mcpManager.CallToolByName(tc.Function.Name, tc.Function.Arguments, toolNameToConn)
			resultJSON, _ := json.Marshal(result)
			allMsgs = append(allMsgs, service.DeepSeekMessage{
				Role:       "tool",
				ToolCallID: tc.ID,
				Content:    string(resultJSON),
			})

			resultStr := fmt.Sprintf("%v", result)
			c.Response().Write([]byte(fmt.Sprintf("event: message\ndata: {\"type\":\"tool_result\",\"tool\":\"%s\",\"message\":\"%s\"}\n\n", tc.Function.Name, strings.ReplaceAll(resultStr, "\"", "\\\""))))
			flusher.Flush()
		}
		fullContent.Reset()
	}

	assistantID := fmt.Sprintf("msg_%d", time.Now().UnixNano())
	database.DB.Exec("INSERT INTO chat_history (id, conversation_id, role, content) VALUES (?, ?, 'assistant', ?)",
		assistantID, conversationID, fullContent.String())
	autoUpdateConversationTitle(conversationID, fullContent.String())

	c.Response().Write([]byte("event: message\ndata: {\"type\":\"done\",\"message\":\"done\"}\n\n"))
	flusher.Flush()
	return nil
}

type chatSettings struct {
	apiKey  string
	baseURL string
	model   string
}

func loadChatSettings() chatSettings {
	s := chatSettings{
		baseURL: "https://api.deepseek.com",
		model:   "deepseek-v4-flash",
	}
	rows, err := database.DB.Query("SELECT key, value FROM settings WHERE key IN ('deepseek.api_key','deepseek.base_url','deepseek.model')")
	if err != nil {
		return s
	}
	defer rows.Close()
	for rows.Next() {
		var k, v string
		rows.Scan(&k, &v)
		switch k {
		case "deepseek.api_key":
			s.apiKey = v
		case "deepseek.base_url":
			if v != "" {
				s.baseURL = v
			}
		case "deepseek.model":
			if v != "" {
				s.model = v
			}
		}
	}
	return s
}

func loadConversationHistory(conversationID string) []service.DeepSeekMessage {
	rows, err := database.DB.Query(
		"SELECT role, content FROM chat_history WHERE conversation_id=? AND role IN ('user','assistant','tool') ORDER BY created_at ASC",
		conversationID,
	)
	if err != nil {
		return nil
	}
	defer rows.Close()

	var msgs []service.DeepSeekMessage
	for rows.Next() {
		var m service.DeepSeekMessage
		rows.Scan(&m.Role, &m.Content)
		msgs = append(msgs, m)
	}
	return msgs
}

func ListConversations(c echo.Context) error {
	rows, err := database.DB.Query(
		"SELECT id, title, message_count, created_at, updated_at FROM conversations ORDER BY updated_at DESC",
	)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	defer rows.Close()

	type Conv struct {
		ID           string `json:"id"`
		Title        string `json:"title"`
		MessageCount int    `json:"message_count"`
		CreatedAt    string `json:"created_at"`
		UpdatedAt    string `json:"updated_at"`
	}
	var convs []Conv
	for rows.Next() {
		var cv Conv
		if err := rows.Scan(&cv.ID, &cv.Title, &cv.MessageCount, &cv.CreatedAt, &cv.UpdatedAt); err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}
		convs = append(convs, cv)
	}
	return c.JSON(http.StatusOK, convs)
}

func CreateConversation(c echo.Context) error {
	_, err := database.DB.Exec(
		"INSERT INTO conversations (id, title) VALUES (hex(randomblob(16)), '新会话')",
	)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	var id, title string
	database.DB.QueryRow("SELECT id, title FROM conversations ORDER BY created_at DESC LIMIT 1").Scan(&id, &title)
	return c.JSON(http.StatusOK, map[string]string{"id": id, "title": title})
}

func UpdateConversation(c echo.Context) error {
	id := c.Param("id")
	var body struct {
		Title string `json:"title"`
	}
	if err := c.Bind(&body); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid body"})
	}
	if body.Title == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "title required"})
	}
	_, err := database.DB.Exec("UPDATE conversations SET title=?, updated_at=CURRENT_TIMESTAMP WHERE id=?", body.Title, id)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}

func DeleteConversation(c echo.Context) error {
	id := c.Param("id")
	database.DB.Exec("DELETE FROM chat_history WHERE conversation_id=?", id)
	_, err := database.DB.Exec("DELETE FROM conversations WHERE id=?", id)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}

func GetChatMessages(c echo.Context) error {
	conversationID := c.QueryParam("conversation_id")
	if conversationID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "conversation_id required"})
	}
	rows, err := database.DB.Query(
		"SELECT id, role, content, tool_calls, created_at FROM chat_history WHERE conversation_id=? ORDER BY created_at ASC",
		conversationID,
	)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	defer rows.Close()

	type Msg struct {
		ID        string `json:"id"`
		Role      string `json:"role"`
		Content   string `json:"content"`
		ToolCalls string `json:"tool_calls"`
		CreatedAt string `json:"created_at"`
	}
	var msgs []Msg
	for rows.Next() {
		var m Msg
		var toolCalls, createdAt sql.NullString
		if err := rows.Scan(&m.ID, &m.Role, &m.Content, &toolCalls, &createdAt); err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}
		m.ToolCalls = toolCalls.String
		m.CreatedAt = createdAt.String
		msgs = append(msgs, m)
	}
	return c.JSON(http.StatusOK, msgs)
}

// Legacy — kept for compat
func GetChatHistory(c echo.Context) error {
	return ListConversations(c)
}

func ClearChatHistory(c echo.Context) error {
	database.DB.Exec("DELETE FROM chat_history")
	_, err := database.DB.Exec("DELETE FROM conversations")
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return c.NoContent(http.StatusOK)
}

func TestDeepSeek(c echo.Context) error {
	settings := loadChatSettings()
	if settings.apiKey == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"status": "error", "message": "API Key 未配置"})
	}
	client := service.NewDeepSeekClientFromSettings(settings.apiKey, settings.baseURL, settings.model)
	if err := client.TestConnection(); err != nil {
		return c.JSON(http.StatusOK, map[string]string{"status": "error", "message": err.Error()})
	}
	return c.JSON(http.StatusOK, map[string]string{"status": "ok", "message": "连接成功"})
}

// Called after full response is generated to auto-update title
func autoUpdateConversationTitle(conversationID, fullContent string) {
	if conversationID == "" || fullContent == "" {
		return
	}
	// Only update title if it's still the default
	var title string
	err := database.DB.QueryRow("SELECT title FROM conversations WHERE id=?", conversationID).Scan(&title)
	if err != nil || !strings.HasPrefix(title, "新会话") {
		return
	}
	newTitle := service.TrimString(fullContent, 50)
	if newTitle != "" {
		database.DB.Exec("UPDATE conversations SET title=?, updated_at=CURRENT_TIMESTAMP WHERE id=?", newTitle, conversationID)
	}
}
