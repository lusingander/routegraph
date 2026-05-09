package main

import "github.com/labstack/echo/v4"

type Router struct {
	api   *echo.Group
	admin *echo.Group
}

func NewRouter(e *echo.Echo) *Router {
	api := e.Group("/api")
	return &Router{
		api:   api,
		admin: api.Group("/admin"),
	}
}

func (r *Router) Register() {
	r.api.GET("/users", listUsers)
	r.admin.GET("/stats", stats)
}

func listUsers(c echo.Context) error {
	return nil
}

func stats(c echo.Context) error {
	return nil
}
