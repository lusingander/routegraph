package main

import "github.com/labstack/echo/v4"

type Router struct {
	api *echo.Group
}

func Register(e *echo.Echo) {
	NewRouter(e.Group("/admin")).RegisterAdmin()
}

func NewRouter(g *echo.Group) *Router {
	return &Router{api: g}
}

func (r *Router) RegisterAdmin() {
	r.api.GET("/stats", stats)
}

func stats(c echo.Context) error {
	return nil
}
