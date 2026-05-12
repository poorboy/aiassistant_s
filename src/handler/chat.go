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
	modelConfigID := c.QueryParam("model_config_id")
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

	// If a model_config_id is provided, use its settings instead
	if modelConfigID != "" {
		var mc struct {
			provider, apiKey, baseURL, model, proxyURL string
		}
		err := database.DB.QueryRow(
			"SELECT provider, api_key, base_url, model, proxy_url FROM model_configs WHERE id=?",
			modelConfigID,
		).Scan(&mc.provider, &mc.apiKey, &mc.baseURL, &mc.model, &mc.proxyURL)
		if err == nil && mc.apiKey != "" {
			settings.apiKey = mc.apiKey
			settings.baseURL = mc.baseURL
			settings.model = mc.model
			settings.proxyURL = mc.proxyURL
		}
	}

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

	client := service.NewDeepSeekClientFromSettings(settings.apiKey, settings.baseURL, settings.model, settings.proxyURL)
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
	var reasoningContent string
	maxLoops := 10
	for loop := 0; loop < maxLoops; loop++ {
		reasoningContent = ""
		var toolCalls []service.DeepSeekToolCall
		_, err := client.ChatStreamWithTools(allMsgs, tools, func(chunk string, calls []service.DeepSeekToolCall) {
			if chunk != "" {
				fullContent.WriteString(chunk)
				textData, _ := json.Marshal(map[string]string{
					"type":    "text",
					"content": chunk,
				})
				c.Response().Write([]byte(fmt.Sprintf("event: message\ndata: %s\n\n", string(textData))))
				flusher.Flush()
			}
			if len(calls) > 0 {
				toolCalls = append(toolCalls, calls...)
			}
		}, &reasoningContent)

		if err != nil {
			errorData, _ := json.Marshal(map[string]string{
				"type":    "text",
				"content": "错误: " + err.Error(),
			})
			c.Response().Write([]byte(fmt.Sprintf("event: message\ndata: %s\n\n", string(errorData))))
			flusher.Flush()
			break
		}

		if len(toolCalls) == 0 {
			break
		}

		assistantMsg := service.DeepSeekMessage{Role: "assistant", Content: fullContent.String(), ToolCalls: toolCalls, ReasoningContent: reasoningContent}
		allMsgs = append(allMsgs, assistantMsg)

		for _, tc := range toolCalls {
			// Skip if tool name is empty (likely broken stream from DeepSeek)
			if tc.Function.Name == "" {
				continue
			}
			
			// Build tool_start event safely using JSON marshaling to avoid SSE-breaking characters
			toolStartData, _ := json.Marshal(map[string]interface{}{
				"type": "tool_start",
				"tool": tc.Function.Name,
				"args": tc.Function.Arguments,
			})
			c.Response().Write([]byte(fmt.Sprintf("event: message\ndata: %s\n\n", string(toolStartData))))
			flusher.Flush()

			result := mcpManager.CallToolByName(tc.Function.Name, tc.Function.Arguments, toolNameToConn)
			// Extract text from MCP tool result
			toolResultContent := extractMCPResultText(result)
			resultJSON, _ := json.Marshal(toolResultContent)
			allMsgs = append(allMsgs, service.DeepSeekMessage{
				Role:       "tool",
				ToolCallID: tc.ID,
				Content:    string(resultJSON),
			})

			// Build tool_result event safely: use json.Marshal for the message field
			toolResultData, _ := json.Marshal(map[string]interface{}{
				"type":    "tool_result",
				"tool":    tc.Function.Name,
				"message": toolResultContent,
			})
			c.Response().Write([]byte(fmt.Sprintf("event: message\ndata: %s\n\n", string(toolResultData))))
			flusher.Flush()
		}
		fullContent.Reset()
	}

	assistantID := fmt.Sprintf("msg_%d", time.Now().UnixNano())
	database.DB.Exec("INSERT INTO chat_history (id, conversation_id, role, content) VALUES (?, ?, 'assistant', ?)",
		assistantID, conversationID, fullContent.String())
	autoUpdateConversationTitle(conversationID, fullContent.String())

	// Compute and update token count from all messages in this conversation
	tokenCount := computeConversationTokens(conversationID)
	database.DB.Exec("UPDATE conversations SET token_count=?, prompt_id=?, updated_at=CURRENT_TIMESTAMP WHERE id=?",
		tokenCount, promptID, conversationID)

	c.Response().Write([]byte("event: message\ndata: {\"type\":\"done\",\"message\":\"done\"}\n\n"))
	flusher.Flush()

	// Send webhook notification after task completes
	go sendWebhookNotification(conversationID, message, fullContent.String())

	return nil
}

type chatSettings struct {
	apiKey   string
	baseURL  string
	model    string
	proxyURL string
}

