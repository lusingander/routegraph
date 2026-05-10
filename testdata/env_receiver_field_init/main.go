package main

import "github.com/labstack/echo/v4"

type Router struct {
	api *echo.Group
}

func Register(e *echo.Echo) {
	router := NewRouter(e.Group("/api"))
	router.Register()
}

func NewRouter(g *echo.Group) *Router {
	router := &Router{}
	router.mount(g)
	return router
}

func (r *Router) mount(g *echo.Group) {
	r.api = g
}

func (r *Router) Register() {
	r.api.GET("/users", users)
}

func users(c echo.Context) error {
	return nil
}
