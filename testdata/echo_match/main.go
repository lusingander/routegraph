package main

import "github.com/labstack/echo/v4"

func Register(e *echo.Echo) {
	api := e.Group("/api")
	methods := []string{"GET", "POST"}
	path := "/users"

	api.Match(methods, path, users)
	api.Match([]string{"PUT", "PATCH"}, "/users/:id", updateUser)
}

func users(c echo.Context) error {
	return nil
}

func updateUser(c echo.Context) error {
	return nil
}
