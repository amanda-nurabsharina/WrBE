package service

import (
	"app/src/model"
	"app/src/utils"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type DashboardService interface {
	GetDashboardData(c *fiber.Ctx) (map[string]interface{}, error)
}

type dashboardService struct {
	Log *logrus.Logger
	DB  *gorm.DB
}

func NewDashboardService(db *gorm.DB) DashboardService {
	return &dashboardService{
		Log: utils.Log,
		DB:  db,
	}
}

type ProductStockSum struct {
	ProductID    string
	Code         string
	Name         string
	TotalStock   int
	MinimumStock int
}

func (s *dashboardService) GetDashboardData(c *fiber.Ctx) (map[string]interface{}, error) {
	var totalProducts int64
	var totalBatches int64
	var totalStock int64
	var totalValue float64

	now := time.Now()

	// 1. Basic Stats
	s.DB.WithContext(c.Context()).Model(&model.Product{}).Count(&totalProducts)
	s.DB.WithContext(c.Context()).Model(&model.InventoryBatch{}).Count(&totalBatches)
	s.DB.WithContext(c.Context()).Model(&model.InventoryBatch{}).Select("COALESCE(SUM(qty), 0)").Scan(&totalStock)
	s.DB.WithContext(c.Context()).Model(&model.InventoryBatch{}).Select("COALESCE(SUM(qty * purchase_price), 0)").Scan(&totalValue)

	// 2. Warning Stats
	var expiredCount int64
	s.DB.WithContext(c.Context()).Model(&model.InventoryBatch{}).
		Where("qty > 0 And (status = ? OR expired_date <= ?)", "expired", now).
		Count(&expiredCount)

	var nearExpiredCount int64 // Expires in less than 90 days
	s.DB.WithContext(c.Context()).Model(&model.InventoryBatch{}).
		Where("qty > 0 And status = ? And expired_date > ? And expired_date <= ?", "active", now, now.AddDate(0, 0, 90)).
		Count(&nearExpiredCount)

	// 3. Low Stock & Out of Stock Calculations
	var lowStockCount int64
	var outOfStockCount int64

	var stockSums []ProductStockSum
	s.DB.WithContext(c.Context()).Table("products").
		Select("products.id as product_id, products.code, products.name, COALESCE(SUM(inventory_batches.qty), 0) as total_stock, products.minimum_stock").
		Joins("left join inventory_batches on inventory_batches.product_id = products.id").
		Group("products.id, products.code, products.name, products.minimum_stock").
		Scan(&stockSums)

	for _, item := range stockSums {
		if item.TotalStock == 0 {
			outOfStockCount++
		} else if item.TotalStock < item.MinimumStock {
			lowStockCount++
		}
	}

	// 4. Monthly Inward vs Outward Stats for Charts
	type MonthlyMovement struct {
		Month string
		In    int
		Out   int
	}

	// For a quick POC, compile mock/live monthly statistics from the last 6 months
	monthlyStats := []map[string]interface{}{
		{"month": "Feb", "inward": 120, "outward": 85},
		{"month": "Mar", "inward": 150, "outward": 110},
		{"month": "Apr", "inward": 90, "outward": 95},
		{"month": "May", "inward": 200, "outward": 130},
		{"month": "Jun", "inward": 140, "outward": 120},
		{"month": "Jul", "inward": 180, "outward": 140},
	}

	// 5. Fast/Slow Moving Mock data for POC UI
	fastMoving := []map[string]interface{}{
		{"name": "Paracetamol 500mg", "qty": 350, "unit": "Box"},
		{"name": "Amoxicillin 250mg", "qty": 210, "unit": "Box"},
	}

	slowMoving := []map[string]interface{}{
		{"name": "Vitamin C 1000mg", "qty": 15, "unit": "Bottle"},
	}

	return map[string]interface{}{
		"summary": map[string]interface{}{
			"total_products":  totalProducts,
			"total_batches":   totalBatches,
			"total_stock":     totalStock,
			"total_value":     totalValue,
		},
		"warnings": map[string]interface{}{
			"expired":      expiredCount,
			"near_expired": nearExpiredCount,
			"low_stock":    lowStockCount,
			"out_of_stock": outOfStockCount,
		},
		"monthly_movements": monthlyStats,
		"fast_moving":       fastMoving,
		"slow_moving":       slowMoving,
	}, nil
}
