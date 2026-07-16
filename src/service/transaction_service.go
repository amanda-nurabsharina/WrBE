package service

import (
	"app/src/model"
	"app/src/utils"
	"app/src/validation"
	"errors"
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
	ApproveB3Inward(c *fiber.Ctx, batchID string) (*model.InventoryBatch, error)
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

	var poUUID *uuid.UUID
	if req.POID != "" {
		p, err := uuid.Parse(req.POID)
		if err == nil {
			poUUID = &p
		}
	}

	var tx *model.StockTransaction

	// Execute GORM Database Transaction
	errTx := s.DB.Transaction(func(txDb *gorm.DB) error {
		// If PO is provided, matching delivery checks
		if poUUID != nil {
			var po model.PurchaseOrder
			if err := txDb.Preload("Items").First(&po, "id = ?", *poUUID).Error; err != nil {
				return fmt.Errorf("purchase order not found")
			}
			if po.Status != "approved" && po.Status != "partially_received" {
				return fmt.Errorf("selected purchase order is not approved or is already completed")
			}

			// Find the item for this product
			foundItem := false
			for i, item := range po.Items {
				if item.ProductID == pID {
					po.Items[i].ReceivedQty += req.Qty
					if po.Items[i].ReceivedQty > item.Qty {
						return fmt.Errorf("received quantity exceeds ordered quantity for product in this PO")
					}
					if errSaveItem := txDb.Save(&po.Items[i]).Error; errSaveItem != nil {
						return errSaveItem
					}
					foundItem = true
					break
				}
			}
			if !foundItem {
				return fmt.Errorf("product not found in the selected purchase order")
			}

			// Update PO status
			allCompleted := true
			for _, item := range po.Items {
				if item.ReceivedQty < item.Qty {
					allCompleted = false
				}
			}
			if allCompleted {
				po.Status = "completed"
			} else {
				po.Status = "partially_received"
			}
			if errSavePO := txDb.Save(&po).Error; errSavePO != nil {
				return errSavePO
			}
		}

		// Retrieve product to check B3 regulation category
		var product model.Product
		if errProd := txDb.First(&product, "id = ?", pID).Error; errProd != nil {
			return fmt.Errorf("product not found")
		}

		batchStatus := "active"
		if product.RegCategory == "B3" {
			batchStatus = "quarantine"
		} else if time.Now().After(expDate) {
			batchStatus = "expired"
		}

		// Check if Batch already exists in this location
		var batch model.InventoryBatch
		errFind := txDb.Where("product_id = ? And batch_number = ? And warehouse_id = ? And location_id = ?",
			pID, req.BatchNumber, wID, locID).First(&batch).Error

		if errFind == nil {
			// Batch exists: update Qty and update Expired Date if provided
			batch.Qty += req.Qty
			batch.ExpiredDate = expDate
			if batchStatus != "quarantine" {
				if time.Now().After(expDate) {
					batch.Status = "expired"
				} else {
					batch.Status = "active"
				}
			}

			if errSave := txDb.Save(&batch).Error; errSave != nil {
				return errSave
			}
		} else if gorm.ErrRecordNotFound == errFind {
			// Batch doesn't exist: create new batch
			batch = model.InventoryBatch{
				ProductID:     pID,
				BatchNumber:   req.BatchNumber,
				ExpiredDate:   expDate,
				Qty:           req.Qty,
				WarehouseID:   wID,
				LocationID:    locID,
				PurchasePrice: req.PurchasePrice,
				Status:        batchStatus,
				POID:          poUUID,
			}

			if errCreate := txDb.Create(&batch).Error; errCreate != nil {
				return errCreate
			}
		} else {
			return errFind
		}

		// Create Stock Transaction log
		tx = &model.StockTransaction{
			BatchID:         batch.ID,
			TransactionType: "IN",
			Qty:             req.Qty,
			ReferenceNo:     req.InvoiceNo,
			UserID:          uID,
			POID:            poUUID,
		}

		if errCreateTx := txDb.Create(tx).Error; errCreateTx != nil {
			return errCreateTx
		}

		return nil
	})

	if errTx != nil {
		s.Log.Errorf("Transaction failed in Inward Stock: %v", errTx)
		return nil, fiber.NewError(fiber.StatusBadRequest, errTx.Error())
	}

	// Retrieve batch details for description
	var product model.Product
	s.DB.First(&product, "id = ?", tx.Batch.ProductID)
	LogCtxActivity(s.DB, c, "CREATE", "inward", tx.ID.String(), fmt.Sprintf("Received inward stock: %d units of product code %s in batch %s", tx.Qty, product.Code, tx.Batch.BatchNumber))

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

	var soUUID *uuid.UUID
	if req.SOID != "" {
		s, err := uuid.Parse(req.SOID)
		if err == nil {
			soUUID = &s
		}
	}

	var createdTxs []model.StockTransaction
	refNo := fmt.Sprintf("TX-OUT-%d", time.Now().Unix())

	errTx := s.DB.Transaction(func(txDb *gorm.DB) error {
		// If SO is provided, validate B3 approvals and update SO progress
		if soUUID != nil {
			var so model.SalesOrder
			if err := txDb.Preload("Items").First(&so, "id = ?", *soUUID).Error; err != nil {
				return fmt.Errorf("sales order not found")
			}

			if so.Status == "pending_b3_approval" {
				return fmt.Errorf("selected Sales Order contains B3 products and requires admin approval before fulfillment")
			}

			if so.Status != "approved" && so.Status != "partially_shipped" {
				return fmt.Errorf("selected Sales Order is not approved or is already fully shipped")
			}

			// Find item for this product
			foundItem := false
			for i, item := range so.Items {
				if item.ProductID == pID {
					so.Items[i].ShippedQty += req.Qty
					if so.Items[i].ShippedQty > item.Qty {
						return fmt.Errorf("shipped quantity exceeds ordered quantity in this SO")
					}
					if errSaveItem := txDb.Save(&so.Items[i]).Error; errSaveItem != nil {
						return errSaveItem
					}
					foundItem = true
					break
				}
			}
			if !foundItem {
				return fmt.Errorf("product not found in the selected sales order")
			}

			// Update SO status
			allCompleted := true
			for _, item := range so.Items {
				if item.ShippedQty < item.Qty {
					allCompleted = false
				}
			}
			if allCompleted {
				so.Status = "shipped"
			} else {
				so.Status = "partially_shipped"
			}
			if errSaveSO := txDb.Save(&so).Error; errSaveSO != nil {
				return errSaveSO
			}
		}

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
			return fmt.Errorf("insufficient active stock: requested %d, but only %d units are available in active batches", req.Qty, totalAvailable)
		}

		remainingQty := req.Qty

		for _, batch := range activeBatches {
			if remainingQty <= 0 {
				break
			}

			deductQty := 0
			if batch.Qty >= remainingQty {
				deductQty = remainingQty
				batch.Qty -= remainingQty
				remainingQty = 0
			} else {
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
				SOID:            soUUID,
				SellingPrice:    req.SellingPrice,
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

	// Retrieve product name
	var product model.Product
	s.DB.First(&product, "id = ?", pID)
	LogCtxActivity(s.DB, c, "CREATE", "outward", refNo, fmt.Sprintf("Shipped outward stock: %d units of product code %s using FEFO", req.Qty, product.Code))

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

		if discrepancy != 0 {
			// Update batch quantity when there is a discrepancy
			batch.Qty = req.PhysicalQty
			if errSave := txDb.Save(&batch).Error; errSave != nil {
				return errSave
			}
		}

		// Determine transaction type: OPNAME_MATCH or ADJUSTMENT
		txType := "ADJUSTMENT"
		if discrepancy == 0 {
			txType = "OPNAME_MATCH"
		}

		desc := req.Description
		if discrepancy == 0 && desc == "" {
			desc = "Stock verified - no discrepancy"
		}

		// Log opname transaction
		tx = &model.StockTransaction{
			BatchID:         batch.ID,
			TransactionType: txType,
			Qty:             discrepancy, // 0 for match, +/- for adjustment
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

	// Retrieve product name
	var product model.Product
	var batch model.InventoryBatch
	s.DB.First(&batch, "id = ?", tx.BatchID)
	s.DB.First(&product, "id = ?", batch.ProductID)
	msg := fmt.Sprintf("Conducted Stock Opname verification for batch %s of product code %s. Physical qty: %d, discrepancy: %d", batch.BatchNumber, product.Code, req.PhysicalQty, tx.Qty)
	LogCtxActivity(s.DB, c, "OPNAME", "opname", tx.ID.String(), msg)

	return tx, nil
}

func (s *transactionService) ApproveB3Inward(c *fiber.Ctx, batchID string) (*model.InventoryBatch, error) {
	userObj := c.Locals("user")
	user, ok := userObj.(*model.User)
	if !ok || user == nil {
		return nil, fiber.NewError(fiber.StatusUnauthorized, "User authentication failed")
	}

	isAllowed := user.Role == "super_admin" || user.Role == "super admin" || user.Role == "approver"
	if !isAllowed {
		var role model.Role
		if err := s.DB.WithContext(c.Context()).First(&role, "name = ? AND deleted_at IS NULL", user.Role).Error; err == nil {
			for _, menu := range role.AccessibleMenus {
				if menu == "approver" {
					isAllowed = true
					break
				}
			}
		}
	}

	if !isAllowed {
		return nil, fiber.NewError(fiber.StatusForbidden, "Only super admin or users with approval roles can approve")
	}

	bID, err := uuid.Parse(batchID)
	if err != nil {
		return nil, fiber.NewError(fiber.StatusBadRequest, "Invalid Batch ID format")
	}

	var batch model.InventoryBatch
	if err := s.DB.WithContext(c.Context()).First(&batch, "id = ?", bID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fiber.NewError(fiber.StatusNotFound, "Inventory batch not found")
		}
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Database error")
	}

	if batch.Status != "quarantine" {
		return nil, fiber.NewError(fiber.StatusBadRequest, "Only quarantined batches can be approved")
	}

	batch.Status = "active"
	if err := s.DB.WithContext(c.Context()).Save(&batch).Error; err != nil {
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Failed to approve quarantined batch")
	}

	// Fetch product code
	var product model.Product
	s.DB.First(&product, "id = ?", batch.ProductID)
	LogCtxActivity(s.DB, c, "APPROVE", "inward", batch.ID.String(), fmt.Sprintf("Approved quarantined B3 product batch %s of product code %s", batch.BatchNumber, product.Code))

	return &batch, nil
}
