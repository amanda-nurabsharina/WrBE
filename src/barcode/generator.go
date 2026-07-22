package barcode

import (
	"fmt"
	"strconv"
	"strings"
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
	if prefix == "" {
		prefix = "BAT"
	}

	// 1. Query highest existing barcode for this prefix from BarcodeRegistry
	var maxBarcode string
	db.Model(&model.BarcodeRegistry{}).
		Where("barcode LIKE ?", prefix+"%").
		Order("barcode DESC").
		Limit(1).
		Pluck("barcode", &maxBarcode)

	// 2. Also check InventoryBatch table if prefix is BAT
	if prefix == "BAT" {
		var maxBatchBarcode string
		db.Model(&model.InventoryBatch{}).
			Where("barcode LIKE ?", prefix+"%").
			Order("barcode DESC").
			Limit(1).
			Pluck("barcode", &maxBatchBarcode)

		if maxBatchBarcode > maxBarcode {
			maxBarcode = maxBatchBarcode
		}
	}

	var nextNum int64 = 1
	if maxBarcode != "" {
		digits := strings.TrimPrefix(maxBarcode, prefix)
		if val, err := strconv.ParseInt(digits, 10, 64); err == nil {
			nextNum = val + 1
		}
	}

	// 3. Loop until we find a guaranteed unique barcode
	for i := 0; i < 1000; i++ {
		candidate := fmt.Sprintf("%s%08d", prefix, nextNum+int64(i))

		var countReg, countBatch int64
		db.Model(&model.BarcodeRegistry{}).Where("barcode = ?", candidate).Count(&countReg)
		db.Model(&model.InventoryBatch{}).Where("barcode = ?", candidate).Count(&countBatch)

		if countReg == 0 && countBatch == 0 {
			// Advance DB sequence if postgres sequence exists
			var seqName string
			switch prefix {
			case "PRD":
				seqName = "barcode_product_seq"
			case "BAT":
				seqName = "barcode_batch_seq"
			case "LOC":
				seqName = "barcode_location_seq"
			}
			if seqName != "" {
				_ = db.Exec(fmt.Sprintf("SELECT setval('%s', %d, true)", seqName, nextNum+int64(i)))
			}
			return candidate, nil
		}
	}

	// Fallback timestamp if 1000 candidates collided
	return fmt.Sprintf("%s%d", prefix, time.Now().UnixNano()/100000), nil
}
