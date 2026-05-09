package main

import "github.com/labstack/echo/v4"

func Register(e *echo.Echo) {
	api := e.Group("/api")
	registerUsers(api)
	registerAdmins(api)
}

func registerUsers(g *echo.Group) {
	registerShared(g)
	g.GET("/users", listUsers)
}

func registerAdmins(g *echo.Group) {
	registerShared(g)
	g.GET("/admins", listAdmins)
}

func registerShared(g *echo.Group) {
	g.GET("/health", health)
}

func health(c echo.Context) error {
	return nil
}

func listUsers(c echo.Context) error {
	return nil
}

func listAdmins(c echo.Context) error {
	return nil
}
