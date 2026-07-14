package service

import (
	"app/src/model"
	"app/src/utils"
	"errors"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type BatchService interface {
	GetBatches(c *fiber.Ctx, search string, productID string, status string, expiryDays string) ([]model.InventoryBatch, error)
	GetBatchByID(c *fiber.Ctx, id string) (*model.InventoryBatch, error)
	UpdateBatchStatus(c *fiber.Ctx, id string, status string) (*model.InventoryBatch, error)
	GetExpiryAlerts(c *fiber.Ctx) ([]model.InventoryBatch, error)
}

type batchService struct {
	Log      *logrus.Logger
	DB       *gorm.DB
	Validate *validator.Validate
}

func NewBatchService(db *gorm.DB, validate *validator.Validate) BatchService {
	return &batchService{
		Log:      utils.Log,
		DB:       db,
		Validate: validate,
	}
}

func (s *batchService) GetBatches(c *fiber.Ctx, search string, productID string, status string, expiryDays string) ([]model.InventoryBatch, error) {
	var batches []model.InventoryBatch
	query := s.DB.WithContext(c.Context()).
		Preload("Product").
		Preload("Warehouse").
		Preload("Location").
		Order("expired_date asc")

	// Filter by search (batch number or product name)
	if search != "" {
		query = query.Joins("Join products On products.id = inventory_batches.product_id").
			Where("inventory_batches.batch_number LIKE ? OR products.name LIKE ?", "%"+search+"%", "%"+search+"%")
	}

	// Filter by product
	if productID != "" {
		pid, err := uuid.Parse(productID)
		if err == nil {
			query = query.Where("inventory_batches.product_id = ?", pid)
		}
	}

	// Filter by status
	if status != "" {
		query = query.Where("inventory_batches.status = ?", status)
	}

	// Filter by expiration window (days)
	if expiryDays != "" {
		now := time.Now()
		switch expiryDays {
		case "expired":
			query = query.Where("inventory_batches.expired_date <= ?", now)
		case "7":
			query = query.Where("inventory_batches.expired_date > ? And inventory_batches.expired_date <= ?", now, now.AddDate(0, 0, 7))
		case "30":
			query = query.Where("inventory_batches.expired_date > ? And inventory_batches.expired_date <= ?", now, now.AddDate(0, 0, 30))
		case "60":
			query = query.Where("inventory_batches.expired_date > ? And inventory_batches.expired_date <= ?", now, now.AddDate(0, 0, 60))
		case "90":
			query = query.Where("inventory_batches.expired_date > ? And inventory_batches.expired_date <= ?", now, now.AddDate(0, 0, 90))
		}
	}

	if err := query.Find(&batches).Error; err != nil {
		s.Log.Errorf("Failed to query batches: %v", err)
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Database error")
	}

	return batches, nil
}

func (s *batchService) GetBatchByID(c *fiber.Ctx, id string) (*model.InventoryBatch, error) {
	var batch model.InventoryBatch
	uid, err := uuid.Parse(id)
	if err != nil {
		return nil, fiber.NewError(fiber.StatusBadRequest, "Invalid UUID format")
	}

	if err := s.DB.WithContext(c.Context()).
		Preload("Product").
		Preload("Warehouse").
		Preload("Location").
		First(&batch, "id = ?", uid).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fiber.NewError(fiber.StatusNotFound, "Inventory batch not found")
		}
		s.Log.Errorf("Failed to query batch by ID: %v", err)
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Database error")
	}

	return &batch, nil
}

func (s *batchService) UpdateBatchStatus(c *fiber.Ctx, id string, status string) (*model.InventoryBatch, error) {
	batch, err := s.GetBatchByID(c, id)
	if err != nil {
		return nil, err
	}

	batch.Status = status
	if err := s.DB.WithContext(c.Context()).Save(batch).Error; err != nil {
		s.Log.Errorf("Failed to update batch status: %v", err)
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Database error")
	}

	return batch, nil
}

func (s *batchService) GetExpiryAlerts(c *fiber.Ctx) ([]model.InventoryBatch, error) {
	var batches []model.InventoryBatch
	now := time.Now()
	// Query active batches with qty > 0 expiring in less than 90 days
	err := s.DB.WithContext(c.Context()).
		Preload("Product").
		Where("qty > 0 And (status = ? OR expired_date <= ?)", "expired", now.AddDate(0, 0, 90)).
		Order("expired_date asc").
		Find(&batches).Error

	if err != nil {
		s.Log.Errorf("Failed to query expiry alerts: %v", err)
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Database error")
	}

	return batches, nil
}
