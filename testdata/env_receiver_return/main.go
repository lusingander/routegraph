package main

import "github.com/labstack/echo/v4"

type Factory struct {
	api *echo.Group
}

type Router struct {
	api *echo.Group
}

func Register(e *echo.Echo) {
	NewFactory(e.Group("/api")).Router().Register()
}

func NewFactory(g *echo.Group) *Factory {
	return &Factory{api: g}
}

func (f *Factory) Router() *Router {
	return &Router{api: f.api}
}

func (r *Router) Register() {
	r.api.GET("/users", users)
}

func users(c echo.Context) error {
	return nil
}
