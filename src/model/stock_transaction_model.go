package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type StockTransaction struct {
	ID              uuid.UUID      `gorm:"primaryKey;not null" json:"id"`
	BatchID         uuid.UUID      `gorm:"not null" json:"batch_id"`
	Batch           InventoryBatch `gorm:"foreignKey:BatchID;references:ID" json:"batch"`
	TransactionType string         `gorm:"not null;type:varchar(20)" json:"transaction_type"` // IN, OUT, TRANSFER, ADJUSTMENT
	Qty             int            `gorm:"not null" json:"qty"`
	ReferenceNo     string         `gorm:"type:varchar(50)" json:"reference_no"`
	UserID          uuid.UUID      `gorm:"not null" json:"user_id"`
	User            User           `gorm:"foreignKey:UserID;references:ID" json:"user"`
	POID            *uuid.UUID     `gorm:"type:uuid" json:"po_id"`
	SOID            *uuid.UUID     `gorm:"type:uuid" json:"so_id"`
	SellingPrice    float64        `gorm:"type:numeric(15,2);default:0" json:"selling_price"`
	CreatedAt       time.Time      `gorm:"autoCreateTime:milli" json:"created_at"`
}

func (tx *StockTransaction) BeforeCreate(_ *gorm.DB) error {
	tx.ID = uuid.New()
	return nil
}
