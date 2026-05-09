package main

import "github.com/labstack/echo/v4"

type fakeRouter struct{}

func (fakeRouter) Add(method string, path string, handler any) {
}

func Register(e *echo.Echo) {
	fake := fakeRouter{}
	fake.Add("GET", "/fake", fakeHandler)

	e.GET("/real", realHandler)
}

func fakeHandler() {
}

func realHandler(c echo.Context) error {
	return nil
}
