package main

import (
	"github.com/labstack/echo/v4"
	"github.com/lusingander/routegraph/testdata/cross_package/routes"
)

func Register(e *echo.Echo) {
	api := e.Group("/api")
	routes.Register(api)
}
