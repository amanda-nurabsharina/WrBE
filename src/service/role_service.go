package service

import (
	"app/src/model"
	"app/src/utils"
	"errors"
	"fmt"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type RoleService interface {
	ListRoles(c *fiber.Ctx) ([]model.Role, error)
	GetRoleByID(c *fiber.Ctx, id string) (*model.Role, error)
	GetRoleByName(c *fiber.Ctx, name string) (*model.Role, error)
	CreateRole(c *fiber.Ctx, name, displayName, description string, accessibleMenus []string) (*model.Role, error)
	UpdateRole(c *fiber.Ctx, id string, displayName, description string, accessibleMenus []string) (*model.Role, error)
	DeleteRole(c *fiber.Ctx, id string) error
}

type roleService struct {
	Log      *logrus.Logger
	DB       *gorm.DB
	Validate *validator.Validate
}

func NewRoleService(db *gorm.DB, validate *validator.Validate) RoleService {
	return &roleService{
		Log:      utils.Log,
		DB:       db,
		Validate: validate,
	}
}

func (s *roleService) ListRoles(c *fiber.Ctx) ([]model.Role, error) {
	var roles []model.Role
	if err := s.DB.WithContext(c.Context()).Order("name ASC").Find(&roles).Error; err != nil {
		s.Log.Errorf("Failed to list roles: %v", err)
		return nil, err
	}
	return roles, nil
}

func (s *roleService) GetRoleByID(c *fiber.Ctx, id string) (*model.Role, error) {
	uid, err := uuid.Parse(id)
	if err != nil {
		return nil, errors.New("invalid role ID format")
	}

	var role model.Role
	if err := s.DB.WithContext(c.Context()).First(&role, "id = ?", uid).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("role not found")
		}
		s.Log.Errorf("Failed to get role by ID: %v", err)
		return nil, err
	}
	return &role, nil
}

func (s *roleService) GetRoleByName(c *fiber.Ctx, name string) (*model.Role, error) {
	var role model.Role
	if err := s.DB.WithContext(c.Context()).First(&role, "name = ?", name).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		s.Log.Errorf("Failed to get role by Name: %v", err)
		return nil, err
	}
	return &role, nil
}

func (s *roleService) CreateRole(c *fiber.Ctx, name, displayName, description string, accessibleMenus []string) (*model.Role, error) {
	existing, err := s.GetRoleByName(c, name)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, fmt.Errorf("role with name '%s' already exists", name)
	}

	role := model.Role{
		ID:              uuid.New(),
		Name:            name,
		DisplayName:     displayName,
		Description:     description,
		AccessibleMenus: model.StringArray(accessibleMenus),
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}

	if err := s.DB.WithContext(c.Context()).Create(&role).Error; err != nil {
		s.Log.Errorf("Failed to create role: %v", err)
		return nil, err
	}

	return &role, nil
}

func (s *roleService) UpdateRole(c *fiber.Ctx, id string, displayName, description string, accessibleMenus []string) (*model.Role, error) {
	role, err := s.GetRoleByID(c, id)
	if err != nil {
		return nil, err
	}

	role.DisplayName = displayName
	role.Description = description
	role.AccessibleMenus = model.StringArray(accessibleMenus)
	role.UpdatedAt = time.Now()

	if err := s.DB.WithContext(c.Context()).Save(role).Error; err != nil {
		s.Log.Errorf("Failed to update role: %v", err)
		return nil, err
	}

	return role, nil
}

func (s *roleService) DeleteRole(c *fiber.Ctx, id string) error {
	role, err := s.GetRoleByID(c, id)
	if err != nil {
		return err
	}

	// Prevent deleting system critical roles
	if role.Name == "super_admin" || role.Name == "admin" || role.Name == "user" || role.Name == "employee" {
		return errors.New("system roles cannot be deleted")
	}

	if err := s.DB.WithContext(c.Context()).Delete(&model.Role{}, "id = ?", role.ID).Error; err != nil {
		s.Log.Errorf("Failed to delete role: %v", err)
		return err
	}

	return nil
}
