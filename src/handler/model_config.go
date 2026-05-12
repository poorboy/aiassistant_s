package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"

	"aiass/src/database"
	"aiass/src/model"

	"github.com/labstack/echo/v4"
)

func ListModelConfigs(c echo.Context) error {
	rows, err := database.DB.Query(
		"SELECT id, provider, name, model, base_url, api_key, proxy_url, is_active, created_at, updated_at FROM model_configs ORDER BY provider, name",
	)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	defer rows.Close()

	var configs []model.ModelConfig
	for rows.Next() {
		var mc model.ModelConfig
		if err := rows.Scan(&mc.ID, &mc.Provider, &mc.Name, &mc.Model, &mc.BaseURL, &mc.APIKey, &mc.ProxyURL, &mc.IsActive, &mc.CreatedAt, &mc.UpdatedAt); err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}
		configs = append(configs, mc)
	}
	return c.JSON(http.StatusOK, configs)
}

func CreateModelConfig(c echo.Context) error {
	var body struct {
		Provider string `json:"provider"`
		Name     string `json:"name"`
		Model    string `json:"model"`
		BaseURL  string `json:"base_url"`
		APIKey   string `json:"api_key"`
		ProxyURL string `json:"proxy_url"`
	}
	if err := c.Bind(&body); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid body"})
	}
	if body.Provider == "" || body.Name == "" || body.Model == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "provider, name, model required"})
	}

	_, err := database.DB.Exec(
		`INSERT INTO model_configs (id, provider, name, model, base_url, api_key, proxy_url, is_active)
		 VALUES (hex(randomblob(16)), ?, ?, ?, ?, ?, ?, 0)`,
		body.Provider, body.Name, body.Model, body.BaseURL, body.APIKey, body.ProxyURL,
	)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	var mc model.ModelConfig
	database.DB.QueryRow(
		"SELECT id, provider, name, model, base_url, api_key, proxy_url, is_active, created_at, updated_at FROM model_configs ORDER BY created_at DESC LIMIT 1",
	).Scan(&mc.ID, &mc.Provider, &mc.Name, &mc.Model, &mc.BaseURL, &mc.APIKey, &mc.ProxyURL, &mc.IsActive, &mc.CreatedAt, &mc.UpdatedAt)
	return c.JSON(http.StatusOK, mc)
}

func UpdateModelConfig(c echo.Context) error {
	id := c.Param("id")
	var body struct {
		Provider string `json:"provider"`
		Name     string `json:"name"`
		Model    string `json:"model"`
		BaseURL  string `json:"base_url"`
		APIKey   string `json:"api_key"`
		ProxyURL string `json:"proxy_url"`
	}
	if err := c.Bind(&body); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid body"})
	}

	_, err := database.DB.Exec(
		`UPDATE model_configs SET provider=?, name=?, model=?, base_url=?, api_key=?, proxy_url=?, updated_at=CURRENT_TIMESTAMP WHERE id=?`,
		body.Provider, body.Name, body.Model, body.BaseURL, body.APIKey, body.ProxyURL, id,
	)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}

func DeleteModelConfig(c echo.Context) error {
	id := c.Param("id")
	_, err := database.DB.Exec("DELETE FROM model_configs WHERE id=?", id)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}

func SetActiveModelConfig(c echo.Context) error {
	id := c.Param("id")
	tx, err := database.DB.Begin()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	tx.Exec("UPDATE model_configs SET is_active=0")
	_, err = tx.Exec("UPDATE model_configs SET is_active=1, updated_at=CURRENT_TIMESTAMP WHERE id=?", id)
	if err != nil {
		tx.Rollback()
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	tx.Commit()
	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}

func TestModelConfig(c echo.Context) error {
	id := c.Param("id")
	var mc model.ModelConfig
	err := database.DB.QueryRow(
		"SELECT base_url, api_key, model, proxy_url FROM model_configs WHERE id=?", id,
	).Scan(&mc.BaseURL, &mc.APIKey, &mc.Model, &mc.ProxyURL)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"status": "error", "message": "model config not found"})
	}
	if mc.APIKey == "" {
		return c.JSON(http.StatusOK, map[string]string{"status": "error", "message": "API Key 未配置"})
	}

	httpClient := &http.Client{}
	if mc.ProxyURL != "" {
		if proxy, err := url.Parse(mc.ProxyURL); err == nil {
			httpClient.Transport = &http.Transport{Proxy: http.ProxyURL(proxy)}
		}
	}

	reqBody, _ := json.Marshal(map[string]interface{}{
		"model":    mc.Model,
		"messages": []map[string]string{{"role": "user", "content": "ping"}},
		"stream":   false,
	})
	req, err := http.NewRequest("POST", mc.BaseURL+"/v1/chat/completions", bytes.NewReader(reqBody))
	if err != nil {
		return c.JSON(http.StatusOK, map[string]string{"status": "error", "message": err.Error()})
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+mc.APIKey)

	log.Printf("[TestModel] POST %s (model=%s, proxy=%q)", mc.BaseURL+"/v1/chat/completions", mc.Model, mc.ProxyURL)
	resp, err := httpClient.Do(req)
	if err != nil {
		return c.JSON(http.StatusOK, map[string]string{"status": "error", "message": err.Error()})
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)
	log.Printf("[TestModel] status=%d body=%s", resp.StatusCode, string(respBody))
	if resp.StatusCode != 200 {
		return c.JSON(http.StatusOK, map[string]string{"status": "error", "message": fmt.Sprintf("API %d: %s", resp.StatusCode, string(respBody))})
	}
	return c.JSON(http.StatusOK, map[string]string{"status": "ok", "message": "连接成功"})
}
