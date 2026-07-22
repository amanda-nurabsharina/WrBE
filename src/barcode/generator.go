package barcode

import (
	"fmt"
	"time"

	"app/src/model"
	"gorm.io/gorm"
)

type Generator interface {
	Generate(db *gorm.DB, prefix string) (string, error)
}

type generator struct{}

func NewGenerator() Generator {
	return &generator{}
}

func (g *generator) Generate(db *gorm.DB, prefix string) (string, error) {
	var seqName string
	switch prefix {
	case "PRD":
		seqName = "barcode_product_seq"
	case "BAT":
		seqName = "barcode_batch_seq"
	case "LOC":
		seqName = "barcode_location_seq"
	default:
		return "", fmt.Errorf("invalid prefix: %s", prefix)
	}

	for i := 0; i < 100; i++ {
		var nextVal int64
		query := fmt.Sprintf("SELECT nextval('%s')", seqName)
		if err := db.Raw(query).Scan(&nextVal).Error; err != nil {
			var count int64
			db.Model(&model.BarcodeRegistry{}).Where("barcode LIKE ?", prefix+"%").Count(&count)
			nextVal = count + 1 + int64(i)
		}

		barcodeStr := fmt.Sprintf("%s%08d", prefix, nextVal)

		// Verify barcode does not already exist in BarcodeRegistry
		var existingCount int64
		db.Model(&model.BarcodeRegistry{}).Where("barcode = ?", barcodeStr).Count(&existingCount)
		if existingCount == 0 {
			return barcodeStr, nil
		}
	}

	// Ultimate fallback with timestamp if sequence is out of sync
	return fmt.Sprintf("%s%d", prefix, time.Now().UnixNano()/100000), nil
}
