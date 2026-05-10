package main

import "github.com/labstack/echo/v4"

type Config struct {
	API *echo.Group
}

type Router struct {
	api *echo.Group
}

func Register(e *echo.Echo) {
	router := NewRouter(Config{API: e.Group("/api")})
	router.Register()
}

func NewRouter(cfg Config) *Router {
	return &Router{api: cfg.API}
}

func (r *Router) Register() {
	r.api.GET("/users", users)
}

func users(c echo.Context) error {
	return nil
}
