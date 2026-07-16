package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type PurchaseOrder struct {
	ID         uuid.UUID           `gorm:"primaryKey;not null" json:"id"`
	PONumber   string              `gorm:"uniqueIndex;not null;type:varchar(50)" json:"po_number"`
	SupplierID uuid.UUID           `gorm:"not null" json:"supplier_id"`
	Supplier   Supplier            `gorm:"foreignKey:SupplierID;references:ID" json:"supplier"`
	OrderDate  time.Time           `gorm:"not null;type:date" json:"order_date"`
	Status     string              `gorm:"not null;default:draft;type:varchar(30)" json:"status"` // draft, approved, partially_received, completed
	Items      []PurchaseOrderItem `gorm:"foreignKey:POID;constraint:OnDelete:CASCADE" json:"items"`
	CreatedAt  time.Time           `gorm:"autoCreateTime:milli" json:"created_at"`
	UpdatedAt  time.Time           `gorm:"autoCreateTime:milli;autoUpdateTime:milli" json:"updated_at"`
}

type PurchaseOrderItem struct {
	ID          uuid.UUID `gorm:"primaryKey;not null" json:"id"`
	POID        uuid.UUID `gorm:"not null" json:"po_id"`
	ProductID   uuid.UUID `gorm:"not null" json:"product_id"`
	Product     Product   `gorm:"foreignKey:ProductID;references:ID" json:"product"`
	Qty         int       `gorm:"not null;default:0" json:"qty"`
	ReceivedQty int       `gorm:"not null;default:0" json:"received_qty"`
	Price       float64   `gorm:"type:numeric(15,2);default:0" json:"price"`
	CreatedAt   time.Time `gorm:"autoCreateTime:milli" json:"created_at"`
	UpdatedAt   time.Time `gorm:"autoCreateTime:milli;autoUpdateTime:milli" json:"updated_at"`
}

func (po *PurchaseOrder) BeforeCreate(_ *gorm.DB) error {
	po.ID = uuid.New()
	return nil
}

func (item *PurchaseOrderItem) BeforeCreate(_ *gorm.DB) error {
	item.ID = uuid.New()
	return nil
}
