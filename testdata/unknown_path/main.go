package main

import "github.com/labstack/echo/v4"

func Register(e *echo.Echo, prefix string) {
	api := e.Group(prefix)
	api.GET("/users", listUsers)
}

func listUsers(c echo.Context) error {
	return nil
}
