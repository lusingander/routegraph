package main

import "github.com/labstack/echo/v4"

type route struct {
	Methods []string
	Path    string
	Handler echo.HandlerFunc
}

type Router struct {
	api    *echo.Group
	routes []route
}

var userRoutes = []route{
	{Methods: []string{"GET", "POST"}, Path: "/users", Handler: users},
	{Methods: []string{"DELETE"}, Path: "/users/:id", Handler: deleteUser},
}

func Register(e *echo.Echo) {
	router := &Router{
		api:    e.Group("/api"),
		routes: userRoutes,
	}
	router.Register()
}

func (r *Router) Register() {
	for _, route := range r.routes {
		r.api.Match(route.Methods, route.Path, route.Handler)
	}
}

func users(c echo.Context) error {
	return nil
}

func deleteUser(c echo.Context) error {
	return nil
}
