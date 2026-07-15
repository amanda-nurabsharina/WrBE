package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Product struct {
	ID               uuid.UUID     `gorm:"primaryKey;not null" json:"id"`
	Code             string        `gorm:"uniqueIndex;not null;type:varchar(50)" json:"code"`
	Barcode          string        `gorm:"type:varchar(50)" json:"barcode"`
	Name             string        `gorm:"not null;type:varchar(100)" json:"name"`
	CategoryID       string        `gorm:"type:varchar(50)" json:"category_id"`
	Unit             string        `gorm:"type:varchar(20)" json:"unit"`
	MinimumStock     int           `gorm:"default:0" json:"minimum_stock"`
	RegCategory      string        `gorm:"type:varchar(20);default:'non-B3'" json:"reg_category"`
	KementanRegNo    string        `gorm:"type:varchar(100)" json:"kementan_reg_no"`
	MSDSReference    string        `gorm:"type:varchar(255)" json:"msds_reference"`
	SubCategory      string        `gorm:"type:varchar(50)" json:"sub_category"`
	PackagingUnitID  uuid.UUID     `gorm:"type:uuid" json:"packaging_unit_id"`
	PackagingUnit    PackagingUnit `gorm:"foreignKey:PackagingUnitID;references:ID" json:"packaging_unit"`
	ConversionRatio  int           `gorm:"default:1" json:"conversion_ratio"`
	PurchasePrice    float64       `gorm:"type:numeric(15,2);default:0" json:"purchase_price"`
	PriceDistributor float64       `gorm:"type:numeric(15,2);default:0" json:"price_distributor"`
	PriceRetail      float64       `gorm:"type:numeric(15,2);default:0" json:"price_retail"`
	Stock            int           `gorm:"-" json:"stock"`
	CreatedAt        time.Time     `gorm:"autoCreateTime:milli" json:"created_at"`
	UpdatedAt        time.Time     `gorm:"autoCreateTime:milli;autoUpdateTime:milli" json:"updated_at"`
}

func (product *Product) BeforeCreate(_ *gorm.DB) error {
	product.ID = uuid.New()
	return nil
}
