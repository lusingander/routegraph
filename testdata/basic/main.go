package main

import "github.com/labstack/echo/v4"

func Register(e *echo.Echo) {
	api := e.Group("/api")
	v1 := api.Group("/v1")

	v1.GET("/users", listUsers)
	v1.POST("/users", createUser)

	admin := v1.Group("/admin")
	admin.GET("/stats", h.Stats)
}

func listUsers(c echo.Context) error {
	return nil
}

func createUser(c echo.Context) error {
	return nil
}

var h handlers

type handlers struct{}

func (handlers) Stats(c echo.Context) error {
	return nil
}
