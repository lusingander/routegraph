package main

import "github.com/labstack/echo/v4"

type Router struct {
	api *echo.Group
}

func Register(e *echo.Echo) {
	users := NewRouter(e.Group("/users"))
	admin := NewRouter(e.Group("/admin"))

	users.RegisterUsers()
	admin.RegisterStats()
}

func NewRouter(g *echo.Group) *Router {
	return &Router{api: g}
}

func (r *Router) RegisterUsers() {
	r.api.GET("", listUsers)
}

func (r *Router) RegisterStats() {
	r.api.GET("/stats", stats)
}

func listUsers(c echo.Context) error {
	return nil
}

func stats(c echo.Context) error {
	return nil
}
