package router

import (
	"app/src/config"
	"app/src/service"
	"app/src/validation"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

func Routes(app *fiber.App, db *gorm.DB) {
	validate := validation.Validator()

	healthCheckService := service.NewHealthCheckService(db)
	emailService := service.NewEmailService()
	userService := service.NewUserService(db, validate)
	tokenService := service.NewTokenService(db, validate, userService)
	authService := service.NewAuthService(db, validate, userService, tokenService)
	roleService := service.NewRoleService(db, validate)

	// IMS Services
	productService := service.NewProductService(db, validate)
	batchService := service.NewBatchService(db, validate)
	txService := service.NewTransactionService(db, validate)
	dashboardService := service.NewDashboardService(db)

	v1 := app.Group("/v1")

	HealthCheckRoutes(v1, healthCheckService)
	AuthRoutes(v1, authService, userService, tokenService, emailService)
	UserRoutes(v1, userService, tokenService)
	RoleRoutes(v1, roleService, userService)

	// IMS Routes registration
	IMSRoutes(v1, userService, productService, batchService, txService, dashboardService, db)

	if !config.IsProd {
		DocsRoutes(v1)
	}
}
