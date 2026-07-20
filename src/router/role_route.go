package router

import (
	"app/src/controller"
	m "app/src/middleware"
	"app/src/service"

	"github.com/gofiber/fiber/v2"
)

func RoleRoutes(v1 fiber.Router, roleService service.RoleService, userService service.UserService) {
	roleController := controller.NewRoleController(roleService)

	roles := v1.Group("/admin/roles")

	roles.Get("/", m.Auth(userService), roleController.ListRoles)
	roles.Post("/", m.Auth(userService, "manageRoles"), roleController.CreateRole)
	roles.Get("/:id", m.Auth(userService, "manageRoles"), roleController.GetRoleByID)
	roles.Put("/:id", m.Auth(userService, "manageRoles"), roleController.UpdateRole)
	roles.Delete("/:id", m.Auth(userService, "manageRoles"), roleController.DeleteRole)
}
