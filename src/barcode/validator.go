package barcode

import (
	"app/src/model"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Validator interface {
	ValidateBatch(db *gorm.DB, batchID uuid.UUID, expectedWarehouseID *uuid.UUID, expectedLocationID *uuid.UUID) (*model.InventoryBatch, error)
	CanStoreProductAtLocation(prod *model.Product, loc *model.Location) error
	ValidateFEFOPicking(db *gorm.DB, scannedBatch *model.InventoryBatch) error
}

type validator struct{}

func NewValidator() Validator {
	return &validator{}
}

func (v *validator) ValidateBatch(db *gorm.DB, batchID uuid.UUID, expectedWarehouseID *uuid.UUID, expectedLocationID *uuid.UUID) (*model.InventoryBatch, error) {
	var batch model.InventoryBatch
	if err := db.Preload("Product").Preload("Warehouse").Preload("Location").Where("id = ?", batchID).First(&batch).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("Barcode Not Found")
		}
		return nil, err
	}

	if batch.Status == "damaged" || batch.Status == "quarantine" {
		return nil, errors.New("Inactive Batch")
	}

	if batch.ExpiredDate.Before(time.Now()) || batch.Status == "expired" {
		return nil, errors.New("Expired Batch")
	}

	if expectedWarehouseID != nil && batch.WarehouseID != *expectedWarehouseID {
		return nil, errors.New("Wrong Warehouse")
	}

	if expectedLocationID != nil && batch.LocationID != *expectedLocationID {
		return nil, errors.New("Wrong Location")
	}

	return &batch, nil
}

func (v *validator) CanStoreProductAtLocation(prod *model.Product, loc *model.Location) error {
	// Hook validation: B3 category products can only be placed on B-zone Racks (e.g. Rack-B1)
	if prod.RegCategory == "B3" && !strings.Contains(strings.ToLower(loc.Rack), "rack-b") {
		return errors.New("Hazardous Material (B3) must be stored in B-zone racks only")
	}
	return nil
}

func (v *validator) ValidateFEFOPicking(db *gorm.DB, scannedBatch *model.InventoryBatch) error {
	var earliestBatch model.InventoryBatch
	err := db.Where("product_id = ? AND status = 'stored' AND available_qty > 0", scannedBatch.ProductID).
		Order("expired_date ASC, created_at ASC").
		First(&earliestBatch).Error

	if err == nil && earliestBatch.ID != scannedBatch.ID {
		if earliestBatch.ExpiredDate.Before(scannedBatch.ExpiredDate) {
			return errors.New("FEFO Warning: Batch #" + earliestBatch.BatchNumber + " (Exp: " + earliestBatch.ExpiredDate.Format("2006-01-02") + ") expires sooner than scanned batch")
		}
	}
	return nil
}
