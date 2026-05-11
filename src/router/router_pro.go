//go:build !pro

package router

import (
	"github.com/labstack/echo/v4"
)

func SetupProRoutes(e *echo.Echo) {
	// OSS 版: 不注册任何 pro 路由
}
