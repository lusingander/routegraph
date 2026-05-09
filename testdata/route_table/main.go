package main

import "github.com/labstack/echo/v4"

const usersPath = "/users"

func Register(e *echo.Echo) {
	api := e.Group("/api")
	routes := []struct {
		method  string
		path    string
		handler echo.HandlerFunc
	}{
		{"GET", usersPath, listUsers},
		{method: "POST", path: usersPath, handler: createUser},
		{"GET", "/admin/stats", stats},
	}

	for _, r := range routes {
		api.Add(r.method, r.path, r.handler)
	}
}

func listUsers(c echo.Context) error {
	return nil
}

func createUser(c echo.Context) error {
	return nil
}

func stats(c echo.Context) error {
	return nil
}
