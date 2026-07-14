package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Supplier struct {
	ID        uuid.UUID `gorm:"primaryKey;not null" json:"id"`
	Name      string    `gorm:"not null;type:varchar(100)" json:"name"`
	Phone     string    `gorm:"type:varchar(20)" json:"phone"`
	Email     string    `gorm:"type:varchar(100)" json:"email"`
	CreatedAt time.Time `gorm:"autoCreateTime:milli" json:"created_at"`
	UpdatedAt time.Time `gorm:"autoCreateTime:milli;autoUpdateTime:milli" json:"updated_at"`
}

func (supplier *Supplier) BeforeCreate(_ *gorm.DB) error {
	supplier.ID = uuid.New()
	return nil
}
