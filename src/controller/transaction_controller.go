package controller

import (
	"strings"

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

	if txType == "" {
		path := c.Path()
		if strings.HasSuffix(path, "/in") {
			txType = "in"
		} else if strings.HasSuffix(path, "/out") {
			txType = "out"
		} else if strings.HasSuffix(path, "/adjustment") {
			txType = "adjustment"
		}
	}

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

func (ctrl *TransactionController) ApproveB3Inward(c *fiber.Ctx) error {
	id := c.Params("id")
	batch, err := ctrl.TxService.ApproveB3Inward(c, id)
	if err != nil {
		return err
	}

	return c.Status(fiber.StatusOK).
		JSON(response.SuccessWithData[*model.InventoryBatch]{
			Code:    fiber.StatusOK,
			Status:  "success",
			Message: "B3 quarantined batch approved successfully",
			Data:    batch,
		})
}

func (ctrl *TransactionController) UpdateTransaction(c *fiber.Ctx) error {
	id := c.Params("id")
	req := new(validation.UpdateTransactionRequest)
	if err := c.BodyParser(req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid request body")
	}

	tx, err := ctrl.TxService.UpdateTransaction(c, id, req)
	if err != nil {
		return err
	}

	return c.Status(fiber.StatusOK).
		JSON(response.SuccessWithData[*model.StockTransaction]{
			Code:    fiber.StatusOK,
			Status:  "success",
			Message: "Transaction updated successfully",
			Data:    tx,
		})
}

func (ctrl *TransactionController) CompleteTransaction(c *fiber.Ctx) error {
	id := c.Params("id")
	type CompleteReq struct {
		ProofDocument string `json:"proof_document"`
	}
	req := new(CompleteReq)
	c.BodyParser(req)

	tx, err := ctrl.TxService.CompleteTransaction(c, id, req.ProofDocument)
	if err != nil {
		return err
	}

	return c.Status(fiber.StatusOK).
		JSON(response.SuccessWithData[*model.StockTransaction]{
			Code:    fiber.StatusOK,
			Status:  "success",
			Message: "Transaction finalized successfully",
			Data:    tx,
		})
}

type ConfirmPickRequest struct {
	BatchBarcode    string `json:"batch_barcode" validate:"required"`
	LocationBarcode string `json:"location_barcode" validate:"required"`
}

func (ctrl *TransactionController) ConfirmPick(c *fiber.Ctx) error {
	id := c.Params("id")
	user, ok := c.Locals("user").(*model.User)
	if !ok || user == nil {
		return fiber.NewError(fiber.StatusUnauthorized, "Unauthorized")
	}

	req := new(ConfirmPickRequest)
	if err := c.BodyParser(req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid request body")
	}

	tx, err := ctrl.TxService.ConfirmPick(c, user.ID.String(), id, req.BatchBarcode, req.LocationBarcode)
	if err != nil {
		return err
	}

	return c.Status(fiber.StatusOK).
		JSON(response.SuccessWithData[*model.StockTransaction]{
			Code:    fiber.StatusOK,
			Status:  "success",
			Message: "Pick confirmed and item dispatched successfully",
			Data:    tx,
		})
}
