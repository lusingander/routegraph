package main

import "github.com/labstack/echo/v4"

func Register(e *echo.Echo, path string, method string) {
	api := e.Group("/api")
	api.GET(path, dynamicHandler)
	api.Add(method, path, dynamicAddHandler)
}

func dynamicHandler(c echo.Context) error {
	return nil
}

func dynamicAddHandler(c echo.Context) error {
	return nil
}
