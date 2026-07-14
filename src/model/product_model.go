package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Product struct {
	ID           uuid.UUID `gorm:"primaryKey;not null" json:"id"`
	Code         string    `gorm:"uniqueIndex;not null;type:varchar(50)" json:"code"`
	Barcode      string    `gorm:"type:varchar(50)" json:"barcode"`
	Name         string    `gorm:"not null;type:varchar(100)" json:"name"`
	CategoryID   string    `gorm:"type:varchar(50)" json:"category_id"`
	Unit         string    `gorm:"type:varchar(20)" json:"unit"`
	MinimumStock int       `gorm:"default:0" json:"minimum_stock"`
	CreatedAt    time.Time `gorm:"autoCreateTime:milli" json:"created_at"`
	UpdatedAt    time.Time `gorm:"autoCreateTime:milli;autoUpdateTime:milli" json:"updated_at"`
}

func (product *Product) BeforeCreate(_ *gorm.DB) error {
	product.ID = uuid.New()
	return nil
}
