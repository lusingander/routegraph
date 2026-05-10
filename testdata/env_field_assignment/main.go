package main

import "github.com/labstack/echo/v4"

type Router struct {
	api *echo.Group
}

func Register(e *echo.Echo) {
	router := &Router{}
	router.api = e.Group("/api")
	router.Register()
}

func (r *Router) Register() {
	r.api.GET("/users", users)
}

func users(c echo.Context) error {
	return nil
}
