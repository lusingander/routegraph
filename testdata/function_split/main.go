package main

import "github.com/labstack/echo/v4"

func Register(e *echo.Echo) {
	api := e.Group("/api")
	registerUsers(api)
}

func registerUsers(g *echo.Group) {
	g.GET("/users", listUsers)
	registerCreate(g)
}

func registerCreate(g *echo.Group) {
	g.POST("/users", createUser)
}

func listUsers(c echo.Context) error {
	return nil
}

func createUser(c echo.Context) error {
	return nil
}
