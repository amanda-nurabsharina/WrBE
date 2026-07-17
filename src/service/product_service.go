package service

import (
	"app/src/model"
	"app/src/utils"
	"app/src/validation"
	"errors"
	"time"

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
	query := s.DB.WithContext(c.Context()).Model(&model.Product{}).Preload("PackagingUnit").Order("code asc")

	if search != "" {
		query = query.Where("code LIKE ? OR name LIKE ? OR category_id LIKE ? OR sub_category LIKE ?", "%"+search+"%", "%"+search+"%", "%"+search+"%", "%"+search+"%")
	}

	if err := query.Find(&products).Error; err != nil {
		s.Log.Errorf("Failed to query products: %v", err)
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Database error")
	}

	// Calculate and populate total stock
	type ProductStock struct {
		ProductID uuid.UUID
		TotalQty  int
	}
	var stocks []ProductStock
	if len(products) > 0 {
		if err := s.DB.WithContext(c.Context()).Model(&model.InventoryBatch{}).
			Where("status = 'active'").
			Select("product_id, SUM(qty) as total_qty").
			Group("product_id").
			Scan(&stocks).Error; err != nil {
			s.Log.Warnf("Failed to query product stock: %v", err)
		}
	}

	stockMap := make(map[uuid.UUID]int)
	for _, st := range stocks {
		stockMap[st.ProductID] = st.TotalQty
	}

	for i := range products {
		products[i].Stock = stockMap[products[i].ID]
	}

	return products, nil
}

