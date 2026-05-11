package handler

import (
	"net/http"

	"aiass/src/database"

	"github.com/labstack/echo/v4"
)

func ListPrompts(c echo.Context) error {
	rows, err := database.DB.Query("SELECT id, title, content, category, created_at, updated_at FROM prompts ORDER BY category ASC, title ASC")
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	defer rows.Close()

	type Prompt struct {
		ID        string `json:"id"`
		Title     string `json:"title"`
		Content   string `json:"content"`
		Category  string `json:"category"`
		CreatedAt string `json:"created_at"`
		UpdatedAt string `json:"updated_at"`
	}
	var prompts []Prompt
	for rows.Next() {
		var p Prompt
		if err := rows.Scan(&p.ID, &p.Title, &p.Content, &p.Category, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}
		prompts = append(prompts, p)
	}
	return c.JSON(http.StatusOK, prompts)
}

func CreatePrompt(c echo.Context) error {
	var body struct {
		Title    string `json:"title"`
		Content  string `json:"content"`
		Category string `json:"category"`
	}
	if err := c.Bind(&body); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid body"})
	}
	_, err := database.DB.Exec(
		`INSERT INTO prompts (id, title, content, category) VALUES (hex(randomblob(16)), ?, ?, ?)`,
		body.Title, body.Content, body.Category,
	)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}

func UpdatePrompt(c echo.Context) error {
	id := c.Param("id")
	var body struct {
		Title    string `json:"title"`
		Content  string `json:"content"`
		Category string `json:"category"`
	}
	if err := c.Bind(&body); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid body"})
	}
	_, err := database.DB.Exec(
		`UPDATE prompts SET title=?, content=?, category=?, updated_at=CURRENT_TIMESTAMP WHERE id=?`,
		body.Title, body.Content, body.Category, id,
	)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}

func DeletePrompt(c echo.Context) error {
	id := c.Param("id")
	_, err := database.DB.Exec("DELETE FROM prompts WHERE id=?", id)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}
