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

type CustomerService interface {
	GetCustomers(c *fiber.Ctx, search string) ([]model.Customer, error)
	GetCustomerByID(c *fiber.Ctx, id string) (*model.Customer, error)
	CreateCustomer(c *fiber.Ctx, req *validation.CreateCustomer) (*model.Customer, error)
	UpdateCustomer(c *fiber.Ctx, id string, req *validation.UpdateCustomer) (*model.Customer, error)
	DeleteCustomer(c *fiber.Ctx, id string) error
}

type customerService struct {
	Log      *logrus.Logger
	DB       *gorm.DB
	Validate *validator.Validate
}

func NewCustomerService(db *gorm.DB, validate *validator.Validate) CustomerService {
	return &customerService{
		Log:      utils.Log,
		DB:       db,
		Validate: validate,
	}
}

func (s *customerService) GetCustomers(c *fiber.Ctx, search string) ([]model.Customer, error) {
	var list []model.Customer
	query := s.DB.WithContext(c.Context()).Order("name asc")

	if search != "" {
		query = query.Where("name LIKE ? OR phone LIKE ? OR email LIKE ? OR pic LIKE ?", "%"+search+"%", "%"+search+"%", "%"+search+"%", "%"+search+"%")
	}

	if err := query.Find(&list).Error; err != nil {
		s.Log.Errorf("Failed to query customers: %v", err)
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Database error")
	}

	return list, nil
}

func (s *customerService) GetCustomerByID(c *fiber.Ctx, id string) (*model.Customer, error) {
	var customer model.Customer
	uid, err := uuid.Parse(id)
	if err != nil {
		return nil, fiber.NewError(fiber.StatusBadRequest, "Invalid UUID format")
	}

	if err := s.DB.WithContext(c.Context()).First(&customer, "id = ?", uid).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fiber.NewError(fiber.StatusNotFound, "Customer not found")
		}
		s.Log.Errorf("Failed to query customer by ID: %v", err)
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Database error")
	}

	return &customer, nil
}

func (s *customerService) CreateCustomer(c *fiber.Ctx, req *validation.CreateCustomer) (*model.Customer, error) {
	if err := s.Validate.Struct(req); err != nil {
		return nil, err
	}

	customer := model.Customer{
		Name:        req.Name,
		Phone:       req.Phone,
		Email:       req.Email,
		PIC:         req.PIC,
		Address:     req.Address,
		NPWP:        req.NPWP,
		PaymentTerm: req.PaymentTerm,
		PriceTier:   req.PriceTier,
	}

	if err := s.DB.WithContext(c.Context()).Create(&customer).Error; err != nil {
		s.Log.Errorf("Failed to create customer: %v", err)
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Database error")
	}

	LogCtxActivity(s.DB, c, "CREATE", "customers", customer.ID.String(), "Created customer: "+customer.Name)

	return &customer, nil
}

func (s *customerService) UpdateCustomer(c *fiber.Ctx, id string, req *validation.UpdateCustomer) (*model.Customer, error) {
	if err := s.Validate.Struct(req); err != nil {
		return nil, err
	}

	customer, err := s.GetCustomerByID(c, id)
	if err != nil {
		return nil, err
	}

	if req.Name != "" {
		customer.Name = req.Name
	}
	customer.Phone = req.Phone
	customer.Email = req.Email
	customer.PIC = req.PIC
	customer.Address = req.Address
	customer.NPWP = req.NPWP
	if req.PaymentTerm != nil {
		customer.PaymentTerm = *req.PaymentTerm
	}
	if req.PriceTier != "" {
		customer.PriceTier = req.PriceTier
	}

	if err := s.DB.WithContext(c.Context()).Save(customer).Error; err != nil {
		s.Log.Errorf("Failed to update customer: %v", err)
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Database error")
	}

	LogCtxActivity(s.DB, c, "UPDATE", "customers", customer.ID.String(), "Updated customer: "+customer.Name)

	return customer, nil
}

func (s *customerService) DeleteCustomer(c *fiber.Ctx, id string) error {
	customer, err := s.GetCustomerByID(c, id)
	if err != nil {
		return err
	}

	if err := s.DB.WithContext(c.Context()).Delete(customer).Error; err != nil {
		s.Log.Errorf("Failed to delete customer: %v", err)
		return fiber.NewError(fiber.StatusInternalServerError, "Database error")
	}

	LogCtxActivity(s.DB, c, "DELETE", "customers", customer.ID.String(), "Deleted customer: "+customer.Name)

	return nil
}
