package main

import "github.com/labstack/echo/v4"

func registerUsers(g *echo.Group) {
	g.GET("/users", listUsers)
}

func listUsers(c echo.Context) error {
	return nil
}