func (s *productService) GetProductByID(c *fiber.Ctx, id string) (*model.Product, error) {
	var product model.Product
	uid, err := uuid.Parse(id)
	if err != nil {
		return nil, fiber.NewError(fiber.StatusBadRequest, "Invalid UUID format")
	}

	if err := s.DB.WithContext(c.Context()).Preload("PackagingUnit").First(&product, "id = ?", uid).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fiber.NewError(fiber.StatusNotFound, "Product not found")
		}
		s.Log.Errorf("Failed to query product: %v", err)
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Database error")
	}

	// Fetch stock
	var totalStock int
	s.DB.WithContext(c.Context()).Model(&model.InventoryBatch{}).
		Where("product_id = ? AND status = 'active'", product.ID).
		Select("COALESCE(SUM(qty), 0)").
		Scan(&totalStock)
	product.Stock = totalStock

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

	packUUID, err := uuid.Parse(req.PackagingUnitID)
	if err != nil {
		return nil, fiber.NewError(fiber.StatusBadRequest, "Invalid Packaging Unit UUID format")
	}

	product := model.Product{
		Code:             req.Code,
		Barcode:          req.Barcode,
		Name:             req.Name,
		CategoryID:       req.CategoryID,
		Unit:             req.Unit,
		MinimumStock:     req.MinimumStock,
		RegCategory:      req.RegCategory,
		KementanRegNo:    req.KementanRegNo,
		MSDSReference:    req.MSDSReference,
		SubCategory:      req.SubCategory,
		PackagingUnitID:  packUUID,
		ConversionRatio:  req.ConversionRatio,
		PurchasePrice:    req.PurchasePrice,
		PriceDistributor: req.PriceDistributor,
		PriceRetail:      req.PriceRetail,
		StorageTemp:         req.StorageTemp,
		StorageHumidity:     req.StorageHumidity,
		StorageRestrictions: req.StorageRestrictions,
	}

	if err := s.DB.WithContext(c.Context()).Create(&product).Error; err != nil {
		s.Log.Errorf("Failed to create product: %v", err)
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Database error")
	}

	if req.InitialBatchNo != "" {
		warehouseUUID, err := uuid.Parse(req.InitialWarehouseID)
		if err == nil {
			var loc model.Location
			s.DB.WithContext(c.Context()).Where("warehouse_id = ?", warehouseUUID).First(&loc)
			if loc.ID == uuid.Nil {
				loc = model.Location{
					WarehouseID: warehouseUUID,
					Rack:        "Rack-A1",
				}
				s.DB.WithContext(c.Context()).Create(&loc)
			}

			parsedExpDate, errDate := time.Parse("2006-01-02", req.InitialExpiryDate)
			if errDate != nil || parsedExpDate.IsZero() {
				parsedExpDate = time.Now().AddDate(2, 0, 0)
			}

			batch := model.InventoryBatch{
				ProductID:     product.ID,
				BatchNumber:   req.InitialBatchNo,
				ExpiredDate:   parsedExpDate,
				Qty:           req.InitialQty,
				WarehouseID:   warehouseUUID,
				LocationID:    loc.ID,
				PurchasePrice: req.PurchasePrice,
				Status:        "active",
			}
			if errBatch := s.DB.WithContext(c.Context()).Create(&batch).Error; errBatch == nil {
				var userID uuid.UUID
				userObj := c.Locals("user")
				if user, ok := userObj.(*model.User); ok && user != nil {
					userID = user.ID
				} else {
					var firstUser model.User
					s.DB.WithContext(c.Context()).First(&firstUser)
					userID = firstUser.ID
				}

				tx := model.StockTransaction{
					BatchID:         batch.ID,
					TransactionType: "IN",
					Qty:             req.InitialQty,
					ReferenceNo:     "INITIAL_STOCK",
					UserID:          userID,
				}
				s.DB.WithContext(c.Context()).Create(&tx)
			}
		}
	}

	// Preload the PackagingUnit relation
	s.DB.WithContext(c.Context()).Preload("PackagingUnit").First(&product, "id = ?", product.ID)

	LogCtxActivity(s.DB, c, "CREATE", "products", product.ID.String(), "Created product: "+product.Name+" ("+product.Code+")")

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
	if req.RegCategory != "" {
		product.RegCategory = req.RegCategory
	}
	if req.KementanRegNo != "" {
		product.KementanRegNo = req.KementanRegNo
	}
	if req.MSDSReference != "" {
		product.MSDSReference = req.MSDSReference
	}
	if req.SubCategory != "" {
		product.SubCategory = req.SubCategory
	}
	if req.PackagingUnitID != "" {
		packUUID, err := uuid.Parse(req.PackagingUnitID)
		if err != nil {
			return nil, fiber.NewError(fiber.StatusBadRequest, "Invalid Packaging Unit UUID format")
		}
		product.PackagingUnitID = packUUID
	}
	if req.ConversionRatio != nil {
		product.ConversionRatio = *req.ConversionRatio
	}
	if req.PurchasePrice != nil {
		product.PurchasePrice = *req.PurchasePrice
	}
	if req.PriceDistributor != nil {
		product.PriceDistributor = *req.PriceDistributor
	}
	if req.PriceRetail != nil {
		product.PriceRetail = *req.PriceRetail
	}
	if req.StorageTemp != "" {
		product.StorageTemp = req.StorageTemp
	}
	if req.StorageHumidity != "" {
		product.StorageHumidity = req.StorageHumidity
	}
	if req.StorageRestrictions != "" {
		product.StorageRestrictions = req.StorageRestrictions
	}

	if err := s.DB.WithContext(c.Context()).Save(product).Error; err != nil {
		s.Log.Errorf("Failed to update product: %v", err)
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Database error")
	}

	// Preload the PackagingUnit relation
	s.DB.WithContext(c.Context()).Preload("PackagingUnit").First(product, "id = ?", product.ID)

	LogCtxActivity(s.DB, c, "UPDATE", "products", product.ID.String(), "Updated product: "+product.Name+" ("+product.Code+")")

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

	LogCtxActivity(s.DB, c, "DELETE", "products", product.ID.String(), "Deleted product: "+product.Name+" ("+product.Code+")")

	return nil
}
