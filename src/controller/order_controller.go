package controller

import (
	"app/src/model"
	"app/src/response"
	"app/src/service"
	"app/src/validation"

	"github.com/gofiber/fiber/v2"
)

type OrderController struct {
	OrderService service.OrderService
}

func NewOrderController(orderService service.OrderService) *OrderController {
	return &OrderController{
		OrderService: orderService,
	}
}

// ----------------------------------------------------------------------------
// Purchase Orders
// ----------------------------------------------------------------------------

func (ctrl *OrderController) GetPurchaseOrders(c *fiber.Ctx) error {
	search := c.Query("search", "")
	list, err := ctrl.OrderService.GetPurchaseOrders(c, search)
	if err != nil {
		return err
	}

	return c.Status(fiber.StatusOK).
		JSON(response.SuccessWithData[[]model.PurchaseOrder]{
			Code:    fiber.StatusOK,
			Status:  "success",
			Message: "Purchase orders retrieved successfully",
			Data:    list,
		})
}

func (ctrl *OrderController) CreatePurchaseOrder(c *fiber.Ctx) error {
	req := new(validation.CreatePurchaseOrder)
	if err := c.BodyParser(req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid request body")
	}

	po, err := ctrl.OrderService.CreatePurchaseOrder(c, req)
	if err != nil {
		return err
	}

	return c.Status(fiber.StatusCreated).
		JSON(response.SuccessWithData[*model.PurchaseOrder]{
			Code:    fiber.StatusCreated,
			Status:  "success",
			Message: "Purchase order created successfully",
			Data:    po,
		})
}

func (ctrl *OrderController) ApprovePurchaseOrder(c *fiber.Ctx) error {
	id := c.Params("id")
	po, err := ctrl.OrderService.ApprovePurchaseOrder(c, id)
	if err != nil {
		return err
	}

	return c.Status(fiber.StatusOK).
		JSON(response.SuccessWithData[*model.PurchaseOrder]{
			Code:    fiber.StatusOK,
			Status:  "success",
			Message: "Purchase order approved successfully",
			Data:    po,
		})
}

// ----------------------------------------------------------------------------
// Sales Orders
// ----------------------------------------------------------------------------

func (ctrl *OrderController) GetSalesOrders(c *fiber.Ctx) error {
	search := c.Query("search", "")
	list, err := ctrl.OrderService.GetSalesOrders(c, search)
	if err != nil {
		return err
	}

	return c.Status(fiber.StatusOK).
		JSON(response.SuccessWithData[[]model.SalesOrder]{
			Code:    fiber.StatusOK,
			Status:  "success",
			Message: "Sales orders retrieved successfully",
			Data:    list,
		})
}

func (ctrl *OrderController) CreateSalesOrder(c *fiber.Ctx) error {
	req := new(validation.CreateSalesOrder)
	if err := c.BodyParser(req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid request body")
	}

	so, err := ctrl.OrderService.CreateSalesOrder(c, req)
	if err != nil {
		return err
	}

	return c.Status(fiber.StatusCreated).
		JSON(response.SuccessWithData[*model.SalesOrder]{
			Code:    fiber.StatusCreated,
			Status:  "success",
			Message: "Sales order created successfully",
			Data:    so,
		})
}

func (ctrl *OrderController) ApproveSalesOrder(c *fiber.Ctx) error {
	id := c.Params("id")
	so, err := ctrl.OrderService.ApproveSalesOrder(c, id)
	if err != nil {
		return err
	}

	return c.Status(fiber.StatusOK).
		JSON(response.SuccessWithData[*model.SalesOrder]{
			Code:    fiber.StatusOK,
			Status:  "success",
			Message: "Sales order approved successfully",
			Data:    so,
		})
}

func (ctrl *OrderController) UpdateSalesOrderPaymentStatus(c *fiber.Ctx) error {
	id := c.Params("id")
	req := new(validation.UpdateSalesOrderPaymentRequest)
	if err := c.BodyParser(req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid request body")
	}

	so, err := ctrl.OrderService.UpdateSalesOrderPaymentStatus(c, id, req)
	if err != nil {
		return err
	}

	return c.Status(fiber.StatusOK).
		JSON(response.SuccessWithData[*model.SalesOrder]{
			Code:    fiber.StatusOK,
			Status:  "success",
			Message: "Sales order payment status updated successfully",
			Data:    so,
		})
}
