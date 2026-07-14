package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Warehouse struct {
	ID        uuid.UUID `gorm:"primaryKey;not null" json:"id"`
	Code      string    `gorm:"uniqueIndex;not null;type:varchar(50)" json:"code"`
	Name      string    `gorm:"not null;type:varchar(100)" json:"name"`
	CreatedAt time.Time `gorm:"autoCreateTime:milli" json:"created_at"`
	UpdatedAt time.Time `gorm:"autoCreateTime:milli;autoUpdateTime:milli" json:"updated_at"`
}

func (warehouse *Warehouse) BeforeCreate(_ *gorm.DB) error {
	warehouse.ID = uuid.New()
	return nil
}
