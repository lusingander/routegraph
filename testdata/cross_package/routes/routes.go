package routes

import "github.com/labstack/echo/v4"

func Register(g *echo.Group) {
	g.GET("/users", listUsers)
}

func listUsers(c echo.Context) error {
	return nil
}
