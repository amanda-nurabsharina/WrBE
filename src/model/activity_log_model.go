package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ActivityLog struct {
	ID          uuid.UUID `gorm:"primaryKey;not null" json:"id"`
	UserID      uuid.UUID `gorm:"not null;index" json:"user_id"`
	User        User      `gorm:"foreignKey:UserID;references:ID" json:"user"`
	Action      string    `gorm:"not null;type:varchar(20);index" json:"action"`  // CREATE, UPDATE, DELETE, APPROVE, LOGIN, OPNAME
	Module      string    `gorm:"not null;type:varchar(50);index" json:"module"`  // products, suppliers, purchase-orders, etc.
	TargetID    string    `gorm:"type:varchar(50)" json:"target_id"`              // ID of the affected record
	Description string    `gorm:"type:text" json:"description"`                   // Human-readable detail
	IPAddress   string    `gorm:"type:varchar(45)" json:"ip_address"`             // Client IP
	CreatedAt   time.Time `gorm:"autoCreateTime:milli;index" json:"created_at"`
}

func (a *ActivityLog) BeforeCreate(_ *gorm.DB) error {
	a.ID = uuid.New()
	return nil
}
