package controller

import (
	"app/src/response"
	"app/src/service"
	"strings"

	"github.com/gofiber/fiber/v2"
)

type RoleController struct {
	RoleService service.RoleService
}

func NewRoleController(roleService service.RoleService) *RoleController {
	return &RoleController{
		RoleService: roleService,
	}
}

type CreateRoleRequest struct {
	Name            string              `json:"name" validate:"required"`
	DisplayName     string              `json:"display_name" validate:"required"`
	Description     string              `json:"description"`
	AccessibleMenus []string            `json:"accessible_menus"`
	Permissions     map[string][]string `json:"permissions"`
}

type UpdateRoleRequest struct {
	DisplayName     string              `json:"display_name" validate:"required"`
	Description     string              `json:"description"`
	AccessibleMenus []string            `json:"accessible_menus"`
	Permissions     map[string][]string `json:"permissions"`
}

func (h *RoleController) ListRoles(c *fiber.Ctx) error {
	roles, err := h.RoleService.ListRoles(c)
	if err != nil {
		return err
	}

	return c.Status(fiber.StatusOK).JSON(response.SuccessWithData[interface{}]{
		Code:    fiber.StatusOK,
		Status:  "success",
		Message: "Roles retrieved successfully",
		Data:    roles,
	})
}

func (h *RoleController) GetRoleByID(c *fiber.Ctx) error {
	id := c.Params("id")
	role, err := h.RoleService.GetRoleByID(c, id)
	if err != nil {
		return err
	}

	return c.Status(fiber.StatusOK).JSON(response.SuccessWithData[interface{}]{
		Code:    fiber.StatusOK,
		Status:  "success",
		Message: "Role retrieved successfully",
		Data:    role,
	})
}

func (h *RoleController) CreateRole(c *fiber.Ctx) error {
	var req CreateRoleRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	req.Name = strings.TrimSpace(strings.ToLower(req.Name))
	if req.Name == "" {
		return fiber.NewError(fiber.StatusBadRequest, "Role name is required")
	}

	role, err := h.RoleService.CreateRole(c, req.Name, req.DisplayName, req.Description, req.AccessibleMenus, req.Permissions)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	return c.Status(fiber.StatusCreated).JSON(response.SuccessWithData[interface{}]{
		Code:    fiber.StatusCreated,
		Status:  "success",
		Message: "Role created successfully",
		Data:    role,
	})
}

func (h *RoleController) UpdateRole(c *fiber.Ctx) error {
	id := c.Params("id")
	var req UpdateRoleRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	role, err := h.RoleService.UpdateRole(c, id, req.DisplayName, req.Description, req.AccessibleMenus, req.Permissions)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	return c.Status(fiber.StatusOK).JSON(response.SuccessWithData[interface{}]{
		Code:    fiber.StatusOK,
		Status:  "success",
		Message: "Role updated successfully",
		Data:    role,
	})
}

func (h *RoleController) DeleteRole(c *fiber.Ctx) error {
	id := c.Params("id")
	if err := h.RoleService.DeleteRole(c, id); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	return c.Status(fiber.StatusOK).JSON(response.Common{
		Code:    fiber.StatusOK,
		Status:  "success",
		Message: "Role deleted successfully",
	})
}
