package main

import "github.com/labstack/echo/v4"

type Router struct{}

func Register(e *echo.Echo) {
	api := e.Group("/api")
	router := &Router{}
	router.RegisterUsers(api)
}

func (r *Router) RegisterUsers(g *echo.Group) {
	g.GET("/users", listUsers)
	r.registerAdmin(g.Group("/admin"))
}

func (r *Router) registerAdmin(g *echo.Group) {
	g.GET("/stats", stats)
}

func listUsers(c echo.Context) error {
	return nil
}

func stats(c echo.Context) error {
	return nil
}
