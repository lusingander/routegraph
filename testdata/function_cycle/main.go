package main

import "github.com/labstack/echo/v4"

func Register(e *echo.Echo) {
	api := e.Group("/api")
	registerA(api)
}

func registerA(g *echo.Group) {
	g.GET("/a", handlerA)
	registerB(g)
}

func registerB(g *echo.Group) {
	g.GET("/b", handlerB)
	registerA(g)
}

func handlerA(c echo.Context) error {
	return nil
}

func handlerB(c echo.Context) error {
	return nil
}
