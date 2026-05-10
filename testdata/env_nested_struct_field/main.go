package main

import "github.com/labstack/echo/v4"

type Config struct {
	API *echo.Group
}

type Router struct {
	cfg Config
}

func Register(e *echo.Echo) {
	router := &Router{
		cfg: Config{API: e.Group("/api")},
	}
	router.Register()
}

func (r *Router) Register() {
	r.cfg.API.GET("/users", users)
}

func users(c echo.Context) error {
	return nil
}
