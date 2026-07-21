package barcode

import (
	"app/src/model"
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Service interface {
	Lookup(barcode string) (*model.BarcodeRegistry, interface{}, error)
	Validate(barcode string, expectedWarehouseID *uuid.UUID, expectedLocationID *uuid.UUID) (*model.BarcodeRegistry, interface{}, error)
	ValidatePick(barcode string, expectedWarehouseID *uuid.UUID, expectedLocationID *uuid.UUID) (*model.BarcodeRegistry, *model.InventoryBatch, error)
	Scan(barcode string, userID uuid.UUID, action string, status string, message string, ip string, device string, sessionID *uuid.UUID) error
	Print(barcode string, labelType string, userID uuid.UUID, qty int, reason string) error
	GenerateBarcode(db *gorm.DB, prefix string, referenceID uuid.UUID) (string, error)
	PutAway(userID uuid.UUID, batchBarcode string, locationBarcode string) (*model.InventoryBatch, error)
}

type service struct {
	repo Repository
	gen  Generator
	val  Validator
}

func NewService(repo Repository, gen Generator, val Validator) Service {
	return &service{
		repo: repo,
		gen:  gen,
		val:  val,
	}
}

func (s *service) Lookup(barcode string) (*model.BarcodeRegistry, interface{}, error) {
	reg, err := s.repo.GetBarcode(barcode)
	if err != nil {
		return nil, nil, errors.New("Barcode Not Found")
	}

	var entity interface{}
	db := s.repo.GetDB()

	switch reg.Type {
	case "PRODUCT":
		var prod model.Product
		if err := db.Preload("PackagingUnit").Where("id = ?", reg.ReferenceID).First(&prod).Error; err != nil {
			return nil, nil, errors.New("Product Not Found")
		}
		entity = &prod
	case "BATCH":
		var batch model.InventoryBatch
		if err := db.Preload("Product").Preload("Warehouse").Preload("Location").Where("id = ?", reg.ReferenceID).First(&batch).Error; err != nil {
			return nil, nil, errors.New("Batch Not Found")
		}
		entity = &batch
	case "LOCATION":
		var loc model.Location
		if err := db.Where("id = ?", reg.ReferenceID).First(&loc).Error; err != nil {
			return nil, nil, errors.New("Location Not Found")
		}
		entity = &loc
	default:
		return nil, nil, errors.New("Invalid Barcode Type")
	}

	return reg, entity, nil
}

func (s *service) Validate(barcode string, expectedWarehouseID *uuid.UUID, expectedLocationID *uuid.UUID) (*model.BarcodeRegistry, interface{}, error) {
	reg, entity, err := s.Lookup(barcode)
	if err != nil {
		return nil, nil, err
	}

	if reg.Type == "BATCH" {
		db := s.repo.GetDB()
		batch, errVal := s.val.ValidateBatch(db, reg.ReferenceID, expectedWarehouseID, expectedLocationID)
		if errVal != nil {
			return nil, nil, errVal
		}
		entity = batch
	}

	return reg, entity, nil
}

func (s *service) Scan(barcode string, userID uuid.UUID, action string, status string, message string, ip string, device string, sessionID *uuid.UUID) error {
	log := model.ScanLog{
		SessionID: sessionID,
		Barcode:   barcode,
		UserID:    userID,
		Action:    action,
		Status:    status,
		Message:   message,
		IP:        ip,
		Device:    device,
		CreatedAt: time.Now(),
	}
	return s.repo.CreateScanLog(&log)
}

func (s *service) Print(barcode string, labelType string, userID uuid.UUID, qty int, reason string) error {
	hist := model.BarcodePrintHistory{
		Barcode:   barcode,
		LabelType: labelType,
		UserID:    userID,
		Qty:       qty,
		Reason:    reason,
		CreatedAt: time.Now(),
	}
	return s.repo.CreatePrintHistory(&hist)
}

func (s *service) GenerateBarcode(db *gorm.DB, prefix string, referenceID uuid.UUID) (string, error) {
	barcodeStr, err := s.gen.Generate(db, prefix)
	if err != nil {
		return "", err
	}

	reg := model.BarcodeRegistry{
		Barcode:     barcodeStr,
		Type:        s.getBarcodeType(prefix),
		ReferenceID: referenceID,
		CreatedAt:   time.Now(),
	}

	if err := db.Create(&reg).Error; err != nil {
		return "", err
	}

	return barcodeStr, nil
}

func (s *service) getBarcodeType(prefix string) string {
	switch prefix {
	case "PRD":
		return "PRODUCT"
	case "BAT":
		return "BATCH"
	case "LOC":
		return "LOCATION"
	default:
		return "UNKNOWN"
	}
}

func (s *service) PutAway(userID uuid.UUID, batchBarcode string, locationBarcode string) (*model.InventoryBatch, error) {
	db := s.repo.GetDB()

	// 1. Resolve Location
	locReg, locEntity, err := s.Lookup(locationBarcode)
	if err != nil || locReg.Type != "LOCATION" {
		return nil, errors.New("Wrong Location")
	}
	location := locEntity.(*model.Location)

	// 2. Resolve Batch
	batchReg, batchEntity, err := s.Lookup(batchBarcode)
	if err != nil || batchReg.Type != "BATCH" {
		return nil, errors.New("Barcode Not Found")
	}
	batch := batchEntity.(*model.InventoryBatch)

	// 3. Put Away Validation Hook
	if errVal := s.val.CanStoreProductAtLocation(&batch.Product, location); errVal != nil {
		_ = s.Scan(batchBarcode, userID, "PUT_AWAY", "FAILED", errVal.Error(), "", "", nil)
		return nil, errVal
	}

	// 4. Perform Relocation Transaction
	errTx := db.Transaction(func(txDb *gorm.DB) error {
		fromLocID := batch.LocationID

		// Update batch location and status
		batch.WarehouseID = location.WarehouseID
		batch.LocationID = location.ID
		batch.Status = "stored"

		if errSave := txDb.Save(batch).Error; errSave != nil {
			return errSave
		}

		// Record InventoryMovement
		movement := model.InventoryMovement{
			BatchID:        batch.ID,
			FromLocationID: &fromLocID,
			ToLocationID:   &location.ID,
			Qty:            batch.Qty,
			MovementType:   "PUT_AWAY",
			CreatedBy:      userID,
		}
		if errMove := txDb.Create(&movement).Error; errMove != nil {
			return errMove
		}

		return nil
	})

	if errTx != nil {
		_ = s.Scan(batchBarcode, userID, "PUT_AWAY", "FAILED", errTx.Error(), "", "", nil)
		return nil, errTx
	}

	_ = s.Scan(batchBarcode, userID, "PUT_AWAY", "SUCCESS", "Put Away completed successfully", "", "", nil)

	return batch, nil
}

func (s *service) ValidatePick(barcode string, expectedWarehouseID *uuid.UUID, expectedLocationID *uuid.UUID) (*model.BarcodeRegistry, *model.InventoryBatch, error) {
	reg, entity, err := s.Validate(barcode, expectedWarehouseID, expectedLocationID)
	if err != nil {
		return nil, nil, err
	}

	if reg.Type != "BATCH" {
		return nil, nil, errors.New("Barcode is not a Batch barcode")
	}

	batch := entity.(*model.InventoryBatch)
	db := s.repo.GetDB()
	if errFEFO := s.val.ValidateFEFOPicking(db, batch); errFEFO != nil {
		return reg, batch, errFEFO
	}

	return reg, batch, nil
}
