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

type OrderService interface {
	// Purchase Orders
	GetPurchaseOrders(c *fiber.Ctx, search string) ([]model.PurchaseOrder, error)
	CreatePurchaseOrder(c *fiber.Ctx, req *validation.CreatePurchaseOrder) (*model.PurchaseOrder, error)
	ApprovePurchaseOrder(c *fiber.Ctx, id string) (*model.PurchaseOrder, error)

	// Sales Orders
	GetSalesOrders(c *fiber.Ctx, search string) ([]model.SalesOrder, error)
	CreateSalesOrder(c *fiber.Ctx, req *validation.CreateSalesOrder) (*model.SalesOrder, error)
	ApproveSalesOrder(c *fiber.Ctx, id string) (*model.SalesOrder, error)
}

type orderService struct {
	Log      *logrus.Logger
	DB       *gorm.DB
	Validate *validator.Validate
}

func NewOrderService(db *gorm.DB, validate *validator.Validate) OrderService {
	return &orderService{
		Log:      utils.Log,
		DB:       db,
		Validate: validate,
	}
}

// ----------------------------------------------------------------------------
// Purchase Orders Service
// ----------------------------------------------------------------------------

func (s *orderService) GetPurchaseOrders(c *fiber.Ctx, search string) ([]model.PurchaseOrder, error) {
	var list []model.PurchaseOrder
	query := s.DB.WithContext(c.Context()).
		Preload("Supplier").
		Preload("Items.Product").
		Order("created_at desc")

	if search != "" {
		query = query.Joins("JOIN suppliers ON suppliers.id = purchase_orders.supplier_id").
			Where("purchase_orders.po_number LIKE ? OR suppliers.name LIKE ?", "%"+search+"%", "%"+search+"%")
	}

	if err := query.Find(&list).Error; err != nil {
		s.Log.Errorf("Failed to query POs: %v", err)
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Database error")
	}

	return list, nil
}

func (s *orderService) CreatePurchaseOrder(c *fiber.Ctx, req *validation.CreatePurchaseOrder) (*model.PurchaseOrder, error) {
	if err := s.Validate.Struct(req); err != nil {
		return nil, err
	}

	supID, err := uuid.Parse(req.SupplierID)
	if err != nil {
		return nil, fiber.NewError(fiber.StatusBadRequest, "Invalid Supplier ID")
	}

	orderDate, err := time.Parse("2006-01-02", req.OrderDate)
	if err != nil {
		return nil, fiber.NewError(fiber.StatusBadRequest, "Invalid Order Date")
	}

	var items []model.PurchaseOrderItem
	for _, it := range req.Items {
		pID, err := uuid.Parse(it.ProductID)
		if err != nil {
			return nil, fiber.NewError(fiber.StatusBadRequest, "Invalid Product ID: "+it.ProductID)
		}
		items = append(items, model.PurchaseOrderItem{
			ProductID:   pID,
			Qty:         it.Qty,
			ReceivedQty: 0,
			Price:       it.Price,
		})
	}

	po := model.PurchaseOrder{
		PONumber:   req.PONumber,
		SupplierID: supID,
		OrderDate:  orderDate,
		Status:     "draft",
		Items:      items,
	}

	if err := s.DB.WithContext(c.Context()).Create(&po).Error; err != nil {
		s.Log.Errorf("Failed to create PO: %v", err)
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Failed to create PO")
	}

	// Reload PO with preloads
	s.DB.WithContext(c.Context()).Preload("Supplier").Preload("Items.Product").First(&po, po.ID)

	LogCtxActivity(s.DB, c, "CREATE", "purchase-orders", po.ID.String(), "Created PO: "+po.PONumber)

	return &po, nil
}

func (s *orderService) ApprovePurchaseOrder(c *fiber.Ctx, id string) (*model.PurchaseOrder, error) {
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

	uid, err := uuid.Parse(id)
	if err != nil {
		return nil, fiber.NewError(fiber.StatusBadRequest, "Invalid UUID")
	}

	var po model.PurchaseOrder
	if err := s.DB.WithContext(c.Context()).First(&po, "id = ?", uid).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fiber.NewError(fiber.StatusNotFound, "PO not found")
		}
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Database error")
	}

	if po.Status != "draft" {
		return nil, fiber.NewError(fiber.StatusBadRequest, "Only draft POs can be approved")
	}

	po.Status = "approved"
	if err := s.DB.WithContext(c.Context()).Save(&po).Error; err != nil {
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Failed to approve PO")
	}

	s.DB.WithContext(c.Context()).Preload("Supplier").Preload("Items.Product").First(&po, po.ID)

	LogCtxActivity(s.DB, c, "APPROVE", "purchase-orders", po.ID.String(), "Approved PO: "+po.PONumber)

	return &po, nil
}

