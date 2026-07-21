package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Location struct {
	ID          uuid.UUID `gorm:"primaryKey;not null" json:"id"`
	WarehouseID uuid.UUID `gorm:"not null" json:"warehouse_id"`
	Aisle       string    `gorm:"type:varchar(50);default:''" json:"aisle"`
	Rack        string    `gorm:"not null;type:varchar(50)" json:"rack"`
	Shelf       string    `gorm:"type:varchar(50);default:''" json:"shelf"`
	Bin         string    `gorm:"type:varchar(50);default:''" json:"bin"`
	MaxWeight   float64   `gorm:"default:0" json:"max_weight"`
	MaxVolume   float64   `gorm:"default:0" json:"max_volume"`
	Barcode     string    `gorm:"type:varchar(50);uniqueIndex" json:"barcode"`
	CreatedAt   time.Time `gorm:"autoCreateTime:milli" json:"created_at"`
	UpdatedAt   time.Time `gorm:"autoCreateTime:milli;autoUpdateTime:milli" json:"updated_at"`
}

func (location *Location) BeforeCreate(_ *gorm.DB) error {
	location.ID = uuid.New()
	return nil
}
