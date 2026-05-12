package handler

import (
	"net/http"

	"aiass/src/database"

	"github.com/labstack/echo/v4"
)

func ListUserPrompts(c echo.Context) error {
	rows, err := database.DB.Query("SELECT id, title, content, created_at, updated_at FROM user_prompts ORDER BY updated_at DESC")
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	defer rows.Close()

	type Prompt struct {
		ID        string `json:"id"`
		Title     string `json:"title"`
		Content   string `json:"content"`
		CreatedAt string `json:"created_at"`
		UpdatedAt string `json:"updated_at"`
	}
	var prompts []Prompt
	for rows.Next() {
		var p Prompt
		if err := rows.Scan(&p.ID, &p.Title, &p.Content, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}
		prompts = append(prompts, p)
	}
	return c.JSON(http.StatusOK, prompts)
}

func CreateUserPrompt(c echo.Context) error {
	var body struct {
		Title   string `json:"title"`
		Content string `json:"content"`
	}
	if err := c.Bind(&body); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid body"})
	}
	if body.Title == "" || body.Content == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "title and content required"})
	}
	_, err := database.DB.Exec(
		`INSERT INTO user_prompts (id, title, content) VALUES (hex(randomblob(16)), ?, ?)`,
		body.Title, body.Content,
	)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}

func UpdateUserPrompt(c echo.Context) error {
	id := c.Param("id")
	var body struct {
		Title   string `json:"title"`
		Content string `json:"content"`
	}
	if err := c.Bind(&body); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid body"})
	}
	_, err := database.DB.Exec(
		`UPDATE user_prompts SET title=?, content=?, updated_at=CURRENT_TIMESTAMP WHERE id=?`,
		body.Title, body.Content, id,
	)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}

func DeleteUserPrompt(c echo.Context) error {
	id := c.Param("id")
	_, err := database.DB.Exec("DELETE FROM user_prompts WHERE id=?", id)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}
