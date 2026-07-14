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

type SupplierService interface {
	GetSuppliers(c *fiber.Ctx, search string) ([]model.Supplier, error)
	GetSupplierByID(c *fiber.Ctx, id string) (*model.Supplier, error)
	CreateSupplier(c *fiber.Ctx, req *validation.CreateSupplier) (*model.Supplier, error)
	UpdateSupplier(c *fiber.Ctx, id string, req *validation.UpdateSupplier) (*model.Supplier, error)
	DeleteSupplier(c *fiber.Ctx, id string) error
}

type supplierService struct {
	Log      *logrus.Logger
	DB       *gorm.DB
	Validate *validator.Validate
}

func NewSupplierService(db *gorm.DB, validate *validator.Validate) SupplierService {
	return &supplierService{
		Log:      utils.Log,
		DB:       db,
		Validate: validate,
	}
}

func (s *supplierService) GetSuppliers(c *fiber.Ctx, search string) ([]model.Supplier, error) {
	var list []model.Supplier
	query := s.DB.WithContext(c.Context()).Order("name asc")

	if search != "" {
		query = query.Where("name LIKE ? OR phone LIKE ? OR email LIKE ?", "%"+search+"%", "%"+search+"%", "%"+search+"%")
	}

	if err := query.Find(&list).Error; err != nil {
		s.Log.Errorf("Failed to query suppliers: %v", err)
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Database error")
	}

	return list, nil
}

func (s *supplierService) GetSupplierByID(c *fiber.Ctx, id string) (*model.Supplier, error) {
	var supplier model.Supplier
	uid, err := uuid.Parse(id)
	if err != nil {
		return nil, fiber.NewError(fiber.StatusBadRequest, "Invalid UUID format")
	}

	if err := s.DB.WithContext(c.Context()).First(&supplier, "id = ?", uid).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fiber.NewError(fiber.StatusNotFound, "Supplier not found")
		}
		s.Log.Errorf("Failed to query supplier by ID: %v", err)
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Database error")
	}

	return &supplier, nil
}

func (s *supplierService) CreateSupplier(c *fiber.Ctx, req *validation.CreateSupplier) (*model.Supplier, error) {
	if err := s.Validate.Struct(req); err != nil {
		return nil, err
	}

	supplier := model.Supplier{
		Name:  req.Name,
		Phone: req.Phone,
		Email: req.Email,
	}

	if err := s.DB.WithContext(c.Context()).Create(&supplier).Error; err != nil {
		s.Log.Errorf("Failed to create supplier: %v", err)
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Database error")
	}

	return &supplier, nil
}

func (s *supplierService) UpdateSupplier(c *fiber.Ctx, id string, req *validation.UpdateSupplier) (*model.Supplier, error) {
	if err := s.Validate.Struct(req); err != nil {
		return nil, err
	}

	supplier, err := s.GetSupplierByID(c, id)
	if err != nil {
		return nil, err
	}

	if req.Name != "" {
		supplier.Name = req.Name
	}
	supplier.Phone = req.Phone
	supplier.Email = req.Email

	if err := s.DB.WithContext(c.Context()).Save(supplier).Error; err != nil {
		s.Log.Errorf("Failed to update supplier: %v", err)
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Database error")
	}

	return supplier, nil
}

func (s *supplierService) DeleteSupplier(c *fiber.Ctx, id string) error {
	supplier, err := s.GetSupplierByID(c, id)
	if err != nil {
		return err
	}

	if err := s.DB.WithContext(c.Context()).Delete(supplier).Error; err != nil {
		s.Log.Errorf("Failed to delete supplier: %v", err)
		return fiber.NewError(fiber.StatusInternalServerError, "Database error")
	}

	return nil
}
