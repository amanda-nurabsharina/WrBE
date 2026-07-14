package controller

import (
	"app/src/model"
	"app/src/response"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

type MetaController struct {
	DB *gorm.DB
}

func NewMetaController(db *gorm.DB) *MetaController {
	return &MetaController{DB: db}
}

func (ctrl *MetaController) GetWarehouses(c *fiber.Ctx) error {
	var list []model.Warehouse
	if err := ctrl.DB.WithContext(c.Context()).Order("code asc").Find(&list).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Database error")
	}
	return c.Status(fiber.StatusOK).JSON(response.SuccessWithData[[]model.Warehouse]{
		Code:    fiber.StatusOK,
		Status:  "success",
		Message: "Warehouses retrieved successfully",
		Data:    list,
	})
}

func (ctrl *MetaController) GetLocations(c *fiber.Ctx) error {
	var list []model.Location
	if err := ctrl.DB.WithContext(c.Context()).Order("rack asc").Find(&list).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Database error")
	}
	return c.Status(fiber.StatusOK).JSON(response.SuccessWithData[[]model.Location]{
		Code:    fiber.StatusOK,
		Status:  "success",
		Message: "Rack locations retrieved successfully",
		Data:    list,
	})
}

func (ctrl *MetaController) GetSuppliers(c *fiber.Ctx) error {
	var list []model.Supplier
	if err := ctrl.DB.WithContext(c.Context()).Order("name asc").Find(&list).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Database error")
	}
	return c.Status(fiber.StatusOK).JSON(response.SuccessWithData[[]model.Supplier]{
		Code:    fiber.StatusOK,
		Status:  "success",
		Message: "Suppliers retrieved successfully",
		Data:    list,
	})
}
