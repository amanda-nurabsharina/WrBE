package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type InventoryBatch struct {
	ID            uuid.UUID  `gorm:"primaryKey;not null" json:"id"`
	ProductID     uuid.UUID  `gorm:"not null" json:"product_id"`
	Product       Product    `gorm:"foreignKey:ProductID;references:ID" json:"product"`
	BatchNumber   string     `gorm:"not null;type:varchar(50)" json:"batch_number"`
	Barcode       string     `gorm:"type:varchar(100);uniqueIndex" json:"barcode"`
	ExpiredDate   time.Time  `gorm:"not null;type:date" json:"expired_date"`
	Qty           int        `gorm:"not null;default:0" json:"qty"` // Physical quantity
	AllocatedQty  int        `gorm:"not null;default:0" json:"allocated_qty"` // Allocated for picking
	AvailableQty  int        `gorm:"not null;default:0" json:"available_qty"` // Available for SO booking (Qty - AllocatedQty)
	WarehouseID   uuid.UUID  `gorm:"not null" json:"warehouse_id"`
	Warehouse     Warehouse  `gorm:"foreignKey:WarehouseID;references:ID" json:"warehouse"`
	LocationID    uuid.UUID  `gorm:"not null" json:"location_id"`
	Location      Location   `gorm:"foreignKey:LocationID;references:ID" json:"location"`
	PurchasePrice float64    `gorm:"not null;default:0" json:"purchase_price"`
	Status        string     `gorm:"not null;default:waiting_put_away;type:varchar(20)" json:"status"` // waiting_put_away, stored, picked, allocated, expired, quarantine, damaged
	POID          *uuid.UUID `gorm:"type:uuid" json:"po_id"`
	ReceivedAt    time.Time  `json:"received_at"`
	CreatedAt     time.Time  `gorm:"autoCreateTime:milli" json:"created_at"`
	UpdatedAt     time.Time  `gorm:"autoCreateTime:milli;autoUpdateTime:milli" json:"updated_at"`
}

func (batch *InventoryBatch) BeforeCreate(_ *gorm.DB) error {
	batch.ID = uuid.New()
	return nil
}
