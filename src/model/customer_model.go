package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Customer struct {
	ID          uuid.UUID `gorm:"primaryKey;not null" json:"id"`
	Name        string    `gorm:"not null;type:varchar(100)" json:"name"`
	Phone       string    `gorm:"type:varchar(20)" json:"phone"`
	Email       string    `gorm:"type:varchar(100)" json:"email"`
	PIC         string    `gorm:"type:varchar(100)" json:"pic"`
	Address     string    `gorm:"type:text" json:"address"`
	NPWP        string    `gorm:"type:varchar(50)" json:"npwp"`
	PaymentTerm int       `gorm:"type:integer;default:0" json:"payment_term"` // termin pembayaran in days
	PriceTier   string    `gorm:"type:varchar(50);default:'distributor'" json:"price_tier"` // 'distributor' or 'retail'
	CreatedAt   time.Time `gorm:"autoCreateTime:milli" json:"created_at"`
	UpdatedAt   time.Time `gorm:"autoCreateTime:milli;autoUpdateTime:milli" json:"updated_at"`
}

func (customer *Customer) BeforeCreate(_ *gorm.DB) error {
	customer.ID = uuid.New()
	return nil
}