// extractMCPResultText extracts human-readable text from an MCP tool result.
// MCP results have format: {"content":[{"type":"text","text":"..."}]}
// If the content structure is recognized, combine all text items.
// Otherwise return the JSON representation of the full result.
func extractMCPResultText(result map[string]interface{}) string {
	if result == nil {
		return ""
	}
	// Try to extract from MCP content array format
	if content, ok := result["content"]; ok {
		if contentArr, ok := content.([]interface{}); ok && len(contentArr) > 0 {
			var texts []string
			for _, item := range contentArr {
				if itemMap, ok := item.(map[string]interface{}); ok {
					if itemMap["type"] == "text" {
						if txt, ok := itemMap["text"].(string); ok {
							texts = append(texts, txt)
						}
					}
				}
			}
			if len(texts) > 0 {
				return strings.Join(texts, "\n")
			}
		}
	}
	// If error field present, return error message
	if errMsg, ok := result["error"]; ok {
		if errStr, ok := errMsg.(string); ok {
			return errStr
		}
	}
	// Fallback: JSON marshal the result
	b, _ := json.Marshal(result)
	return string(b)
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

// estimateTokens approximates token count from text length.
// 1 token ≈ 4 ASCII chars or ≈ 1.5 CJK chars.
func estimateTokens(text string) int {
	if text == "" {
		return 0
	}
	cjk, ascii := 0, 0
	for _, r := range text {
		if r >= 0x4E00 && r <= 0x9FFF || r >= 0x3040 && r <= 0x30FF || r >= 0xAC00 && r <= 0xD7AF {
			cjk++
		} else {
			ascii++
		}
	}
	return (cjk*2 + ascii/2) / 3 // ceil(cjk/1.5 + ascii/4)
}

func computeConversationTokens(conversationID string) int {
	rows, err := database.DB.Query("SELECT role, content FROM chat_history WHERE conversation_id=?", conversationID)
	if err != nil {
		return 0
	}
	defer rows.Close()
	total := 0
	for rows.Next() {
		var role, content string
		rows.Scan(&role, &content)
		total += estimateTokens(content)
	}
	return total
}

func ListConversations(c echo.Context) error {
	rows, err := database.DB.Query(
		"SELECT id, title, message_count, token_count, prompt_id, created_at, updated_at FROM conversations ORDER BY updated_at DESC",
	)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	defer rows.Close()

	type Conv struct {
		ID           string `json:"id"`
		Title        string `json:"title"`
		MessageCount int    `json:"message_count"`
		TokenCount   int    `json:"token_count"`
		PromptID     string `json:"prompt_id"`
		CreatedAt    string `json:"created_at"`
		UpdatedAt    string `json:"updated_at"`
	}
	var convs []Conv
	for rows.Next() {
		var cv Conv
		if err := rows.Scan(&cv.ID, &cv.Title, &cv.MessageCount, &cv.TokenCount, &cv.PromptID, &cv.CreatedAt, &cv.UpdatedAt); err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}
		convs = append(convs, cv)
	}
	return c.JSON(http.StatusOK, convs)
}

func CreateConversation(c echo.Context) error {
	var body struct {
		Title    string `json:"title"`
		PromptID string `json:"prompt_id"`
	}
	c.Bind(&body)
	title := body.Title
	if title == "" {
		title = "新会话"
	}
	promptID := body.PromptID
	_, err := database.DB.Exec(
		"INSERT INTO conversations (id, title, prompt_id) VALUES (hex(randomblob(16)), ?, ?)",
		title, promptID,
	)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	var id, titleRet, promptIDRet string
	var tokenCount int
	database.DB.QueryRow("SELECT id, title, token_count, prompt_id FROM conversations ORDER BY created_at DESC LIMIT 1").Scan(&id, &titleRet, &tokenCount, &promptIDRet)
	return c.JSON(http.StatusOK, map[string]interface{}{"id": id, "title": titleRet, "token_count": tokenCount, "prompt_id": promptIDRet})
}

func UpdateConversation(c echo.Context) error {
	id := c.Param("id")
	var body struct {
		Title    string `json:"title"`
		PromptID string `json:"prompt_id"`
	}
	if err := c.Bind(&body); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid body"})
	}
	if body.Title != "" {
		database.DB.Exec("UPDATE conversations SET title=?, updated_at=CURRENT_TIMESTAMP WHERE id=?", body.Title, id)
	}
	if body.PromptID != "" || c.Request().Method == "PUT" { // allow clearing
		database.DB.Exec("UPDATE conversations SET prompt_id=?, updated_at=CURRENT_TIMESTAMP WHERE id=?", body.PromptID, id)
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
	client := service.NewDeepSeekClientFromSettings(settings.apiKey, settings.baseURL, settings.model, settings.proxyURL)
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
