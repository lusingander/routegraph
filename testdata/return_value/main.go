package main

import "github.com/labstack/echo/v4"

type Router struct {
	api *echo.Group
}

func Register(e *echo.Echo) {
	fromLocal := NewRouterFromLocal(e.Group("/api"))
	fromLocal.RegisterUsers()

	NewRouterFromCall(e.Group("/admin")).RegisterStats()
}

func NewRouterFromLocal(g *echo.Group) *Router {
	router := &Router{api: g}
	return router
}

func NewRouterFromCall(g *echo.Group) *Router {
	return wrapRouter(g)
}

func wrapRouter(g *echo.Group) *Router {
	return &Router{api: g}
}

func (r *Router) RegisterUsers() {
	r.api.GET("/users", listUsers)
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
