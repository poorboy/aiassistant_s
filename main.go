package main

import (
	"io"
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
	cfg := config.Load()

	os.MkdirAll(cfg.DataDir, 0755)
	os.MkdirAll(cfg.LogDir, 0755)

	service.InitLogDir()

	if err := database.Init(cfg.DBPath); err != nil {
		log.Fatalf("Database init failed: %v", err)
	}
	defer database.Close()

	if err := service.InitMCPManagerFromDB(); err != nil {
		log.Printf("[Server] MCP manager init warning: %v", err)
	}

	go func() {
		time.Sleep(1 * time.Second)
		service.AutoConnectAll()
	}()

	e := echo.New()
	e.HideBanner = true
	e.Debug = false
	e.HidePort = true
	e.Logger.SetOutput(io.Discard)

	router.Setup(e)

	e.Static("/assets", cfg.StaticDir+"/assets")
	e.Static("/static", cfg.StaticDir)

	e.GET("/", func(c echo.Context) error {
		return c.File(cfg.StaticDir + "/index.html")
	})

	e.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			err := next(c)
			if err != nil {
				path := c.Request().URL.Path
				isAPI := strings.HasPrefix(path, "/api/")
				isAssets := strings.HasPrefix(path, "/assets/")
				isStatic := strings.HasPrefix(path, "/static/")
				if !isAPI && !isAssets && !isStatic {
					return c.File(cfg.StaticDir + "/index.html")
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
