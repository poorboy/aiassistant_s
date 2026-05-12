package router

import (
	"aiass/src/handler"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

func Setup(e *echo.Echo) {
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: []string{"*"},
		AllowMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
	}))

	e.GET("/health", handler.HealthCheck)

	e.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			c.Response().Header().Set("X-Edition", "oss")
			return next(c)
		}
	})

	api := e.Group("/api")
	{
		chat := api.Group("/chat")
		chat.GET("/stream", handler.ChatStream)
		chat.GET("/history", handler.GetChatHistory)
		chat.DELETE("/history", handler.ClearChatHistory)
		chat.GET("/messages", handler.GetChatMessages)
		chat.GET("/conversations", handler.ListConversations)
		chat.POST("/conversations", handler.CreateConversation)
		chat.PUT("/conversations/:id", handler.UpdateConversation)
		chat.DELETE("/conversations/:id", handler.DeleteConversation)

		api.POST("/deepseek/test", handler.TestDeepSeek)

		mcp := api.Group("/mcp/connections")
		mcp.GET("", handler.ListMCPConnections)
		mcp.POST("/:id/connect", handler.ConnectMCP)
		mcp.POST("/:id/disconnect", handler.DisconnectMCP)
		mcp.GET("/:id/tools", handler.ListMCPTools)
		mcp.POST("/:id/start-process", handler.StartMCPProcess)
		mcp.POST("/:id/stop-process", handler.StopMCPProcess)
		mcp.GET("/:id/process-status", handler.GetMCPProcessStatus)
		mcp.POST("/:id/call-tool", handler.CallMCPTool)
		mcp.GET("/:id/logs", handler.GetMCPLogs)

		api.GET("/settings", handler.GetSettings)
		api.PUT("/settings", handler.UpdateSettings)

		modelCfgs := api.Group("/model-configs")
		modelCfgs.GET("", handler.ListModelConfigs)
		modelCfgs.POST("", handler.CreateModelConfig)
		modelCfgs.PUT("/:id", handler.UpdateModelConfig)
		modelCfgs.DELETE("/:id", handler.DeleteModelConfig)
		modelCfgs.POST("/:id/activate", handler.SetActiveModelConfig)
		modelCfgs.POST("/:id/test", handler.TestModelConfig)

		api.GET("/prompts", handler.ListPrompts)
		api.POST("/prompts", handler.CreatePrompt)
		api.PUT("/prompts/:id", handler.UpdatePrompt)
		api.DELETE("/prompts/:id", handler.DeletePrompt)

		api.GET("/webhook-settings", handler.GetWebhookSettings)
		api.PUT("/webhook-settings", handler.UpdateWebhookSettings)
		api.POST("/webhook-test", handler.TestWebhook)

		userPrompt := api.Group("/user-prompts")
		userPrompt.GET("", handler.ListUserPrompts)
		userPrompt.POST("", handler.CreateUserPrompt)
		userPrompt.PUT("/:id", handler.UpdateUserPrompt)
		userPrompt.DELETE("/:id", handler.DeleteUserPrompt)
	}
}
