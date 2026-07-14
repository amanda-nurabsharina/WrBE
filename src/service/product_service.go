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

type ProductService interface {
	GetProducts(c *fiber.Ctx, search string) ([]model.Product, error)
	GetProductByID(c *fiber.Ctx, id string) (*model.Product, error)
	CreateProduct(c *fiber.Ctx, req *validation.CreateProduct) (*model.Product, error)
	UpdateProduct(c *fiber.Ctx, id string, req *validation.UpdateProduct) (*model.Product, error)
	DeleteProduct(c *fiber.Ctx, id string) error
}

type productService struct {
	Log      *logrus.Logger
	DB       *gorm.DB
	Validate *validator.Validate
}

func NewProductService(db *gorm.DB, validate *validator.Validate) ProductService {
	return &productService{
		Log:      utils.Log,
		DB:       db,
		Validate: validate,
	}
}

func (s *productService) GetProducts(c *fiber.Ctx, search string) ([]model.Product, error) {
	var products []model.Product
	query := s.DB.WithContext(c.Context()).Model(&model.Product{}).Order("code asc")

	if search != "" {
		query = query.Where("code LIKE ? OR name LIKE ? OR category_id LIKE ?", "%"+search+"%", "%"+search+"%", "%"+search+"%")
	}

	if err := query.Find(&products).Error; err != nil {
		s.Log.Errorf("Failed to query products: %v", err)
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Database error")
	}

	return products, nil
}

func (s *productService) GetProductByID(c *fiber.Ctx, id string) (*model.Product, error) {
	var product model.Product
	uid, err := uuid.Parse(id)
	if err != nil {
		return nil, fiber.NewError(fiber.StatusBadRequest, "Invalid UUID format")
	}

	if err := s.DB.WithContext(c.Context()).First(&product, "id = ?", uid).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fiber.NewError(fiber.StatusNotFound, "Product not found")
		}
		s.Log.Errorf("Failed to query product: %v", err)
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Database error")
	}

	return &product, nil
}

func (s *productService) CreateProduct(c *fiber.Ctx, req *validation.CreateProduct) (*model.Product, error) {
	if err := s.Validate.Struct(req); err != nil {
		return nil, err
	}

	// Check if product code already exists
	var count int64
	s.DB.WithContext(c.Context()).Model(&model.Product{}).Where("code = ?", req.Code).Count(&count)
	if count > 0 {
		return nil, fiber.NewError(fiber.StatusBadRequest, "Product code already exists")
	}

	product := model.Product{
		Code:         req.Code,
		Barcode:      req.Barcode,
		Name:         req.Name,
		CategoryID:   req.CategoryID,
		Unit:         req.Unit,
		MinimumStock: req.MinimumStock,
	}

	if err := s.DB.WithContext(c.Context()).Create(&product).Error; err != nil {
		s.Log.Errorf("Failed to create product: %v", err)
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Database error")
	}

	return &product, nil
}

func (s *productService) UpdateProduct(c *fiber.Ctx, id string, req *validation.UpdateProduct) (*model.Product, error) {
	if err := s.Validate.Struct(req); err != nil {
		return nil, err
	}

	product, err := s.GetProductByID(c, id)
	if err != nil {
		return nil, err
	}

	if req.Name != "" {
		product.Name = req.Name
	}
	if req.Barcode != "" {
		product.Barcode = req.Barcode
	}
	if req.CategoryID != "" {
		product.CategoryID = req.CategoryID
	}
	if req.Unit != "" {
		product.Unit = req.Unit
	}
	if req.MinimumStock != nil {
		product.MinimumStock = *req.MinimumStock
	}

	if err := s.DB.WithContext(c.Context()).Save(product).Error; err != nil {
		s.Log.Errorf("Failed to update product: %v", err)
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Database error")
	}

	return product, nil
}

func (s *productService) DeleteProduct(c *fiber.Ctx, id string) error {
	product, err := s.GetProductByID(c, id)
	if err != nil {
		return err
	}

	// Check if there are any batches referencing this product
	var count int64
	s.DB.WithContext(c.Context()).Model(&model.InventoryBatch{}).Where("product_id = ?", product.ID).Count(&count)
	if count > 0 {
		return fiber.NewError(fiber.StatusBadRequest, "Cannot delete product. It has associated inventory batches.")
	}

	if err := s.DB.WithContext(c.Context()).Delete(product).Error; err != nil {
		s.Log.Errorf("Failed to delete product: %v", err)
		return fiber.NewError(fiber.StatusInternalServerError, "Database error")
	}

	return nil
}
