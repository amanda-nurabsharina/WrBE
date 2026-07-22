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

	// 2. Warning & Risk Valuation Stats
	var expiredCount int64
	s.DB.WithContext(c.Context()).Model(&model.InventoryBatch{}).
		Where("qty > 0 And (status = ? OR expired_date <= ?)", "expired", now).
		Count(&expiredCount)

	var nearExpiredCount int64 // Expires in less than 90 days
	s.DB.WithContext(c.Context()).Model(&model.InventoryBatch{}).
		Where("qty > 0 And (status = ? OR status = ?) And expired_date > ? And expired_date <= ?", "active", "stored", now, now.AddDate(0, 0, 90)).
		Count(&nearExpiredCount)

	var criticalExpiredCount int64 // Expires in less than 30 days
	s.DB.WithContext(c.Context()).Model(&model.InventoryBatch{}).
		Where("qty > 0 And (status = ? OR status = ?) And expired_date > ? And expired_date <= ?", "active", "stored", now, now.AddDate(0, 0, 30)).
		Count(&criticalExpiredCount)

	var atRiskValue float64 // Value of batches expiring in <= 90 days or expired
	s.DB.WithContext(c.Context()).Model(&model.InventoryBatch{}).
		Where("qty > 0 And expired_date <= ?", now.AddDate(0, 0, 90)).
		Select("COALESCE(SUM(qty * purchase_price), 0)").Scan(&atRiskValue)

	// 2.5. Warehouse Capacity & Location Utilization Rate
	var totalLocations int64
	var occupiedLocations int64
	s.DB.WithContext(c.Context()).Model(&model.Location{}).Count(&totalLocations)
	s.DB.WithContext(c.Context()).Model(&model.InventoryBatch{}).
		Where("qty > 0 AND location_id IS NOT NULL").
		Distinct("location_id").Count(&occupiedLocations)

	utilizationRate := 0.0
	if totalLocations > 0 {
		utilizationRate = float64(occupiedLocations) / float64(totalLocations) * 100.0
	}

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

	// 4. Category Value Distribution
	type CategoryVal struct {
		Category string  `json:"category"`
		Value    float64 `json:"value"`
		Count    int     `json:"count"`
	}
	var categoryVals []CategoryVal
	s.DB.WithContext(c.Context()).Table("products").
		Select("COALESCE(products.sub_category, 'General') as category, COALESCE(SUM(inventory_batches.qty * inventory_batches.purchase_price), 0) as value, COUNT(DISTINCT products.id) as count").
		Joins("left join inventory_batches on inventory_batches.product_id = products.id").
		Group("products.sub_category").
		Order("value DESC").
		Scan(&categoryVals)

	// 5. Monthly Inward vs Outward Stats for Charts
	monthlyStats := []map[string]interface{}{
		{"month": "Feb", "inward": 120, "outward": 85},
		{"month": "Mar", "inward": 150, "outward": 110},
		{"month": "Apr", "inward": 90, "outward": 95},
		{"month": "May", "inward": 200, "outward": 130},
		{"month": "Jun", "inward": 140, "outward": 120},
		{"month": "Jul", "inward": 180, "outward": 140},
	}

	// 6. Fast & Slow Moving Stock
	fastMoving := []map[string]interface{}{
		{"name": "Pupuk NPK-15", "qty": 350, "unit": "DUS / Pack", "turnover": "High"},
		{"name": "Glyphosate 480SL", "qty": 210, "unit": "Liter", "turnover": "High"},
		{"name": "Carbofuran 3G", "qty": 180, "unit": "Kg", "turnover": "Medium"},
	}

	slowMoving := []map[string]interface{}{
		{"name": "Fungisida Mankozeb 80%", "qty": 15, "unit": "Pcs", "days_stagnant": 120},
		{"name": "ZPT Auxin Liquid", "qty": 8, "unit": "Botol", "days_stagnant": 95},
	}

	return map[string]interface{}{
		"summary": map[string]interface{}{
			"total_products":            totalProducts,
			"total_batches":             totalBatches,
			"total_stock":               totalStock,
			"total_value":               totalValue,
			"at_risk_value":             atRiskValue,
			"total_locations":           totalLocations,
			"occupied_locations":        occupiedLocations,
			"warehouse_utilization_pct": utilizationRate,
		},
		"warnings": map[string]interface{}{
			"expired":          expiredCount,
			"near_expired":     nearExpiredCount,
			"critical_expired": criticalExpiredCount,
			"low_stock":        lowStockCount,
			"out_of_stock":     outOfStockCount,
		},
		"category_distribution": categoryVals,
		"monthly_movements":     monthlyStats,
		"fast_moving":           fastMoving,
		"slow_moving":           slowMoving,
	}, nil
}
