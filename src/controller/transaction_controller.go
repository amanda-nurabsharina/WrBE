package controller

import (
	"app/src/model"
	"app/src/response"
	"app/src/service"
	"app/src/validation"

	"github.com/gofiber/fiber/v2"
)

type TransactionController struct {
	TxService service.TransactionService
}

func NewTransactionController(txService service.TransactionService) *TransactionController {
	return &TransactionController{
		TxService: txService,
	}
}

func (ctrl *TransactionController) GetTransactions(c *fiber.Ctx) error {
	search := c.Query("search", "")
	txType := c.Query("type", "")

	txs, err := ctrl.TxService.GetTransactions(c, search, txType)
	if err != nil {
		return err
	}

	return c.Status(fiber.StatusOK).
		JSON(response.SuccessWithData[[]model.StockTransaction]{
			Code:    fiber.StatusOK,
			Status:  "success",
			Message: "Transactions retrieved successfully",
			Data:    txs,
		})
}

func (ctrl *TransactionController) CreateInwardTransaction(c *fiber.Ctx) error {
	user, ok := c.Locals("user").(*model.User)
	if !ok || user == nil {
		return fiber.NewError(fiber.StatusUnauthorized, "Unauthorized")
	}

	req := new(validation.InwardRequest)
	if err := c.BodyParser(req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid request body")
	}

	tx, err := ctrl.TxService.CreateInwardTransaction(c, user.ID.String(), req)
	if err != nil {
		return err
	}

	return c.Status(fiber.StatusCreated).
		JSON(response.SuccessWithData[*model.StockTransaction]{
			Code:    fiber.StatusCreated,
			Status:  "success",
			Message: "Inward transaction recorded successfully",
			Data:    tx,
		})
}

func (ctrl *TransactionController) CreateOutwardTransaction(c *fiber.Ctx) error {
	user, ok := c.Locals("user").(*model.User)
	if !ok || user == nil {
		return fiber.NewError(fiber.StatusUnauthorized, "Unauthorized")
	}

	req := new(validation.OutwardRequest)
	if err := c.BodyParser(req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid request body")
	}

	txs, err := ctrl.TxService.CreateOutwardTransaction(c, user.ID.String(), req)
	if err != nil {
		return err
	}

	return c.Status(fiber.StatusCreated).
		JSON(response.SuccessWithData[[]model.StockTransaction]{
			Code:    fiber.StatusCreated,
			Status:  "success",
			Message: "Outward FEFO transaction recorded successfully",
			Data:    txs,
		})
}

func (ctrl *TransactionController) CreateStockOpname(c *fiber.Ctx) error {
	user, ok := c.Locals("user").(*model.User)
	if !ok || user == nil {
		return fiber.NewError(fiber.StatusUnauthorized, "Unauthorized")
	}

	req := new(validation.StockOpnameRequest)
	if err := c.BodyParser(req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid request body")
	}

	tx, err := ctrl.TxService.CreateStockOpname(c, user.ID.String(), req)
	if err != nil {
		return err
	}

	return c.Status(fiber.StatusOK).
		JSON(response.SuccessWithData[*model.StockTransaction]{
			Code:    fiber.StatusOK,
			Status:  "success",
			Message: "Stock Opname discrepancy adjusted successfully",
			Data:    tx,
		})
}
