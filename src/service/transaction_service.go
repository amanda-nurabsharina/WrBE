package service

import (
	"app/src/model"
	"app/src/utils"
	"app/src/validation"
	"fmt"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type TransactionService interface {
	GetTransactions(c *fiber.Ctx, search string, txType string) ([]model.StockTransaction, error)
	CreateInwardTransaction(c *fiber.Ctx, userID string, req *validation.InwardRequest) (*model.StockTransaction, error)
	CreateOutwardTransaction(c *fiber.Ctx, userID string, req *validation.OutwardRequest) ([]model.StockTransaction, error)
	CreateStockOpname(c *fiber.Ctx, userID string, req *validation.StockOpnameRequest) (*model.StockTransaction, error)
}

type transactionService struct {
	Log      *logrus.Logger
	DB       *gorm.DB
	Validate *validator.Validate
}

func NewTransactionService(db *gorm.DB, validate *validator.Validate) TransactionService {
	return &transactionService{
		Log:      utils.Log,
		DB:       db,
		Validate: validate,
	}
}

func (s *transactionService) GetTransactions(c *fiber.Ctx, search string, txType string) ([]model.StockTransaction, error) {
	var txs []model.StockTransaction
	query := s.DB.WithContext(c.Context()).
		Preload("User").
		Preload("Batch").
		Preload("Batch.Product").
		Preload("Batch.Warehouse").
		Preload("Batch.Location").
		Order("created_at desc")

	if txType != "" {
		query = query.Where("transaction_type = ?", txType)
	}

	if search != "" {
		query = query.Joins("Join inventory_batches On inventory_batches.id = stock_transactions.batch_id").
			Joins("Join products On products.id = inventory_batches.product_id").
			Where("stock_transactions.reference_no LIKE ? OR products.name LIKE ? OR inventory_batches.batch_number LIKE ?", "%"+search+"%", "%"+search+"%", "%"+search+"%")
	}

	if err := query.Find(&txs).Error; err != nil {
		s.Log.Errorf("Failed to query stock transactions: %v", err)
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Database error")
	}

	return txs, nil
}

func (s *transactionService) CreateInwardTransaction(c *fiber.Ctx, userID string, req *validation.InwardRequest) (*model.StockTransaction, error) {
	if err := s.Validate.Struct(req); err != nil {
		return nil, err
	}

	uID, err := uuid.Parse(userID)
	if err != nil {
		return nil, fiber.NewError(fiber.StatusBadRequest, "Invalid User ID format")
	}

	pID, err := uuid.Parse(req.ProductID)
	if err != nil {
		return nil, fiber.NewError(fiber.StatusBadRequest, "Invalid Product ID format")
	}

	wID, err := uuid.Parse(req.WarehouseID)
	if err != nil {
		return nil, fiber.NewError(fiber.StatusBadRequest, "Invalid Warehouse ID format")
	}

	locID, err := uuid.Parse(req.LocationID)
	if err != nil {
		return nil, fiber.NewError(fiber.StatusBadRequest, "Invalid Location ID format")
	}

	expDate, err := time.Parse("2006-01-02", req.ExpiredDate)
	if err != nil {
		return nil, fiber.NewError(fiber.StatusBadRequest, "Invalid Expiration Date format (must be YYYY-MM-DD)")
	}

	var tx *model.StockTransaction

	// Execute GORM Database Transaction
	errTx := s.DB.Transaction(func(txDb *gorm.DB) error {
		// 1. Check if Batch already exists in this location
		var batch model.InventoryBatch
		errFind := txDb.Where("product_id = ? And batch_number = ? And warehouse_id = ? And location_id = ?",
			pID, req.BatchNumber, wID, locID).First(&batch).Error

		if errFind == nil {
			// Batch exists: update Qty and update Expired Date if provided
			batch.Qty += req.Qty
			batch.ExpiredDate = expDate
			if time.Now().After(expDate) {
				batch.Status = "expired"
			} else {
				batch.Status = "active"
			}

			if errSave := txDb.Save(&batch).Error; errSave != nil {
				return errSave
			}
		} else if gorm.ErrRecordNotFound == errFind {
			// Batch doesn't exist: create new batch
			batchStatus := "active"
			if time.Now().After(expDate) {
				batchStatus = "expired"
			}

			batch = model.InventoryBatch{
				ProductID:     pID,
				BatchNumber:   req.BatchNumber,
				ExpiredDate:   expDate,
				Qty:           req.Qty,
				WarehouseID:   wID,
				LocationID:    locID,
				PurchasePrice: req.PurchasePrice,
				Status:        batchStatus,
			}

			if errCreate := txDb.Create(&batch).Error; errCreate != nil {
				return errCreate
			}
		} else {
			return errFind
		}

		// 2. Create Stock Transaction log
		tx = &model.StockTransaction{
			BatchID:         batch.ID,
			TransactionType: "IN",
			Qty:             req.Qty,
			ReferenceNo:     req.InvoiceNo,
			UserID:          uID,
		}

		if errCreateTx := txDb.Create(tx).Error; errCreateTx != nil {
			return errCreateTx
		}

		return nil
	})

	if errTx != nil {
		s.Log.Errorf("Transaction failed in Inward Stock: %v", errTx)
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Failed to record inward transaction")
	}

	return tx, nil
}

func (s *transactionService) CreateOutwardTransaction(c *fiber.Ctx, userID string, req *validation.OutwardRequest) ([]model.StockTransaction, error) {
	if err := s.Validate.Struct(req); err != nil {
		return nil, err
	}

	uID, err := uuid.Parse(userID)
	if err != nil {
		return nil, fiber.NewError(fiber.StatusBadRequest, "Invalid User ID format")
	}

	pID, err := uuid.Parse(req.ProductID)
	if err != nil {
		return nil, fiber.NewError(fiber.StatusBadRequest, "Invalid Product ID format")
	}

	var createdTxs []model.StockTransaction

	errTx := s.DB.Transaction(func(txDb *gorm.DB) error {
		// FEFO Selection Algorithm:
		// 1. Fetch active batches for this product with Qty > 0, ordered by expired_date ASC
		var activeBatches []model.InventoryBatch
		errQuery := txDb.Where("product_id = ? And qty > 0 And status = ?", pID, "active").
			Order("expired_date asc").Find(&activeBatches).Error
		if errQuery != nil {
			return errQuery
		}

		// Calculate total available stock in active batches
		totalAvailable := 0
		for _, b := range activeBatches {
			totalAvailable += b.Qty
		}

		if totalAvailable < req.Qty {
			return fmt.Errorf("insufficient stock: requested %d, but only %d units are available under active batches", req.Qty, totalAvailable)
		}

		remainingQty := req.Qty
		refNo := fmt.Sprintf("TX-OUT-%d", time.Now().Unix())

		for _, batch := range activeBatches {
			if remainingQty <= 0 {
				break
			}

			deductQty := 0
			if batch.Qty >= remainingQty {
				// Deduct everything from this batch
				deductQty = remainingQty
				batch.Qty -= remainingQty
				remainingQty = 0
			} else {
				// Empty this batch and move to next
				deductQty = batch.Qty
				remainingQty -= batch.Qty
				batch.Qty = 0
			}

			// Update batch in DB
			if errSave := txDb.Save(&batch).Error; errSave != nil {
				return errSave
			}

			// Create outward transaction log
			txLog := model.StockTransaction{
				BatchID:         batch.ID,
				TransactionType: "OUT",
				Qty:             deductQty,
				ReferenceNo:     refNo,
				UserID:          uID,
			}

			if errCreate := txDb.Create(&txLog).Error; errCreate != nil {
				return errCreate
			}

			createdTxs = append(createdTxs, txLog)
		}

		return nil
	})

	if errTx != nil {
		s.Log.Errorf("Transaction failed in FEFO Outward Stock: %v", errTx)
		return nil, fiber.NewError(fiber.StatusBadRequest, errTx.Error())
	}

	return createdTxs, nil
}

func (s *transactionService) CreateStockOpname(c *fiber.Ctx, userID string, req *validation.StockOpnameRequest) (*model.StockTransaction, error) {
	if err := s.Validate.Struct(req); err != nil {
		return nil, err
	}

	uID, err := uuid.Parse(userID)
	if err != nil {
		return nil, fiber.NewError(fiber.StatusBadRequest, "Invalid User ID format")
	}

	bID, err := uuid.Parse(req.BatchID)
	if err != nil {
		return nil, fiber.NewError(fiber.StatusBadRequest, "Invalid Batch ID format")
	}

	var tx *model.StockTransaction

	errTx := s.DB.Transaction(func(txDb *gorm.DB) error {
		var batch model.InventoryBatch
		if errFind := txDb.First(&batch, "id = ?", bID).Error; errFind != nil {
			return errFind
		}

		discrepancy := req.PhysicalQty - batch.Qty
		if discrepancy == 0 {
			return fmt.Errorf("no discrepancy found for batch %s (system qty matches physical qty)", batch.BatchNumber)
		}

		// Update batch quantity
		batch.Qty = req.PhysicalQty
		if errSave := txDb.Save(&batch).Error; errSave != nil {
			return errSave
		}

		// Log ADJUSTMENT transaction
		tx = &model.StockTransaction{
			BatchID:         batch.ID,
			TransactionType: "ADJUSTMENT",
			Qty:             discrepancy, // Positive indicates surplus, Negative indicates shortage
			ReferenceNo:     fmt.Sprintf("OP-%d", time.Now().Unix()),
			UserID:          uID,
		}

		if errCreate := txDb.Create(tx).Error; errCreate != nil {
			return errCreate
		}

		return nil
	})

	if errTx != nil {
		s.Log.Errorf("Transaction failed in Stock Opname: %v", errTx)
		return nil, fiber.NewError(fiber.StatusBadRequest, errTx.Error())
	}

	return tx, nil
}
