package main

import "github.com/labstack/echo/v4"

func Register(e *echo.Echo) {
	var api = e.Group("/api")
	api.GET("/users", listUsers)

	api = e.Group("/v2")
	api.POST("/users", createUser)

	e.Group("/chained").GET("/health", health)
}

func RegisterLocal() {
	e := echo.New()
	e.GET("/local", localHandler)
}

func listUsers(c echo.Context) error {
	return nil
}

func createUser(c echo.Context) error {
	return nil
}

func health(c echo.Context) error {
	return nil
}

func localHandler(c echo.Context) error {
	return nil
}
