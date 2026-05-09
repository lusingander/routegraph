package main

import "github.com/labstack/echo/v4"

func Register(e *echo.Echo) {
	e.Static("/assets", "public")
	e.File("/", "public/index.html")
	e.RouteNotFound("/*", notFound)

	api := e.Group("/api")
	api.Static("/docs", "docs")
	api.File("/openapi.json", "openapi.json")
	api.RouteNotFound("/*", apiNotFound)
}

func notFound(c echo.Context) error {
	return nil
}

func apiNotFound(c echo.Context) error {
	return nil
}
