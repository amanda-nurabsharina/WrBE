package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Location struct {
	ID          uuid.UUID `gorm:"primaryKey;not null" json:"id"`
	WarehouseID uuid.UUID `gorm:"not null" json:"warehouse_id"`
	Rack        string    `gorm:"not null;type:varchar(50)" json:"rack"`
	CreatedAt   time.Time `gorm:"autoCreateTime:milli" json:"created_at"`
	UpdatedAt   time.Time `gorm:"autoCreateTime:milli;autoUpdateTime:milli" json:"updated_at"`
}

func (location *Location) BeforeCreate(_ *gorm.DB) error {
	location.ID = uuid.New()
	return nil
}
