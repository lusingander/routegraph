package main

import "github.com/labstack/echo/v4"

type Router struct {
	api *echo.Group
}

func Register(e *echo.Echo) {
	api := e.Group("/api")
	router := NewRouter(api)
	router.Register()
}

func NewRouter(g *echo.Group) *Router {
	return &Router{api: g}
}

func (r *Router) Register() {
	r.api.GET("/users", listUsers)
}

func listUsers(c echo.Context) error {
	return nil
}
