package service

import (
	"app/src/model"
	"app/src/utils"
	"app/src/validation"
	"errors"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type PackagingService interface {
	GetPackagingUnits(c *fiber.Ctx, search string) ([]model.PackagingUnit, error)
	GetPackagingUnitByID(c *fiber.Ctx, id string) (*model.PackagingUnit, error)
	CreatePackagingUnit(c *fiber.Ctx, req *validation.CreatePackagingUnit) (*model.PackagingUnit, error)
	UpdatePackagingUnit(c *fiber.Ctx, id string, req *validation.UpdatePackagingUnit) (*model.PackagingUnit, error)
	DeletePackagingUnit(c *fiber.Ctx, id string) error
}

type packagingService struct {
	Log      *logrus.Logger
	DB       *gorm.DB
	Validate *validator.Validate
}

func NewPackagingService(db *gorm.DB, validate *validator.Validate) PackagingService {
	return &packagingService{
		Log:      utils.Log,
		DB:       db,
		Validate: validate,
	}
}

func (s *packagingService) GetPackagingUnits(c *fiber.Ctx, search string) ([]model.PackagingUnit, error) {
	var list []model.PackagingUnit
	query := s.DB.WithContext(c.Context()).Order("name asc")

	if search != "" {
		query = query.Where("code LIKE ? OR name LIKE ? OR description LIKE ?", "%"+search+"%", "%"+search+"%", "%"+search+"%")
	}

	if err := query.Find(&list).Error; err != nil {
		s.Log.Errorf("Failed to query packaging units: %v", err)
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Database error")
	}

	return list, nil
}

func (s *packagingService) GetPackagingUnitByID(c *fiber.Ctx, id string) (*model.PackagingUnit, error) {
	var packaging model.PackagingUnit
	uid, err := uuid.Parse(id)
	if err != nil {
		return nil, fiber.NewError(fiber.StatusBadRequest, "Invalid UUID format")
	}

	if err := s.DB.WithContext(c.Context()).First(&packaging, "id = ?", uid).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fiber.NewError(fiber.StatusNotFound, "Packaging unit not found")
		}
		s.Log.Errorf("Failed to query packaging unit by ID: %v", err)
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Database error")
	}

	return &packaging, nil
}

func (s *packagingService) CreatePackagingUnit(c *fiber.Ctx, req *validation.CreatePackagingUnit) (*model.PackagingUnit, error) {
	if err := s.Validate.Struct(req); err != nil {
		return nil, err
	}

	// Check if packaging unit code already exists
	var count int64
	s.DB.WithContext(c.Context()).Model(&model.PackagingUnit{}).Where("code = ?", req.Code).Count(&count)
	if count > 0 {
		return nil, fiber.NewError(fiber.StatusBadRequest, "Packaging unit code already exists")
	}

	packaging := model.PackagingUnit{
		Code:        req.Code,
		Name:        req.Name,
		Description: req.Description,
	}

	if err := s.DB.WithContext(c.Context()).Create(&packaging).Error; err != nil {
		s.Log.Errorf("Failed to create packaging unit: %v", err)
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Database error")
	}

	return &packaging, nil
}

func (s *packagingService) UpdatePackagingUnit(c *fiber.Ctx, id string, req *validation.UpdatePackagingUnit) (*model.PackagingUnit, error) {
	if err := s.Validate.Struct(req); err != nil {
		return nil, err
	}

	packaging, err := s.GetPackagingUnitByID(c, id)
	if err != nil {
		return nil, err
	}

	if req.Name != "" {
		packaging.Name = req.Name
	}
	packaging.Description = req.Description

	if err := s.DB.WithContext(c.Context()).Save(packaging).Error; err != nil {
		s.Log.Errorf("Failed to update packaging unit: %v", err)
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Database error")
	}

	return packaging, nil
}

func (s *packagingService) DeletePackagingUnit(c *fiber.Ctx, id string) error {
	packaging, err := s.GetPackagingUnitByID(c, id)
	if err != nil {
		return err
	}

	// Check if any products reference this packaging unit
	var count int64
	s.DB.WithContext(c.Context()).Model(&model.Product{}).Where("packaging_unit_id = ?", packaging.ID).Count(&count)
	if count > 0 {
		return fiber.NewError(fiber.StatusBadRequest, "Cannot delete packaging unit. It is currently associated with one or more products.")
	}

	if err := s.DB.WithContext(c.Context()).Delete(packaging).Error; err != nil {
		s.Log.Errorf("Failed to delete packaging unit: %v", err)
		return fiber.NewError(fiber.StatusInternalServerError, "Database error")
	}

	return nil
}
