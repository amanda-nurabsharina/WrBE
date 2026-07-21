package router

import (
	"app/src/barcode"
	m "app/src/middleware"
	"app/src/service"

	"github.com/gofiber/fiber/v2"
)

func BarcodeRoutes(v1 fiber.Router, bcController *barcode.Controller, userService service.UserService) {
	auth := m.Auth(userService)

	bc := v1.Group("/barcode", auth)

	bc.Get("/lookup", bcController.Lookup)
	bc.Post("/validate", bcController.Validate)
	bc.Post("/validate-pick", bcController.ValidatePick)
	bc.Post("/scan", bcController.RecordScan)
	bc.Post("/print", bcController.RecordPrint)
	bc.Post("/session", bcController.StartSession)
	bc.Post("/put-away", bcController.PutAway)
}
