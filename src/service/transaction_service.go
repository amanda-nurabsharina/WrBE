package service

import (
	"app/src/barcode"
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
	UpdateTransaction(c *fiber.Ctx, id string, req *validation.UpdateTransactionRequest) (*model.StockTransaction, error)
	CompleteTransaction(c *fiber.Ctx, id string, proofDocument string) (*model.StockTransaction, error)
	ConfirmPick(c *fiber.Ctx, userID string, txID string, batchBarcode string, locationBarcode string) (*model.StockTransaction, error)
}

type transactionService struct {
	Log       *logrus.Logger
	DB        *gorm.DB
	Validate  *validator.Validate
	bcService barcode.Service
}

func NewTransactionService(db *gorm.DB, validate *validator.Validate, bcService barcode.Service) TransactionService {
	return &transactionService{
		Log:       utils.Log,
		DB:        db,
		Validate:  validate,
		bcService: bcService,
	}
}

func (s *transactionService) GetTransactions(c *fiber.Ctx, search string, txType string) ([]model.StockTransaction, error) {
	var txs []model.StockTransaction
	query := s.DB.WithContext(c.Context()).
		Preload("User").
		Preload("Supplier").
		Preload("Batch").
		Preload("Batch.Product").
		Preload("Batch.Warehouse").
		Preload("Batch.Location").
		Order("created_at desc")

	if txType != "" {
		query = query.Where("UPPER(transaction_type) = UPPER(?)", txType)
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

	var supUUID *uuid.UUID
	if req.SupplierID != "" {
		s, err := uuid.Parse(req.SupplierID)
		if err == nil {
			supUUID = &s
		}
	}

	var tx *model.StockTransaction

	// Execute GORM Database Transaction
	errTx := s.DB.Transaction(func(txDb *gorm.DB) error {
		// If PO is provided, matching delivery checks (only if not draft!)
		txStatus := "completed"
		if req.Status == "draft" {
			txStatus = "draft"
		}

		if poUUID != nil && txStatus != "draft" {
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

		// Retrieve virtual temporary warehouse and location for receiving
		var tempWH model.Warehouse
		if err := txDb.Where("code = ?", "TEMP-WH").First(&tempWH).Error; err != nil {
			return fmt.Errorf("temporary warehouse not found")
		}

		var tempLoc model.Location
		if err := txDb.Where("warehouse_id = ? AND rack = ?", tempWH.ID, "TEMP-RECEIVING").First(&tempLoc).Error; err != nil {
			return fmt.Errorf("temporary receiving location not found")
		}

		// Reassign target location and warehouse to the virtual temp area
		wID = tempWH.ID
		locID = tempLoc.ID

		// Check if Batch already exists in the temp receiving location (only if not draft)
		var batch model.InventoryBatch
		var errFind error
		if txStatus == "draft" {
			errFind = gorm.ErrRecordNotFound
		} else {
			errFind = txDb.Where("product_id = ? And batch_number = ? And warehouse_id = ? And location_id = ? And status != ?",
				pID, req.BatchNumber, wID, locID, "draft").First(&batch).Error
		}

		if errFind == nil {
			// Batch exists: update Qty and update Expired Date if provided
			batch.Qty += req.Qty
			batch.AvailableQty = batch.Qty - batch.AllocatedQty
			batch.ExpiredDate = expDate

			if errSave := txDb.Save(&batch).Error; errSave != nil {
				return errSave
			}

			// Create Inventory Movement log for receiving
			parsedUserID, _ := uuid.Parse(userID)
			movement := model.InventoryMovement{
				BatchID:      batch.ID,
				ToLocationID: &tempLoc.ID,
				Qty:          req.Qty,
				MovementType: "RECEIVING",
				CreatedBy:    parsedUserID,
			}
			if errMove := txDb.Create(&movement).Error; errMove != nil {
				return errMove
			}
		} else if errors.Is(errFind, gorm.ErrRecordNotFound) {
			// Batch doesn't exist: create new batch in waiting_put_away status
			batch = model.InventoryBatch{
				ProductID:     pID,
				BatchNumber:   req.BatchNumber,
				ExpiredDate:   expDate,
				Qty:           req.Qty,
				AllocatedQty:  0,
				AvailableQty:  req.Qty,
				WarehouseID:   wID,
				LocationID:    locID,
				PurchasePrice: req.PurchasePrice,
				Status:        "waiting_put_away",
				POID:          poUUID,
				ReceivedAt:    time.Now(),
			}

			if errCreate := txDb.Create(&batch).Error; errCreate != nil {
				return errCreate
			}

			// Generate sequential Batch Barcode and register it internally
			barcodeStr, errBar := s.bcService.GenerateBarcode(txDb, "BAT", batch.ID)
			if errBar != nil {
				return errBar
			}
			batch.Barcode = barcodeStr
			if errSaveBar := txDb.Save(&batch).Error; errSaveBar != nil {
				return errSaveBar
			}

			// Create Inventory Movement log for receiving
			parsedUserID, _ := uuid.Parse(userID)
			movement := model.InventoryMovement{
				BatchID:      batch.ID,
				ToLocationID: &tempLoc.ID,
				Qty:          req.Qty,
				MovementType: "RECEIVING",
				CreatedBy:    parsedUserID,
			}
			if errMove := txDb.Create(&movement).Error; errMove != nil {
				return errMove
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
			SupplierID:      supUUID,
			ProofDocument:   req.ProofDocument,
			Status:          txStatus,
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
	LogCtxActivity(s.DB, c, "CREATE", "inward", tx.ID.String(), fmt.Sprintf("Received inward stock: %d units of product code %s in batch %s (status: %s)", tx.Qty, product.Code, tx.Batch.BatchNumber, tx.Status))

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
	refNo := req.InvoiceNo
	if refNo == "" {
		refNo = fmt.Sprintf("TX-OUT-%d", time.Now().Unix())
	}

	errTx := s.DB.Transaction(func(txDb *gorm.DB) error {
		txStatus := "completed"
		if req.Status == "draft" {
			txStatus = "draft"
		}

		// If SO is provided, validate B3 approvals and update SO progress
		if soUUID != nil && txStatus != "draft" {
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
		// 1. Fetch stored batches for this product with AvailableQty > 0, ordered by FEFO rules
		var activeBatches []model.InventoryBatch
		errQuery := txDb.Where("product_id = ? And available_qty > 0 And status IN ('stored', 'active', 'waiting_put_away')", pID).
			Order("expired_date asc, received_at asc, id asc").Find(&activeBatches).Error
		if errQuery != nil {
			return errQuery
		}

		// Calculate total available stock in active batches
		totalAvailable := 0
		for _, b := range activeBatches {
			totalAvailable += b.AvailableQty
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
			if batch.AvailableQty >= remainingQty {
				deductQty = remainingQty
				batch.AvailableQty -= remainingQty
				batch.AllocatedQty += remainingQty
				remainingQty = 0
			} else {
				deductQty = batch.AvailableQty
				remainingQty -= batch.AvailableQty
				batch.AllocatedQty += batch.AvailableQty
				batch.AvailableQty = 0
			}

			// Update batch in DB
			if errSave := txDb.Save(&batch).Error; errSave != nil {
				return errSave
			}

			// Create outward transaction log (initialized in 'picking' status)
			txLog := model.StockTransaction{
				BatchID:         batch.ID,
				TransactionType: "OUT",
				Qty:             deductQty,
				ReferenceNo:     refNo,
				UserID:          uID,
				SOID:            soUUID,
				SellingPrice:    req.SellingPrice,
				Destination:     req.Destination,
				Description:     req.Description,
				ProofDocument:   req.ProofDocument,
				Status:          "picking",
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
	LogCtxActivity(s.DB, c, "CREATE", "outward", refNo, fmt.Sprintf("Shipped outward stock: %d units of product code %s using FEFO (status: %s)", req.Qty, product.Code, req.Status))

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

func (s *transactionService) UpdateTransaction(c *fiber.Ctx, id string, req *validation.UpdateTransactionRequest) (*model.StockTransaction, error) {
	if err := s.Validate.Struct(req); err != nil {
		return nil, err
	}

	txID, err := uuid.Parse(id)
	if err != nil {
		return nil, fiber.NewError(fiber.StatusBadRequest, "Invalid Transaction ID format")
	}

	var tx model.StockTransaction
	if err := s.DB.Preload("Batch").First(&tx, "id = ?", txID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fiber.NewError(fiber.StatusNotFound, "Transaction not found")
		}
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Database error")
	}

	errTx := s.DB.Transaction(func(txDb *gorm.DB) error {
		var batch model.InventoryBatch
		if err := txDb.First(&batch, "id = ?", tx.BatchID).Error; err != nil {
			return err
		}

		if tx.TransactionType == "IN" {
			// Inward edit: Qty adjustment check
			adjustedQty := batch.Qty - tx.Qty + req.Qty
			if adjustedQty < 0 {
				return fmt.Errorf("cannot reduce quantity to %d because batch only has %d units left (stock already consumed)", req.Qty, batch.Qty)
			}

			// Update batch quantities and settings
			batch.Qty = adjustedQty
			if req.BatchNumber != "" {
				batch.BatchNumber = req.BatchNumber
			}
			if req.ExpiredDate != "" {
				expDate, err := time.Parse("2006-01-02", req.ExpiredDate)
				if err != nil {
					return fmt.Errorf("invalid expired date format")
				}
				batch.ExpiredDate = expDate
			}
			if req.WarehouseID != "" {
				wID, err := uuid.Parse(req.WarehouseID)
				if err != nil {
					return fmt.Errorf("invalid warehouse id")
				}
				batch.WarehouseID = wID
			}
			if req.LocationID != "" {
				locID, err := uuid.Parse(req.LocationID)
				if err != nil {
					return fmt.Errorf("invalid location id")
				}
				batch.LocationID = locID
			}
			if req.Price > 0 {
				batch.PurchasePrice = req.Price
			}

			if errSave := txDb.Save(&batch).Error; errSave != nil {
				return errSave
			}

			// Update transaction
			tx.Qty = req.Qty
			tx.ReferenceNo = req.ReferenceNo
			tx.ProofDocument = req.ProofDocument
			if req.SupplierID != "" {
				supID, err := uuid.Parse(req.SupplierID)
				if err == nil {
					tx.SupplierID = &supID
				}
			}

			if errSaveTx := txDb.Save(&tx).Error; errSaveTx != nil {
				return errSaveTx
			}

		} else if tx.TransactionType == "OUT" {
			// Outward edit: Qty adjustment check
			adjustedQty := batch.Qty + tx.Qty - req.Qty
			if adjustedQty < 0 {
				return fmt.Errorf("insufficient stock in batch %s: requested adjustment requires %d units, but only %d units are available in this batch", batch.BatchNumber, req.Qty, batch.Qty)
			}

			batch.Qty = adjustedQty
			if errSave := txDb.Save(&batch).Error; errSave != nil {
				return errSave
			}

			// Update transaction
			tx.Qty = req.Qty
			tx.SellingPrice = req.Price
			tx.ReferenceNo = req.ReferenceNo
			tx.Destination = req.Destination
			tx.Description = req.Description
			tx.ProofDocument = req.ProofDocument

			if errSaveTx := txDb.Save(&tx).Error; errSaveTx != nil {
				return errSaveTx
			}
		}

		return nil
	})

	if errTx != nil {
		return nil, fiber.NewError(fiber.StatusBadRequest, errTx.Error())
	}

	// Reload with preloads
	s.DB.Preload("User").Preload("Supplier").Preload("Batch").Preload("Batch.Product").Preload("Batch.Warehouse").Preload("Batch.Location").First(&tx, tx.ID)

	LogCtxActivity(s.DB, c, "UPDATE", "transaction", tx.ID.String(), fmt.Sprintf("Updated transaction %s (type: %s, new qty: %d)", tx.ReferenceNo, tx.TransactionType, tx.Qty))

	return &tx, nil
}

func (s *transactionService) CompleteTransaction(c *fiber.Ctx, id string, proofDocument string) (*model.StockTransaction, error) {
	txID, err := uuid.Parse(id)
	if err != nil {
		return nil, fiber.NewError(fiber.StatusBadRequest, "Invalid Transaction ID format")
	}

	var tx model.StockTransaction
	if err := s.DB.Preload("Batch").First(&tx, "id = ?", txID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fiber.NewError(fiber.StatusNotFound, "Transaction not found")
		}
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Database error")
	}

	if tx.Status != "draft" {
		return nil, fiber.NewError(fiber.StatusBadRequest, "Only draft transactions can be completed")
	}

	errTx := s.DB.Transaction(func(txDb *gorm.DB) error {
		var batch model.InventoryBatch
		if err := txDb.First(&batch, "id = ?", tx.BatchID).Error; err != nil {
			return err
		}

		// Activate batch/transaction
		tx.Status = "completed"
		if proofDocument != "" {
			tx.ProofDocument = proofDocument
		}

		if tx.TransactionType == "IN" {
			// Determine final batch status
			var product model.Product
			if errProd := txDb.First(&product, "id = ?", batch.ProductID).Error; errProd != nil {
				return errProd
			}

			batchStatus := "active"
			if product.RegCategory == "B3" {
				batchStatus = "quarantine"
			} else if time.Now().After(batch.ExpiredDate) {
				batchStatus = "expired"
			}
			batch.Status = batchStatus

			// If linked to PO, perform check & PO progress update now!
			if tx.POID != nil {
				var po model.PurchaseOrder
				if err := txDb.Preload("Items").First(&po, "id = ?", *tx.POID).Error; err == nil {
					for i, item := range po.Items {
						if item.ProductID == batch.ProductID {
							po.Items[i].ReceivedQty += tx.Qty
							txDb.Save(&po.Items[i])
							break
						}
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
					txDb.Save(&po)
				}
			}
		} else if tx.TransactionType == "OUT" {
			// If linked to SO, perform SO progress update now!
			if tx.SOID != nil {
				var so model.SalesOrder
				if err := txDb.Preload("Items").First(&so, "id = ?", *tx.SOID).Error; err == nil {
					for i, item := range so.Items {
						if item.ProductID == batch.ProductID {
							so.Items[i].ShippedQty += tx.Qty
							txDb.Save(&so.Items[i])
							break
						}
					}
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
					txDb.Save(&so)
				}
			}
		}

		if errSaveBatch := txDb.Save(&batch).Error; errSaveBatch != nil {
			return errSaveBatch
		}

		if errSaveTx := txDb.Save(&tx).Error; errSaveTx != nil {
			return errSaveTx
		}

		return nil
	})

	if errTx != nil {
		return nil, fiber.NewError(fiber.StatusBadRequest, errTx.Error())
	}

	// Reload with preloads
	s.DB.Preload("User").Preload("Supplier").Preload("Batch").Preload("Batch.Product").Preload("Batch.Warehouse").Preload("Batch.Location").First(&tx, tx.ID)

	LogCtxActivity(s.DB, c, "APPROVE", "transaction", tx.ID.String(), fmt.Sprintf("Completed draft transaction %s (type: %s, qty: %d)", tx.ReferenceNo, tx.TransactionType, tx.Qty))

	return &tx, nil
}

func (s *transactionService) ConfirmPick(c *fiber.Ctx, userID string, txID string, batchBarcode string, locationBarcode string) (*model.StockTransaction, error) {
	uID, _ := uuid.Parse(userID)
	tID, err := uuid.Parse(txID)
	if err != nil {
		return nil, fiber.NewError(fiber.StatusBadRequest, "Invalid Transaction ID")
	}

	var tx model.StockTransaction
	if err := s.DB.Preload("Batch").Preload("Batch.Product").Preload("Batch.Location").First(&tx, "id = ?", tID).Error; err != nil {
		return nil, fiber.NewError(fiber.StatusNotFound, "Picking item not found")
	}

	if tx.Status != "picking" {
		return nil, fiber.NewError(fiber.StatusBadRequest, "This item is not in picking status")
	}

	// 1. Resolve scanned location
	var loc model.Location
	if err := s.DB.First(&loc, "barcode = ?", locationBarcode).Error; err != nil {
		_ = s.bcService.Scan(locationBarcode, uID, "PICKING", "FAILED", "Wrong Location", c.IP(), c.Get("User-Agent"), nil)
		return nil, fiber.NewError(fiber.StatusBadRequest, "Wrong Location")
	}

	// Verify scanned location matches allocated location
	if tx.Batch.LocationID != loc.ID {
		_ = s.bcService.Scan(locationBarcode, uID, "PICKING", "FAILED", "Wrong Location", c.IP(), c.Get("User-Agent"), nil)
		return nil, fiber.NewError(fiber.StatusBadRequest, "Wrong Location")
	}

	// 2. Resolve scanned batch
	var batch model.InventoryBatch
	if err := s.DB.First(&batch, "barcode = ?", batchBarcode).Error; err != nil {
		_ = s.bcService.Scan(batchBarcode, uID, "PICKING", "FAILED", "Barcode Not Found", c.IP(), c.Get("User-Agent"), nil)
		return nil, fiber.NewError(fiber.StatusBadRequest, "Barcode Not Found")
	}

	// Verify scanned batch matches allocated batch
	if tx.BatchID != batch.ID {
		_ = s.bcService.Scan(batchBarcode, uID, "PICKING", "FAILED", "Incorrect Batch Picked", c.IP(), c.Get("User-Agent"), nil)
		return nil, fiber.NewError(fiber.StatusBadRequest, "Incorrect Batch Picked")
	}

	// Validation checks on batch (Expired? Active?)
	if batch.Status == "damaged" || batch.Status == "quarantine" {
		_ = s.bcService.Scan(batchBarcode, uID, "PICKING", "FAILED", "Inactive Batch", c.IP(), c.Get("User-Agent"), nil)
		return nil, fiber.NewError(fiber.StatusBadRequest, "Inactive Batch")
	}
	if batch.ExpiredDate.Before(time.Now()) || batch.Status == "expired" {
		_ = s.bcService.Scan(batchBarcode, uID, "PICKING", "FAILED", "Expired Batch", c.IP(), c.Get("User-Agent"), nil)
		return nil, fiber.NewError(fiber.StatusBadRequest, "Expired Batch")
	}

	// 3. Confirm Pick Transaction
	errTx := s.DB.Transaction(func(txDb *gorm.DB) error {
		// Deduct physical quantity and allocated quantity
		batch.Qty -= tx.Qty
		batch.AllocatedQty -= tx.Qty
		
		// If fully picked and stock is empty, we can update status, else keep stored
		if batch.Qty == 0 {
			batch.Status = "picked"
		}

		if errSaveBatch := txDb.Save(&batch).Error; errSaveBatch != nil {
			return errSaveBatch
		}

		// Update transaction status
		tx.Status = "completed"
		if errSaveTx := txDb.Save(&tx).Error; errSaveTx != nil {
			return errSaveTx
		}

		// Record Movement: location -> picked/shipped (nil ToLocationID)
		movement := model.InventoryMovement{
			BatchID:        batch.ID,
			FromLocationID: &loc.ID,
			ToLocationID:   nil, // picked for shipping
			Qty:            tx.Qty,
			MovementType:   "PICKING",
			CreatedBy:      uID,
		}
		if errMove := txDb.Create(&movement).Error; errMove != nil {
			return errMove
		}

		// Update Sales Order progress
		if tx.SOID != nil {
			var so model.SalesOrder
			if err := txDb.Preload("Items").First(&so, "id = ?", *tx.SOID).Error; err == nil {
				for i, item := range so.Items {
					if item.ProductID == batch.ProductID {
						so.Items[i].ShippedQty += tx.Qty
						txDb.Save(&so.Items[i])
						break
					}
				}
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
				txDb.Save(&so)
			}
		}

		return nil
	})

	if errTx != nil {
		_ = s.bcService.Scan(batchBarcode, uID, "PICKING", "FAILED", errTx.Error(), c.IP(), c.Get("User-Agent"), nil)
		return nil, fiber.NewError(fiber.StatusBadRequest, errTx.Error())
	}

	// Record success scan log
	_ = s.bcService.Scan(batchBarcode, uID, "PICKING", "SUCCESS", "Picked successfully", c.IP(), c.Get("User-Agent"), nil)

	// Reload with preloads
	s.DB.Preload("User").Preload("Supplier").Preload("Batch").Preload("Batch.Product").Preload("Batch.Warehouse").Preload("Batch.Location").First(&tx, tx.ID)

	LogCtxActivity(s.DB, c, "APPROVE", "picking", tx.ID.String(), fmt.Sprintf("Confirmed barcode pick for outward item ref %s, qty: %d", tx.ReferenceNo, tx.Qty))

	return &tx, nil
}
