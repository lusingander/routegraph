package main

import "github.com/labstack/echo/v4"

const getMethod = "GET"

func Register(e *echo.Echo, method string) {
	e.Any("/health", health)

	api := e.Group("/api")
	api.Add(getMethod, "/users", listUsers)
	api.Add(method, "/dynamic", dynamicHandler)
}

func health(c echo.Context) error {
	return nil
}

func listUsers(c echo.Context) error {
	return nil
}

func dynamicHandler(c echo.Context) error {
	return nil
}
