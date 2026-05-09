package main

import "github.com/labstack/echo/v4"

func Register(e *echo.Echo) {
	api := e.Group("/api")
	if api != nil {
		registerUsers(api)
	} else {
		api.GET("/fallback", fallback)
	}
	for i := 0; i < 1; i++ {
		api.GET("/health", health)
	}
}

func registerUsers(g *echo.Group) {
	g.GET("/users", listUsers)
}

func listUsers(c echo.Context) error {
	return nil
}

func fallback(c echo.Context) error {
	return nil
}

func health(c echo.Context) error {
	return nil
}
