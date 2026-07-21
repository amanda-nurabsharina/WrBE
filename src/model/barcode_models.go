package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type BarcodeRegistry struct {
	ID          uuid.UUID `gorm:"primaryKey;not null" json:"id"`
	Barcode     string    `gorm:"uniqueIndex;not null;type:varchar(100)" json:"barcode"`
	Type        string    `gorm:"not null;type:varchar(20)" json:"type"` // "PRODUCT", "BATCH", "LOCATION"
	ReferenceID uuid.UUID `gorm:"not null;type:uuid" json:"reference_id"`
	CreatedAt   time.Time `gorm:"autoCreateTime" json:"created_at"`
}

func (r *BarcodeRegistry) BeforeCreate(_ *gorm.DB) error {
	r.ID = uuid.New()
	return nil
}

type InventoryMovement struct {
	ID             uuid.UUID      `gorm:"primaryKey;not null" json:"id"`
	BatchID        uuid.UUID      `gorm:"not null;type:uuid" json:"batch_id"`
	Batch          InventoryBatch `gorm:"foreignKey:BatchID" json:"-"`
	FromLocationID *uuid.UUID     `gorm:"type:uuid" json:"from_location_id"`
	ToLocationID   *uuid.UUID     `gorm:"type:uuid" json:"to_location_id"`
	Qty            int            `gorm:"not null" json:"qty"`
	MovementType   string         `gorm:"not null;type:varchar(50)" json:"movement_type"` // "RECEIVING", "PUT_AWAY", "RELOCATION", "PICKING", "SHIPPED", "DAMAGE_ADJUSTMENT"
	CreatedBy      uuid.UUID      `gorm:"not null;type:uuid" json:"created_by"`
	CreatedAt      time.Time      `gorm:"autoCreateTime" json:"created_at"`
}

func (m *InventoryMovement) BeforeCreate(_ *gorm.DB) error {
	m.ID = uuid.New()
	return nil
}

type ScanSession struct {
	ID          uuid.UUID `gorm:"primaryKey;not null" json:"id"`
	SessionType string    `gorm:"not null;type:varchar(50)" json:"session_type"` // "PUT_AWAY", "PICKING", "STOCK_OPNAME"
	UserID      uuid.UUID `gorm:"not null;type:uuid" json:"user_id"`
	Status      string    `gorm:"not null;type:varchar(20)" json:"status"`      // "IN_PROGRESS", "COMPLETED", "ABANDONED"
	CreatedAt   time.Time `gorm:"autoCreateTime" json:"created_at"`
}

func (s *ScanSession) BeforeCreate(_ *gorm.DB) error {
	s.ID = uuid.New()
	return nil
}

type ScanLog struct {
	ID        uuid.UUID  `gorm:"primaryKey;not null" json:"id"`
	SessionID *uuid.UUID `gorm:"type:uuid" json:"session_id"`
	Barcode   string     `gorm:"not null;type:varchar(100)" json:"barcode"`
	UserID    uuid.UUID  `gorm:"not null;type:uuid" json:"user_id"`
	Action    string     `gorm:"not null;type:varchar(50)" json:"action"`    // "RECEIVING", "PUT_AWAY", "PICKING", "TRANSFER", "OPNAME"
	Status    string     `gorm:"not null;type:varchar(20)" json:"status"`    // "SUCCESS", "FAILED"
	Message   string     `json:"message"`                                    // e.g. "Wrong Location" or "Success"
	IP        string     `gorm:"type:varchar(50)" json:"ip"`
	Device    string     `gorm:"type:varchar(100)" json:"device"`
	CreatedAt time.Time  `gorm:"autoCreateTime" json:"created_at"`
}

func (l *ScanLog) BeforeCreate(_ *gorm.DB) error {
	l.ID = uuid.New()
	return nil
}

type BarcodePrintHistory struct {
	ID        uuid.UUID `gorm:"primaryKey;not null" json:"id"`
	Barcode   string    `gorm:"not null;type:varchar(100)" json:"barcode"`
	LabelType string    `gorm:"not null;type:varchar(20)" json:"label_type"` // "PRODUCT", "BATCH"
	UserID    uuid.UUID `gorm:"not null;type:uuid" json:"user_id"`
	Qty       int       `gorm:"not null;default:1" json:"qty"`
	Reason    string    `gorm:"type:varchar(100)" json:"reason"` // "Initial Print", "Reprint", "Replacement"
	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
}

func (h *BarcodePrintHistory) BeforeCreate(_ *gorm.DB) error {
	h.ID = uuid.New()
	return nil
}
