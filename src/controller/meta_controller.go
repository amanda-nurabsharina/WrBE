package controller

import (
	"app/src/model"
	"app/src/response"
	"fmt"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
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

type CreateWarehouseReq struct {
	Code string `json:"code"`
	Name string `json:"name"`
}

func (ctrl *MetaController) CreateWarehouse(c *fiber.Ctx) error {
	var req CreateWarehouseReq
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid request body")
	}
	if req.Code == "" || req.Name == "" {
		return fiber.NewError(fiber.StatusBadRequest, "Code and Name are required")
	}

	var count int64
	ctrl.DB.WithContext(c.Context()).Model(&model.Warehouse{}).Where("code = ?", req.Code).Count(&count)
	if count > 0 {
		return fiber.NewError(fiber.StatusBadRequest, "Warehouse code already exists")
	}

	wh := model.Warehouse{
		Code: req.Code,
		Name: req.Name,
	}

	if err := ctrl.DB.WithContext(c.Context()).Create(&wh).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to create warehouse")
	}

	return c.Status(fiber.StatusCreated).JSON(response.SuccessWithData[model.Warehouse]{
		Code:    fiber.StatusCreated,
		Status:  "success",
		Message: "Warehouse created successfully",
		Data:    wh,
	})
}

func (ctrl *MetaController) UpdateWarehouse(c *fiber.Ctx) error {
	idStr := c.Params("id")
	whID, err := uuid.Parse(idStr)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid Warehouse ID")
	}

	var wh model.Warehouse
	if err := ctrl.DB.WithContext(c.Context()).First(&wh, "id = ?", whID).Error; err != nil {
		return fiber.NewError(fiber.StatusNotFound, "Warehouse not found")
	}

	var req CreateWarehouseReq
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid request body")
	}

	if req.Code != "" {
		wh.Code = req.Code
	}
	if req.Name != "" {
		wh.Name = req.Name
	}

	if err := ctrl.DB.WithContext(c.Context()).Save(&wh).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to update warehouse")
	}

	return c.Status(fiber.StatusOK).JSON(response.SuccessWithData[model.Warehouse]{
		Code:    fiber.StatusOK,
		Status:  "success",
		Message: "Warehouse updated successfully",
		Data:    wh,
	})
}

func (ctrl *MetaController) DeleteWarehouse(c *fiber.Ctx) error {
	idStr := c.Params("id")
	whID, err := uuid.Parse(idStr)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid Warehouse ID")
	}

	if err := ctrl.DB.WithContext(c.Context()).Delete(&model.Warehouse{}, "id = ?", whID).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to delete warehouse")
	}

	return c.Status(fiber.StatusOK).JSON(response.SuccessWithData[interface{}]{
		Code:    fiber.StatusOK,
		Status:  "success",
		Message: "Warehouse deleted successfully",
		Data:    nil,
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

type CreateLocationReq struct {
	WarehouseID string  `json:"warehouse_id"`
	Aisle       string  `json:"aisle"`
	Rack        string  `json:"rack"`
	Shelf       string  `json:"shelf"`
	Bin         string  `json:"bin"`
	MaxWeight   float64 `json:"max_weight"`
	MaxVolume   float64 `json:"max_volume"`
}

func (ctrl *MetaController) CreateLocation(c *fiber.Ctx) error {
	var req CreateLocationReq
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid request body")
	}
	if req.Rack == "" {
		return fiber.NewError(fiber.StatusBadRequest, "Rack name is required")
	}

	whID, _ := uuid.Parse(req.WarehouseID)
	if whID == uuid.Nil {
		var firstWh model.Warehouse
		ctrl.DB.WithContext(c.Context()).First(&firstWh)
		whID = firstWh.ID
	}

	// Generate structured barcode for location
	var locCount int64
	ctrl.DB.WithContext(c.Context()).Model(&model.Location{}).Count(&locCount)
	barcodeStr := fmt.Sprintf("LOC-R%02d-S%02d", locCount+1, 1)
	if req.Aisle != "" && req.Rack != "" {
		barcodeStr = fmt.Sprintf("LOC-%s-%s", req.Aisle, req.Rack)
	}

	loc := model.Location{
		WarehouseID: whID,
		Aisle:       req.Aisle,
		Rack:        req.Rack,
		Shelf:       req.Shelf,
		Bin:         req.Bin,
		MaxWeight:   req.MaxWeight,
		MaxVolume:   req.MaxVolume,
		Barcode:     barcodeStr,
	}

	if err := ctrl.DB.WithContext(c.Context()).Create(&loc).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to create location: "+err.Error())
	}

	// Register in BarcodeRegistry
	reg := model.BarcodeRegistry{
		Barcode:     loc.Barcode,
		Type:        "LOCATION",
		ReferenceID: loc.ID,
	}
	ctrl.DB.WithContext(c.Context()).Create(&reg)

	return c.Status(fiber.StatusCreated).JSON(response.SuccessWithData[model.Location]{
		Code:    fiber.StatusCreated,
		Status:  "success",
		Message: "Location created successfully",
		Data:    loc,
	})
}

func (ctrl *MetaController) UpdateLocation(c *fiber.Ctx) error {
	idStr := c.Params("id")
	locID, err := uuid.Parse(idStr)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid Location ID")
	}

	var loc model.Location
	if err := ctrl.DB.WithContext(c.Context()).First(&loc, "id = ?", locID).Error; err != nil {
		return fiber.NewError(fiber.StatusNotFound, "Location not found")
	}

	var req CreateLocationReq
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid request body")
	}

	if req.Aisle != "" {
		loc.Aisle = req.Aisle
	}
	if req.Rack != "" {
		loc.Rack = req.Rack
	}
	if req.Shelf != "" {
		loc.Shelf = req.Shelf
	}
	if req.Bin != "" {
		loc.Bin = req.Bin
	}
	if req.MaxWeight > 0 {
		loc.MaxWeight = req.MaxWeight
	}
	if req.MaxVolume > 0 {
		loc.MaxVolume = req.MaxVolume
	}

	if err := ctrl.DB.WithContext(c.Context()).Save(&loc).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to update location")
	}

	return c.Status(fiber.StatusOK).JSON(response.SuccessWithData[model.Location]{
		Code:    fiber.StatusOK,
		Status:  "success",
		Message: "Location updated successfully",
		Data:    loc,
	})
}

func (ctrl *MetaController) DeleteLocation(c *fiber.Ctx) error {
	idStr := c.Params("id")
	locID, err := uuid.Parse(idStr)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid Location ID")
	}

	if err := ctrl.DB.WithContext(c.Context()).Delete(&model.Location{}, "id = ?", locID).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to delete location")
	}

	return c.Status(fiber.StatusOK).JSON(response.SuccessWithData[interface{}]{
		Code:    fiber.StatusOK,
		Status:  "success",
		Message: "Location deleted successfully",
		Data:    nil,
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
