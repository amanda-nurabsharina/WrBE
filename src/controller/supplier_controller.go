package controller

import (
	"app/src/model"
	"app/src/response"
	"app/src/service"
	"app/src/validation"

	"github.com/gofiber/fiber/v2"
)

type SupplierController struct {
	SupplierService service.SupplierService
}

func NewSupplierController(supplierService service.SupplierService) *SupplierController {
	return &SupplierController{
		SupplierService: supplierService,
	}
}

func (ctrl *SupplierController) GetSuppliers(c *fiber.Ctx) error {
	search := c.Query("search", "")
	list, err := ctrl.SupplierService.GetSuppliers(c, search)
	if err != nil {
		return err
	}

	return c.Status(fiber.StatusOK).
		JSON(response.SuccessWithData[[]model.Supplier]{
			Code:    fiber.StatusOK,
			Status:  "success",
			Message: "Suppliers retrieved successfully",
			Data:    list,
		})
}

func (ctrl *SupplierController) GetSupplierByID(c *fiber.Ctx) error {
	id := c.Params("id")
	supplier, err := ctrl.SupplierService.GetSupplierByID(c, id)
	if err != nil {
		return err
	}

	return c.Status(fiber.StatusOK).
		JSON(response.SuccessWithData[*model.Supplier]{
			Code:    fiber.StatusOK,
			Status:  "success",
			Message: "Supplier retrieved successfully",
			Data:    supplier,
		})
}

func (ctrl *SupplierController) CreateSupplier(c *fiber.Ctx) error {
	req := new(validation.CreateSupplier)
	if err := c.BodyParser(req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid request body")
	}

	supplier, err := ctrl.SupplierService.CreateSupplier(c, req)
	if err != nil {
		return err
	}

	return c.Status(fiber.StatusCreated).
		JSON(response.SuccessWithData[*model.Supplier]{
			Code:    fiber.StatusCreated,
			Status:  "success",
			Message: "Supplier created successfully",
			Data:    supplier,
		})
}

func (ctrl *SupplierController) UpdateSupplier(c *fiber.Ctx) error {
	id := c.Params("id")
	req := new(validation.UpdateSupplier)
	if err := c.BodyParser(req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid request body")
	}

	supplier, err := ctrl.SupplierService.UpdateSupplier(c, id, req)
	if err != nil {
		return err
	}

	return c.Status(fiber.StatusOK).
		JSON(response.SuccessWithData[*model.Supplier]{
			Code:    fiber.StatusOK,
			Status:  "success",
			Message: "Supplier updated successfully",
			Data:    supplier,
		})
}

func (ctrl *SupplierController) DeleteSupplier(c *fiber.Ctx) error {
	id := c.Params("id")
	err := ctrl.SupplierService.DeleteSupplier(c, id)
	if err != nil {
		return err
	}

	return c.Status(fiber.StatusOK).
		JSON(response.Common{
			Code:    fiber.StatusOK,
			Status:  "success",
			Message: "Supplier deleted successfully",
		})
}
