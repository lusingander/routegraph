package main

import "github.com/labstack/echo/v4"

type Router struct {
	api *echo.Group
}

func Register(e *echo.Echo) {
	NewRouter(e).Register()
}

func NewRouter(e *echo.Echo) *Router {
	return &Router{api: e.Group("/api")}
}

func (r *Router) Register() {
	r.api.GET("/users", listUsers)
}

func listUsers(c echo.Context) error {
	return nil
}
