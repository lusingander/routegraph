package main

import "github.com/labstack/echo/v4"

var routes = []struct {
	Methods []string
	Path    string
	Handler echo.HandlerFunc
}{
	{Methods: []string{"GET", "POST"}, Path: "/users", Handler: users},
	{Methods: []string{"DELETE"}, Path: "/users/:id", Handler: deleteUser},
}

func Register(e *echo.Echo) {
	api := e.Group("/api")
	for _, route := range routes {
		api.Match(route.Methods, route.Path, route.Handler)
	}
}

func users(c echo.Context) error {
	return nil
}

func deleteUser(c echo.Context) error {
	return nil
}
