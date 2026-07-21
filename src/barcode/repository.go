package barcode

import (
	"app/src/model"
	"gorm.io/gorm"
)

type Repository interface {
	GetDB() *gorm.DB
	GetBarcode(barcode string) (*model.BarcodeRegistry, error)
	CreateBarcode(registry *model.BarcodeRegistry) error
	CreateScanLog(log *model.ScanLog) error
	CreatePrintHistory(history *model.BarcodePrintHistory) error
	CreateMovement(movement *model.InventoryMovement) error
}

type repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) Repository {
	return &repository{db: db}
}

func (r *repository) GetDB() *gorm.DB {
	return r.db
}

func (r *repository) GetBarcode(barcode string) (*model.BarcodeRegistry, error) {
	var reg model.BarcodeRegistry
	if err := r.db.Where("barcode = ?", barcode).First(&reg).Error; err != nil {
		return nil, err
	}
	return &reg, nil
}

func (r *repository) CreateBarcode(registry *model.BarcodeRegistry) error {
	return r.db.Create(registry).Error
}

func (r *repository) CreateScanLog(log *model.ScanLog) error {
	return r.db.Create(log).Error
}

func (r *repository) CreatePrintHistory(history *model.BarcodePrintHistory) error {
	return r.db.Create(history).Error
}

func (r *repository) CreateMovement(movement *model.InventoryMovement) error {
	return r.db.Create(movement).Error
}
