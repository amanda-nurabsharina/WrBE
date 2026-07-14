package model

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type StringArray []string

func (s StringArray) Value() (driver.Value, error) {
	if s == nil {
		return json.Marshal([]string{})
	}
	return json.Marshal(s)
}

func (s *StringArray) Scan(value interface{}) error {
	if value == nil {
		*s = []string{}
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed")
	}
	return json.Unmarshal(bytes, s)
}

type Role struct {
	ID              uuid.UUID      `gorm:"primaryKey;not null" json:"id"`
	Name            string         `gorm:"uniqueIndex;not null" json:"name"`
	DisplayName     string         `gorm:"not null" json:"display_name"`
	Description     string         `json:"description"`
	AccessibleMenus StringArray    `gorm:"type:jsonb" json:"accessible_menus"`
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
	DeletedAt       gorm.DeletedAt `gorm:"index" json:"-"`
}

func (role *Role) BeforeCreate(_ *gorm.DB) error {
	role.ID = uuid.New()
	return nil
}
