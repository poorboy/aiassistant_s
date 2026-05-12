package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"aiass/src/database"

	"github.com/labstack/echo/v4"
)

type webhookSettings struct {
	url      string
	keywords string
}

func loadWebhookSettings() webhookSettings {
	s := webhookSettings{}
	rows, err := database.DB.Query("SELECT key, value FROM settings WHERE key IN ('webhook.url','webhook.keywords')")
	if err != nil {
		return s
	}
	defer rows.Close()
	for rows.Next() {
		var k, v string
		rows.Scan(&k, &v)
		switch k {
		case "webhook.url":
			s.url = v
		case "webhook.keywords":
			s.keywords = v
		}
	}
	return s
}

func shouldNotify(content string, keywords string) bool {
	if keywords == "" {
		return true
	}
	for _, kw := range strings.Split(keywords, ",") {
		kw = strings.TrimSpace(kw)
		if kw != "" && strings.Contains(content, kw) {
			return true
		}
	}
	return false
}

func sendWebhookNotification(conversationID, userMessage, assistantResponse string) {
	ws := loadWebhookSettings()
	if ws.url == "" {
		return
	}
	if !shouldNotify(assistantResponse, ws.keywords) {
		return
	}

	var title string
	database.DB.QueryRow("SELECT title FROM conversations WHERE id=?", conversationID).Scan(&title)
	if title == "" {
		title = conversationID
	}

	payload := map[string]interface{}{
		"msg_type": "interactive",
		"card": map[string]interface{}{
			"header": map[string]interface{}{
				"title": map[string]string{"tag": "plain_text", "content": "AI Assistant 任务完成通知"},
			},
			"elements": []map[string]interface{}{
				{"tag": "markdown", "content": fmt.Sprintf("**会话**: %s", title)},
				{"tag": "markdown", "content": fmt.Sprintf("**用户消息**: %s", truncateText(userMessage, 200))},
				{"tag": "markdown", "content": fmt.Sprintf("**AI 回复**: %s", truncateText(assistantResponse, 500))},
				{"tag": "markdown", "content": fmt.Sprintf("**时间**: %s", time.Now().Format("2006-01-02 15:04:05"))},
			},
		},
	}

	body, _ := json.Marshal(payload)
	http.Post(ws.url, "application/json", bytes.NewReader(body))
}

func truncateText(s string, maxLen int) string {
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	return string(runes[:maxLen]) + "..."
}

func GetWebhookSettings(c echo.Context) error {
	ws := loadWebhookSettings()
	return c.JSON(http.StatusOK, map[string]string{
		"webhook.url":      ws.url,
		"webhook.keywords": ws.keywords,
	})
}

func UpdateWebhookSettings(c echo.Context) error {
	var body struct {
		URL      string `json:"url"`
		Keywords string `json:"keywords"`
	}
	if err := c.Bind(&body); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid body"})
	}
	for key, value := range map[string]string{
		"webhook.url":      body.URL,
		"webhook.keywords": body.Keywords,
	} {
		database.DB.Exec(
			`INSERT INTO settings (key, value, updated_at) VALUES (?, ?, CURRENT_TIMESTAMP)
			 ON CONFLICT(key) DO UPDATE SET value=excluded.value, updated_at=CURRENT_TIMESTAMP`,
			key, value,
		)
	}
	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}

func TestWebhook(c echo.Context) error {
	ws := loadWebhookSettings()
	if ws.url == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Webhook URL 未配置"})
	}

	keywords := ws.keywords
	if keywords == "" {
		keywords = "@pd:,@Msg:,#Tlog:,$TSK:,#MMSG:"
	}
	randomKW := pickRandomKeyword(keywords)

	content := fmt.Sprintf("这是一条测试通知，您的 Webhook 配置已生效。\n\n**测试关键词**: %s\n**时间**: %s", randomKW, time.Now().Format("2006-01-02 15:04:05"))

	payload := map[string]interface{}{
		"msg_type": "interactive",
		"card": map[string]interface{}{
			"header": map[string]interface{}{
				"title": map[string]string{"tag": "plain_text", "content": "AI Assistant 测试通知"},
			},
			"elements": []map[string]interface{}{
				{"tag": "markdown", "content": content},
			},
		},
	}

	body, _ := json.Marshal(payload)
	resp, err := http.Post(ws.url, "application/json", bytes.NewReader(body))
	if err != nil {
		return c.JSON(http.StatusOK, map[string]string{"status": "error", "message": err.Error()})
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return c.JSON(http.StatusOK, map[string]string{"status": "error", "message": "Webhook 返回状态码: " + http.StatusText(resp.StatusCode)})
	}
	return c.JSON(http.StatusOK, map[string]string{"status": "ok", "message": "通知发送成功"})
}

func pickRandomKeyword(keywords string) string {
	parts := strings.Split(keywords, ",")
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}
	var valid []string
	for _, p := range parts {
		if p != "" {
			valid = append(valid, p)
		}
	}
	if len(valid) == 0 {
		return "@pd:"
	}
	return valid[time.Now().UnixNano()%int64(len(valid))]
}
