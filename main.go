package main

import (
	"log"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"aiass/src/config"
	"aiass/src/database"
	"aiass/src/router"
	"aiass/src/service"

	"github.com/labstack/echo/v4"
)

func main() {
	os.MkdirAll("./data", 0755)
	cfg := config.Load()

	if err := database.Init(cfg.DBPath); err != nil {
		log.Fatalf("Database init failed: %v", err)
	}
	defer database.Close()

	if err := service.InitMCPManagerFromDB(); err != nil {
		log.Printf("[Server] MCP manager init warning: %v", err)
	}

	// Auto-connect all MCP services sequentially
	go func() {
		time.Sleep(1 * time.Second)
		service.AutoConnectAll()
	}()

	e := echo.New()
	e.HideBanner = true
	e.Debug = false
	e.HidePort = true

	router.Setup(e)

	e.Static("/assets", "./static/assets")
	e.Static("/static", "./static")

	// 根路径 → index.html
	e.GET("/", func(c echo.Context) error {
		return c.File("./static/index.html")
	})

	// SPA fallback — 先尝试匹配路由，失败则返回 index.html
	e.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			err := next(c)
			if err != nil {
				path := c.Request().URL.Path
				isAPI := strings.HasPrefix(path, "/api/")
				isAssets := strings.HasPrefix(path, "/assets/")
				isStatic := strings.HasPrefix(path, "/static/")
				if !isAPI && !isAssets && !isStatic {
					return c.File("./static/index.html")
				}
			}
			return err
		}
	})

	addr := ":" + cfg.Port
	log.Printf("[Server] Starting on %s", addr)

	go openBrowser("http://localhost" + addr)

	defer service.Manager.ShutdownAll()

	if err := e.Start(addr); err != nil {
		log.Fatalf("Server start failed: %v", err)
	}
}

func openBrowser(url string) {
	time.Sleep(2 * time.Second)
	var cmd string
	var args []string
	switch runtime.GOOS {
	case "windows":
		cmd = "cmd"
		args = []string{"/c", "start", url}
	case "darwin":
		cmd = "open"
		args = []string{url}
	default:
		cmd = "xdg-open"
		args = []string{url}
	}
	if err := exec.Command(cmd, args...).Start(); err != nil {
		log.Printf("[Server] Failed to open browser: %v", err)
	}
}
