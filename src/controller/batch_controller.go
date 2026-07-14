package controller

import (
	"app/src/model"
	"app/src/response"
	"app/src/service"

	"github.com/gofiber/fiber/v2"
)

type BatchController struct {
	BatchService service.BatchService
}

func NewBatchController(batchService service.BatchService) *BatchController {
	return &BatchController{
		BatchService: batchService,
	}
}

func (ctrl *BatchController) GetBatches(c *fiber.Ctx) error {
	search := c.Query("search", "")
	productID := c.Query("product_id", "")
	status := c.Query("status", "")
	expiryDays := c.Query("expiry_days", "")

	batches, err := ctrl.BatchService.GetBatches(c, search, productID, status, expiryDays)
	if err != nil {
		return err
	}

	return c.Status(fiber.StatusOK).
		JSON(response.SuccessWithData[[]model.InventoryBatch]{
			Code:    fiber.StatusOK,
			Status:  "success",
			Message: "Batches retrieved successfully",
			Data:    batches,
		})
}

func (ctrl *BatchController) GetBatchByID(c *fiber.Ctx) error {
	id := c.Params("id")
	batch, err := ctrl.BatchService.GetBatchByID(c, id)
	if err != nil {
		return err
	}

	return c.Status(fiber.StatusOK).
		JSON(response.SuccessWithData[*model.InventoryBatch]{
			Code:    fiber.StatusOK,
			Status:  "success",
			Message: "Batch retrieved successfully",
			Data:    batch,
		})
}

func (ctrl *BatchController) UpdateBatchStatus(c *fiber.Ctx) error {
	id := c.Params("id")
	type UpdateStatusReq struct {
		Status string `json:"status" validate:"required"`
	}
	req := new(UpdateStatusReq)
	if err := c.BodyParser(req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid request body")
	}

	batch, err := ctrl.BatchService.UpdateBatchStatus(c, id, req.Status)
	if err != nil {
		return err
	}

	return c.Status(fiber.StatusOK).
		JSON(response.SuccessWithData[*model.InventoryBatch]{
			Code:    fiber.StatusOK,
			Status:  "success",
			Message: "Batch status updated successfully",
			Data:    batch,
		})
}

func (ctrl *BatchController) GetExpiryAlerts(c *fiber.Ctx) error {
	batches, err := ctrl.BatchService.GetExpiryAlerts(c)
	if err != nil {
		return err
	}

	return c.Status(fiber.StatusOK).
		JSON(response.SuccessWithData[[]model.InventoryBatch]{
			Code:    fiber.StatusOK,
			Status:  "success",
			Message: "Expiry alerts retrieved successfully",
			Data:    batches,
		})
}
