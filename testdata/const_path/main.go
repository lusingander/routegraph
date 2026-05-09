package main

import "github.com/labstack/echo/v4"

const apiPrefix = "/api"
const usersPath = "/users"

func Register(e *echo.Echo) {
	const version = "/v1"
	const adminPath = "/admin"

	api := e.Group(apiPrefix)
	v1 := api.Group(version)

	v1.GET(usersPath, listUsers)
	v1.GET(usersPath+"/:id", getUser)
	admin := v1.Group(adminPath)
	admin.POST("/stats", createStat)
}

func listUsers(c echo.Context) error {
	return nil
}

func getUser(c echo.Context) error {
	return nil
}

func createStat(c echo.Context) error {
	return nil
}
