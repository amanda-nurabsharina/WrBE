package barcode

import (
	"fmt"
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

	var nextVal int64
	query := fmt.Sprintf("SELECT nextval('%s')", seqName)
	if err := db.Raw(query).Scan(&nextVal).Error; err != nil {
		return "", err
	}

	return fmt.Sprintf("%s%08d", prefix, nextVal), nil
}
