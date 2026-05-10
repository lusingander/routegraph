package main

import "github.com/labstack/echo/v4"

type Router struct {
	api *echo.Group
}

func Register(e *echo.Echo) {
	users := &Router{api: e.Group("/users")}
	admin := &Router{api: e.Group("/admin")}

	users.RegisterUsers()
	admin.RegisterStats()
}

func (r *Router) RegisterUsers() {
	r.api.GET("", users)
}

func (r *Router) RegisterStats() {
	r.api.GET("/stats", stats)
}

func users(c echo.Context) error {
	return nil
}

func stats(c echo.Context) error {
	return nil
}
