package main

import "github.com/labstack/echo/v4"

func Register(e *echo.Echo) {
	api := e.Group("/api")
	withGroup(api, func(g *echo.Group) {
		g.GET("/users", listUsers)
	})
}

func withGroup(g *echo.Group, register func(*echo.Group)) {
	register(g)
}

func listUsers(c echo.Context) error {
	return nil
}