// ----------------------------------------------------------------------------
// Sales Orders Service
// ----------------------------------------------------------------------------

func (s *orderService) GetSalesOrders(c *fiber.Ctx, search string) ([]model.SalesOrder, error) {
	var list []model.SalesOrder
	query := s.DB.WithContext(c.Context()).
		Preload("Customer").
		Preload("Items.Product").
		Order("created_at desc")

	if search != "" {
		query = query.Joins("JOIN customers ON customers.id = sales_orders.customer_id").
			Where("sales_orders.so_number LIKE ? OR customers.name LIKE ?", "%"+search+"%", "%"+search+"%")
	}

	if err := query.Find(&list).Error; err != nil {
		s.Log.Errorf("Failed to query SOs: %v", err)
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Database error")
	}

	return list, nil
}

func (s *orderService) CreateSalesOrder(c *fiber.Ctx, req *validation.CreateSalesOrder) (*model.SalesOrder, error) {
	if err := s.Validate.Struct(req); err != nil {
		return nil, err
	}

	custID, err := uuid.Parse(req.CustomerID)
	if err != nil {
		return nil, fiber.NewError(fiber.StatusBadRequest, "Invalid Customer ID")
	}

	orderDate, err := time.Parse("2006-01-02", req.OrderDate)
	if err != nil {
		return nil, fiber.NewError(fiber.StatusBadRequest, "Invalid Order Date")
	}

	var items []model.SalesOrderItem
	hasB3Product := false

	for _, it := range req.Items {
		pID, err := uuid.Parse(it.ProductID)
		if err != nil {
			return nil, fiber.NewError(fiber.StatusBadRequest, "Invalid Product ID: "+it.ProductID)
		}

		// Retrieve product to check B3 regulation category
		var prod model.Product
		if errProd := s.DB.First(&prod, "id = ?", pID).Error; errProd == nil {
			if prod.RegCategory == "B3" {
				hasB3Product = true
			}
		}

		items = append(items, model.SalesOrderItem{
			ProductID:  pID,
			Qty:        it.Qty,
			ShippedQty: 0,
			Price:      it.Price,
		})
	}

	status := "draft"
	if hasB3Product {
		status = "pending_b3_approval"
	}

	so := model.SalesOrder{
		SONumber:   req.SONumber,
		CustomerID: custID,
		OrderDate:  orderDate,
		Status:     status,
		Items:      items,
	}

	if err := s.DB.WithContext(c.Context()).Create(&so).Error; err != nil {
		s.Log.Errorf("Failed to create SO: %v", err)
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Failed to create SO")
	}

	// Reload SO with preloads
	s.DB.WithContext(c.Context()).Preload("Customer").Preload("Items.Product").First(&so, so.ID)

	LogCtxActivity(s.DB, c, "CREATE", "sales-orders", so.ID.String(), "Created SO: "+so.SONumber)

	return &so, nil
}

func (s *orderService) ApproveSalesOrder(c *fiber.Ctx, id string) (*model.SalesOrder, error) {
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

	uid, err := uuid.Parse(id)
	if err != nil {
		return nil, fiber.NewError(fiber.StatusBadRequest, "Invalid UUID")
	}

	var so model.SalesOrder
	if err := s.DB.WithContext(c.Context()).First(&so, "id = ?", uid).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fiber.NewError(fiber.StatusNotFound, "SO not found")
		}
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Database error")
	}

	if so.Status != "draft" && so.Status != "pending_b3_approval" {
		return nil, fiber.NewError(fiber.StatusBadRequest, "Only draft or pending B3 approval SOs can be approved")
	}

	so.Status = "approved"
	if err := s.DB.WithContext(c.Context()).Save(&so).Error; err != nil {
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Failed to approve SO")
	}

	s.DB.WithContext(c.Context()).Preload("Customer").Preload("Items.Product").First(&so, so.ID)

	LogCtxActivity(s.DB, c, "APPROVE", "sales-orders", so.ID.String(), "Approved SO: "+so.SONumber)

	return &so, nil
}
