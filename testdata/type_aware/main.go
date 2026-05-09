package main

import "github.com/labstack/echo/v4"

type fakeRouter struct{}

func (fakeRouter) Group(path string) fakeRouter {
	return fakeRouter{}
}

func (fakeRouter) GET(path string, handler any) {
}

func Register(e *echo.Echo) {
	fake := fakeRouter{}
	api := fake.Group("/fake")
	api.GET("/users", fakeHandler)

	e.GET("/ok", realHandler)
}

func fakeHandler() {
}

func realHandler(c echo.Context) error {
	return nil
}
