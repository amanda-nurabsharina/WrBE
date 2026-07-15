package controller

import (
	"app/src/model"
	"app/src/response"
	"app/src/service"
	"app/src/validation"

	"github.com/gofiber/fiber/v2"
)

type PackagingController struct {
	PackagingService service.PackagingService
}

func NewPackagingController(packagingService service.PackagingService) *PackagingController {
	return &PackagingController{
		PackagingService: packagingService,
	}
}

func (ctrl *PackagingController) GetPackagingUnits(c *fiber.Ctx) error {
	search := c.Query("search", "")
	list, err := ctrl.PackagingService.GetPackagingUnits(c, search)
	if err != nil {
		return err
	}

	return c.Status(fiber.StatusOK).
		JSON(response.SuccessWithData[[]model.PackagingUnit]{
			Code:    fiber.StatusOK,
			Status:  "success",
			Message: "Packaging units retrieved successfully",
			Data:    list,
		})
}

func (ctrl *PackagingController) GetPackagingUnitByID(c *fiber.Ctx) error {
	id := c.Params("id")
	packaging, err := ctrl.PackagingService.GetPackagingUnitByID(c, id)
	if err != nil {
		return err
	}

	return c.Status(fiber.StatusOK).
		JSON(response.SuccessWithData[*model.PackagingUnit]{
			Code:    fiber.StatusOK,
			Status:  "success",
			Message: "Packaging unit retrieved successfully",
			Data:    packaging,
		})
}

func (ctrl *PackagingController) CreatePackagingUnit(c *fiber.Ctx) error {
	req := new(validation.CreatePackagingUnit)
	if err := c.BodyParser(req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid request body")
	}

	packaging, err := ctrl.PackagingService.CreatePackagingUnit(c, req)
	if err != nil {
		return err
	}

	return c.Status(fiber.StatusCreated).
		JSON(response.SuccessWithData[*model.PackagingUnit]{
			Code:    fiber.StatusCreated,
			Status:  "success",
			Message: "Packaging unit created successfully",
			Data:    packaging,
		})
}

func (ctrl *PackagingController) UpdatePackagingUnit(c *fiber.Ctx) error {
	id := c.Params("id")
	req := new(validation.UpdatePackagingUnit)
	if err := c.BodyParser(req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid request body")
	}

	packaging, err := ctrl.PackagingService.UpdatePackagingUnit(c, id, req)
	if err != nil {
		return err
	}

	return c.Status(fiber.StatusOK).
		JSON(response.SuccessWithData[*model.PackagingUnit]{
			Code:    fiber.StatusOK,
			Status:  "success",
			Message: "Packaging unit updated successfully",
			Data:    packaging,
		})
}

func (ctrl *PackagingController) DeletePackagingUnit(c *fiber.Ctx) error {
	id := c.Params("id")
	err := ctrl.PackagingService.DeletePackagingUnit(c, id)
	if err != nil {
		return err
	}

	return c.Status(fiber.StatusOK).
		JSON(response.Common{
			Code:    fiber.StatusOK,
			Status:  "success",
			Message: "Packaging unit deleted successfully",
		})
}
