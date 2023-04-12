package controllers

import (
	"github.com/labstack/echo/v4"
)

func addServer(matrix matrixService) echo.HandlerFunc {
	return func(c echo.Context) error {
		code := matrix.AddServer(c.Param("name"))
		return c.NoContent(code)
	}
}
