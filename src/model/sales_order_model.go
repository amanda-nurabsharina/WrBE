package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type SalesOrder struct {
	ID            uuid.UUID        `gorm:"primaryKey;not null" json:"id"`
	SONumber      string           `gorm:"uniqueIndex;not null;type:varchar(50)" json:"so_number"`
	CustomerID    uuid.UUID        `gorm:"not null" json:"customer_id"`
	Customer      Customer         `gorm:"foreignKey:CustomerID;references:ID" json:"customer"`
	OrderDate     time.Time        `gorm:"not null;type:date" json:"order_date"`
	Status        string           `gorm:"not null;default:draft;type:varchar(30)" json:"status"` // draft, pending_b3_approval, approved, partially_shipped, shipped
	PaymentStatus string           `gorm:"not null;default:unpaid;type:varchar(30)" json:"payment_status"` // unpaid, partially_paid, paid
	Items         []SalesOrderItem `gorm:"foreignKey:SOID;constraint:OnDelete:CASCADE" json:"items"`
	CreatedAt     time.Time        `gorm:"autoCreateTime:milli" json:"created_at"`
	UpdatedAt     time.Time        `gorm:"autoCreateTime:milli;autoUpdateTime:milli" json:"updated_at"`
}

type SalesOrderItem struct {
	ID        uuid.UUID `gorm:"primaryKey;not null" json:"id"`
	SOID      uuid.UUID `gorm:"not null" json:"so_id"`
	ProductID uuid.UUID `gorm:"not null" json:"product_id"`
	Product   Product   `gorm:"foreignKey:ProductID;references:ID" json:"product"`
	Qty       int       `gorm:"not null;default:0" json:"qty"`
	ShippedQty int       `gorm:"not null;default:0" json:"shipped_qty"`
	Price     float64   `gorm:"type:numeric(15,2);default:0" json:"price"`
	CreatedAt time.Time `gorm:"autoCreateTime:milli" json:"created_at"`
	UpdatedAt time.Time `gorm:"autoCreateTime:milli;autoUpdateTime:milli" json:"updated_at"`
}

func (so *SalesOrder) BeforeCreate(_ *gorm.DB) error {
	so.ID = uuid.New()
	return nil
}

func (item *SalesOrderItem) BeforeCreate(_ *gorm.DB) error {
	item.ID = uuid.New()
	return nil
}
