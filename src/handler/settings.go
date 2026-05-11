package handler

import (
	"database/sql"
	"net/http"

	"aiass/src/database"

	"github.com/labstack/echo/v4"
)

func GetSettings(c echo.Context) error {
	rows, err := database.DB.Query("SELECT key, value, updated_at FROM settings")
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	defer rows.Close()

	settings := make(map[string]string)
	for rows.Next() {
		var key, value string
		var updatedAt sql.NullString
		if err := rows.Scan(&key, &value, &updatedAt); err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}
		settings[key] = value
	}
	return c.JSON(http.StatusOK, settings)
}

func UpdateSettings(c echo.Context) error {
	var body map[string]string
	if err := c.Bind(&body); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid body"})
	}
	for key, value := range body {
		_, err := database.DB.Exec(
			`INSERT INTO settings (key, value, updated_at) VALUES (?, ?, CURRENT_TIMESTAMP)
			 ON CONFLICT(key) DO UPDATE SET value=excluded.value, updated_at=CURRENT_TIMESTAMP`,
			key, value,
		)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}
	}
	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}
