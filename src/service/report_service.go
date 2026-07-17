package service

import (
	"app/src/model"
	"app/src/utils"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type ReportService interface {
	GetInventoryValueReport(c *fiber.Ctx, categoryID, warehouseID string) (interface{}, error)
	GetStockAgingReport(c *fiber.Ctx, categoryID, warehouseID, productID string) (interface{}, error)
	GetStockMutationReport(c *fiber.Ctx, startDate, endDate, productID, warehouseID string) (interface{}, error)
	GetDistributionReport(c *fiber.Ctx, categoryID, subCategory, startDate, endDate string) (interface{}, error)
	GetReorderPointReport(c *fiber.Ctx, leadTime int) (interface{}, error)
}

type reportService struct {
	Log *logrus.Logger
	DB  *gorm.DB
}

func NewReportService(db *gorm.DB) ReportService {
	return &reportService{
		Log: utils.Log,
		DB:  db,
	}
}

type InventoryValueItem struct {
	BatchID           uuid.UUID `json:"batch_id"`
	BatchNumber       string    `json:"batch_number"`
	ProductID         uuid.UUID `json:"product_id"`
	ProductCode       string    `json:"product_code"`
	ProductName       string    `json:"product_name"`
	Category          string    `json:"category"`
	SubCategory       string    `json:"sub_category"`
	Qty               int       `json:"qty"`
	Unit              string    `json:"unit"`
	PackagingUnitName string    `json:"packaging_unit_name"`
	WarehouseCode     string    `json:"warehouse_code"`
	WarehouseName     string    `json:"warehouse_name"`
	LocationRack      string    `json:"location_rack"`
	PurchasePrice     float64   `json:"purchase_price"`
	TotalValue        float64   `json:"total_value"`
}

type InventoryValueReport struct {
	Items      []InventoryValueItem `json:"items"`
	TotalQty   int                  `json:"total_qty"`
	TotalValue float64              `json:"total_value"`
}

func (s *reportService) GetInventoryValueReport(c *fiber.Ctx, categoryID, warehouseID string) (interface{}, error) {
	var batches []model.InventoryBatch

	query := s.DB.WithContext(c.Context()).
		Preload("Product").
		Preload("Product.PackagingUnit").
		Preload("Warehouse").
		Preload("Location").
		Where("qty > 0")

	if categoryID != "" {
		query = query.Joins("JOIN products ON products.id = inventory_batches.product_id").
			Where("products.category_id = ?", categoryID)
	}

	if warehouseID != "" {
		query = query.Where("inventory_batches.warehouse_id = ?", warehouseID)
	}

	if err := query.Find(&batches).Error; err != nil {
		s.Log.Errorf("Failed to query batches for inventory value report: %v", err)
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Database error")
	}

	items := make([]InventoryValueItem, 0, len(batches))
	var totalQty int
	var totalValue float64

	for _, b := range batches {
		val := float64(b.Qty) * b.PurchasePrice
		totalQty += b.Qty
		totalValue += val

		packName := ""
		if b.Product.PackagingUnit.ID != uuid.Nil {
			packName = b.Product.PackagingUnit.Name
		}

		items = append(items, InventoryValueItem{
			BatchID:           b.ID,
			BatchNumber:       b.BatchNumber,
			ProductID:         b.ProductID,
			ProductCode:       b.Product.Code,
			ProductName:       b.Product.Name,
			Category:          b.Product.CategoryID,
			SubCategory:       b.Product.SubCategory,
			Qty:               b.Qty,
			Unit:              b.Product.Unit,
			PackagingUnitName: packName,
			WarehouseCode:     b.Warehouse.Code,
			WarehouseName:     b.Warehouse.Name,
			LocationRack:      b.Location.Rack,
			PurchasePrice:     b.PurchasePrice,
			TotalValue:        val,
		})
	}

	return InventoryValueReport{
		Items:      items,
		TotalQty:   totalQty,
		TotalValue: totalValue,
	}, nil
}

type StockAgingItem struct {
	BatchID           uuid.UUID `json:"batch_id"`
	BatchNumber       string    `json:"batch_number"`
	ProductID         uuid.UUID `json:"product_id"`
	ProductCode       string    `json:"product_code"`
	ProductName       string    `json:"product_name"`
	Category          string    `json:"category"`
	Qty               int       `json:"qty"`
	Unit              string    `json:"unit"`
	WarehouseName     string    `json:"warehouse_name"`
	PurchasePrice     float64   `json:"purchase_price"`
	TotalValue        float64   `json:"total_value"`
	CreatedAt         time.Time `json:"created_at"`
	AgeDays           int       `json:"age_days"`
	Bucket            string    `json:"bucket"` // "0-30", "31-60", "61-90", ">90"
}

type StockAgingBucketSummary struct {
	Count      int     `json:"count"`
	TotalQty   int     `json:"total_qty"`
	TotalValue float64 `json:"total_value"`
}

type StockAgingSummary struct {
	Bucket0_30  StockAgingBucketSummary `json:"bucket_0_30"`
	Bucket31_60 StockAgingBucketSummary `json:"bucket_31_60"`
	Bucket61_90 StockAgingBucketSummary `json:"bucket_61_90"`
	BucketOver90 StockAgingBucketSummary `json:"bucket_over_90"`
}

type StockAgingReport struct {
	Items   []StockAgingItem  `json:"items"`
	Summary StockAgingSummary `json:"summary"`
}

func (s *reportService) GetStockAgingReport(c *fiber.Ctx, categoryID, warehouseID, productID string) (interface{}, error) {
	var batches []model.InventoryBatch

	query := s.DB.WithContext(c.Context()).
		Preload("Product").
		Preload("Warehouse").
		Where("qty > 0")

	if categoryID != "" {
		query = query.Joins("JOIN products ON products.id = inventory_batches.product_id").
			Where("products.category_id = ?", categoryID)
	}

	if warehouseID != "" {
		query = query.Where("inventory_batches.warehouse_id = ?", warehouseID)
	}

	if productID != "" {
		query = query.Where("inventory_batches.product_id = ?", productID)
	}

	if err := query.Find(&batches).Error; err != nil {
		s.Log.Errorf("Failed to query batches for stock aging report: %v", err)
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Database error")
	}

	items := make([]StockAgingItem, 0, len(batches))
	var summary StockAgingSummary

	for _, b := range batches {
		ageDays := int(time.Since(b.CreatedAt).Hours() / 24)
		if ageDays < 0 {
			ageDays = 0
		}
		val := float64(b.Qty) * b.PurchasePrice

		bucket := ""
		if ageDays <= 30 {
			bucket = "0-30"
			summary.Bucket0_30.Count++
			summary.Bucket0_30.TotalQty += b.Qty
			summary.Bucket0_30.TotalValue += val
		} else if ageDays <= 60 {
			bucket = "31-60"
			summary.Bucket31_60.Count++
			summary.Bucket31_60.TotalQty += b.Qty
			summary.Bucket31_60.TotalValue += val
		} else if ageDays <= 90 {
			bucket = "61-90"
			summary.Bucket61_90.Count++
			summary.Bucket61_90.TotalQty += b.Qty
			summary.Bucket61_90.TotalValue += val
		} else {
			bucket = ">90"
			summary.BucketOver90.Count++
			summary.BucketOver90.TotalQty += b.Qty
			summary.BucketOver90.TotalValue += val
		}

		items = append(items, StockAgingItem{
			BatchID:       b.ID,
			BatchNumber:   b.BatchNumber,
			ProductID:     b.ProductID,
			ProductCode:   b.Product.Code,
			ProductName:   b.Product.Name,
			Category:      b.Product.CategoryID,
			Qty:           b.Qty,
			Unit:          b.Product.Unit,
			WarehouseName: b.Warehouse.Name,
			PurchasePrice: b.PurchasePrice,
			TotalValue:    val,
			CreatedAt:     b.CreatedAt,
			AgeDays:       ageDays,
			Bucket:        bucket,
		})
	}

	return StockAgingReport{
		Items:   items,
		Summary: summary,
	}, nil
}

type StockMutationItem struct {
	ProductID        uuid.UUID `json:"product_id"`
	ProductCode      string    `json:"product_code"`
	ProductName      string    `json:"product_name"`
	Category         string    `json:"category"`
	Unit             string    `json:"unit"`
	BeginningBalance int64     `json:"beginning_balance"`
	InQty            int64     `json:"in_qty"`
	OutQty           int64     `json:"out_qty"`
	EndingBalance    int64     `json:"ending_balance"`
}

func (s *reportService) GetStockMutationReport(c *fiber.Ctx, startDate, endDate, productID, warehouseID string) (interface{}, error) {
	var startDateTime, endDateTime time.Time
	var err error

	if startDate == "" {
		now := time.Now()
		startDateTime = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	} else {
		startDateTime, err = time.Parse("2006-01-02", startDate)
		if err != nil {
			return nil, fiber.NewError(fiber.StatusBadRequest, "Invalid start date format (use YYYY-MM-DD)")
		}
	}

	if endDate == "" {
		endDateTime = time.Now()
	} else {
		endDateTime, err = time.Parse("2006-01-02", endDate)
		if err != nil {
			return nil, fiber.NewError(fiber.StatusBadRequest, "Invalid end date format (use YYYY-MM-DD)")
		}
		endDateTime = endDateTime.Add(24 * time.Hour).Add(-time.Second)
	}

	// 1. Fetch products
	var products []model.Product
	pQuery := s.DB.WithContext(c.Context()).Model(&model.Product{})
	if productID != "" {
		pQuery = pQuery.Where("id = ?", productID)
	}
	if err := pQuery.Order("code asc").Find(&products).Error; err != nil {
		s.Log.Errorf("Failed to query products: %v", err)
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Database error")
	}

	// 2. Query beginning balances
	type BegBalRow struct {
		ProductID  uuid.UUID `gorm:"column:product_id"`
		BegBalance int64     `gorm:"column:beg_balance"`
	}
	var begBals []BegBalRow
	qBeg := s.DB.WithContext(c.Context()).Table("stock_transactions").
		Joins("join inventory_batches on inventory_batches.id = stock_transactions.batch_id").
		Where("stock_transactions.created_at < ?", startDateTime)

	if warehouseID != "" {
		qBeg = qBeg.Where("inventory_batches.warehouse_id = ?", warehouseID)
	}
	if productID != "" {
		qBeg = qBeg.Where("inventory_batches.product_id = ?", productID)
	}
	qBeg.Select("inventory_batches.product_id, COALESCE(SUM(CASE WHEN transaction_type = 'IN' THEN stock_transactions.qty WHEN transaction_type = 'OUT' THEN -stock_transactions.qty WHEN transaction_type = 'ADJUSTMENT' THEN stock_transactions.qty ELSE 0 END), 0) as beg_balance").
		Group("inventory_batches.product_id").
		Scan(&begBals)

	begMap := make(map[uuid.UUID]int64)
	for _, r := range begBals {
		begMap[r.ProductID] = r.BegBalance
	}

	// 3. Query period sums (inflow & outflow)
	type PeriodSumRow struct {
		ProductID uuid.UUID `gorm:"column:product_id"`
		InQty     int64     `gorm:"column:in_qty"`
		OutQty    int64     `gorm:"column:out_qty"`
	}
	var periodSums []PeriodSumRow
	qPer := s.DB.WithContext(c.Context()).Table("stock_transactions").
		Joins("join inventory_batches on inventory_batches.id = stock_transactions.batch_id").
		Where("stock_transactions.created_at >= ? AND stock_transactions.created_at <= ?", startDateTime, endDateTime)

	if warehouseID != "" {
		qPer = qPer.Where("inventory_batches.warehouse_id = ?", warehouseID)
	}
	if productID != "" {
		qPer = qPer.Where("inventory_batches.product_id = ?", productID)
	}
	qPer.Select("inventory_batches.product_id, " +
		"COALESCE(SUM(CASE WHEN transaction_type = 'IN' THEN stock_transactions.qty WHEN (transaction_type = 'ADJUSTMENT' AND stock_transactions.qty > 0) THEN stock_transactions.qty ELSE 0 END), 0) as in_qty, " +
		"COALESCE(SUM(CASE WHEN transaction_type = 'OUT' THEN stock_transactions.qty WHEN (transaction_type = 'ADJUSTMENT' AND stock_transactions.qty < 0) THEN -stock_transactions.qty ELSE 0 END), 0) as out_qty").
		Group("inventory_batches.product_id").
		Scan(&periodSums)

	inMap := make(map[uuid.UUID]int64)
	outMap := make(map[uuid.UUID]int64)
	for _, r := range periodSums {
		inMap[r.ProductID] = r.InQty
		outMap[r.ProductID] = r.OutQty
	}

	// 4. Construct items list
	items := make([]StockMutationItem, 0, len(products))
	for _, p := range products {
		beg := begMap[p.ID]
		in := inMap[p.ID]
		out := outMap[p.ID]
		end := beg + in - out

		items = append(items, StockMutationItem{
			ProductID:        p.ID,
			ProductCode:      p.Code,
			ProductName:      p.Name,
			Category:         p.CategoryID,
			Unit:             p.Unit,
			BeginningBalance: beg,
			InQty:            in,
			OutQty:           out,
			EndingBalance:    end,
		})
	}

	return items, nil
}

type DistributionReportItem struct {
	TxID            uuid.UUID `json:"tx_id"`
	Date            time.Time `json:"date"`
	ReferenceNo     string    `json:"reference_no"`
	ProductCode     string    `json:"product_code"`
	ProductName     string    `json:"product_name"`
	Category        string    `json:"category"`
	SubCategory     string    `json:"sub_category"`
	Qty             int       `json:"qty"`
	Unit            string    `json:"unit"`
	CustomerName    string    `json:"customer_name"`
	CustomerPIC     string    `json:"customer_pic"`
	CustomerAddress string    `json:"customer_address"`
	CustomerNPWP    string    `json:"customer_npwp"`
}

func (s *reportService) GetDistributionReport(c *fiber.Ctx, categoryID, subCategory, startDate, endDate string) (interface{}, error) {
	var txs []model.StockTransaction

	query := s.DB.WithContext(c.Context()).
		Preload("Batch").
		Preload("Batch.Product").
		Preload("Batch.Product.PackagingUnit").
		Joins("JOIN inventory_batches ON inventory_batches.id = stock_transactions.batch_id").
		Joins("JOIN products ON products.id = inventory_batches.product_id").
		Where("stock_transactions.transaction_type = ?", "OUT")

	if categoryID != "" {
		query = query.Where("products.category_id = ?", categoryID)
	}

	if subCategory != "" {
		query = query.Where("products.sub_category = ?", subCategory)
	}

	if startDate != "" {
		tMin, err := time.Parse("2006-01-02", startDate)
		if err == nil {
			query = query.Where("stock_transactions.created_at >= ?", tMin)
		}
	}

	if endDate != "" {
		tMax, err := time.Parse("2006-01-02", endDate)
		if err == nil {
			tMax = tMax.Add(24 * time.Hour).Add(-time.Second)
			query = query.Where("stock_transactions.created_at <= ?", tMax)
		}
	}

	if err := query.Order("stock_transactions.created_at desc").Find(&txs).Error; err != nil {
		s.Log.Errorf("Failed to query stock transactions for distribution report: %v", err)
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Database error")
	}

	// Fetch linked Sales Orders & Customer details
	var soIDs []uuid.UUID
	for _, tx := range txs {
		if tx.SOID != nil {
			soIDs = append(soIDs, *tx.SOID)
		}
	}

	soMap := make(map[uuid.UUID]model.SalesOrder)
	if len(soIDs) > 0 {
		var orders []model.SalesOrder
		if err := s.DB.WithContext(c.Context()).Preload("Customer").Where("id IN ?", soIDs).Find(&orders).Error; err == nil {
			for _, order := range orders {
				soMap[order.ID] = order
			}
		}
	}

	items := make([]DistributionReportItem, 0, len(txs))
	for _, tx := range txs {
		custName := "Manual Outward"
		custPIC := ""
		custAddress := ""
		custNPWP := ""

		if tx.SOID != nil {
			if order, exists := soMap[*tx.SOID]; exists {
				custName = order.Customer.Name
				custPIC = order.Customer.PIC
				custAddress = order.Customer.Address
				custNPWP = order.Customer.NPWP
			}
		}

		items = append(items, DistributionReportItem{
			TxID:            tx.ID,
			Date:            tx.CreatedAt,
			ReferenceNo:     tx.ReferenceNo,
			ProductCode:     tx.Batch.Product.Code,
			ProductName:     tx.Batch.Product.Name,
			Category:        tx.Batch.Product.CategoryID,
			SubCategory:     tx.Batch.Product.SubCategory,
			Qty:             tx.Qty,
			Unit:            tx.Batch.Product.Unit,
			CustomerName:    custName,
			CustomerPIC:     custPIC,
			CustomerAddress: custAddress,
			CustomerNPWP:    custNPWP,
		})
	}

	return items, nil
}

type ReorderPointItem struct {
	ProductID        uuid.UUID `json:"product_id"`
	ProductCode      string    `json:"product_code"`
	ProductName      string    `json:"product_name"`
	Category         string    `json:"category"`
	Unit             string    `json:"unit"`
	MinimumStock     int       `json:"minimum_stock"` // Safety Stock
	CurrentStock     int       `json:"current_stock"`
	ADU              float64   `json:"adu"`           // Average Daily Usage
	ROP              int       `json:"rop"`           // Reorder Point
	Status           string    `json:"status"`        // "SAFE", "RESTOCK"
	SuggestedQty     int       `json:"suggested_qty"`
	LastSupplierID   uuid.UUID `json:"last_supplier_id"`
	LastSupplierName string    `json:"last_supplier_name"`
}

func (s *reportService) GetReorderPointReport(c *fiber.Ctx, leadTime int) (interface{}, error) {
	var products []model.Product
	if err := s.DB.WithContext(c.Context()).Order("code asc").Find(&products).Error; err != nil {
		s.Log.Errorf("Failed to query products for reorder point: %v", err)
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Database error")
	}

	items := make([]ReorderPointItem, 0, len(products))
	thirtyDaysAgo := time.Now().AddDate(0, 0, -30)

	for _, p := range products {
		// 1. Calculate current active stock
		var currentStock int64
		s.DB.WithContext(c.Context()).Model(&model.InventoryBatch{}).
			Where("product_id = ? AND status = 'active'", p.ID).
			Select("COALESCE(SUM(qty), 0)").
			Row().Scan(&currentStock)

		// 2. Outbound usage in last 30 days
		var totalOut int64
		s.DB.WithContext(c.Context()).Table("stock_transactions").
			Joins("join inventory_batches on inventory_batches.id = stock_transactions.batch_id").
			Where("inventory_batches.product_id = ? AND stock_transactions.transaction_type = 'OUT' AND stock_transactions.created_at >= ?", p.ID, thirtyDaysAgo).
			Select("COALESCE(SUM(stock_transactions.qty), 0)").
			Row().Scan(&totalOut)

		adu := float64(totalOut) / 30.0
		rop := int(adu*float64(leadTime)) + p.MinimumStock

		status := "SAFE"
		suggestedQty := 0
		if int(currentStock) <= rop {
			status = "RESTOCK"
			// Order enough to cover safety stock + 30 days of average usage
			suggestedQty = int(adu*30.0) + p.MinimumStock - int(currentStock)
			if suggestedQty < p.MinimumStock || suggestedQty <= 0 {
				suggestedQty = p.MinimumStock
			}
		}

		// 3. Find the last supplier used
		type SupplierRow struct {
			ID   uuid.UUID `gorm:"column:id"`
			Name string    `gorm:"column:name"`
		}
		var supRow SupplierRow
		s.DB.WithContext(c.Context()).Table("purchase_order_items").
			Joins("join purchase_orders on purchase_orders.id = purchase_order_items.po_id").
			Joins("join suppliers on suppliers.id = purchase_orders.supplier_id").
			Where("purchase_order_items.product_id = ?", p.ID).
			Order("purchase_orders.order_date desc").
			Select("suppliers.id, suppliers.name").
			Limit(1).
			Scan(&supRow)

		// Fallback to first supplier if none used yet
		if supRow.ID == uuid.Nil {
			var firstSup model.Supplier
			if err := s.DB.WithContext(c.Context()).First(&firstSup).Error; err == nil {
				supRow.ID = firstSup.ID
				supRow.Name = firstSup.Name
			}
		}

		items = append(items, ReorderPointItem{
			ProductID:        p.ID,
			ProductCode:      p.Code,
			ProductName:      p.Name,
			Category:         p.CategoryID,
			Unit:             p.Unit,
			MinimumStock:     p.MinimumStock,
			CurrentStock:     int(currentStock),
			ADU:              adu,
			ROP:              rop,
			Status:           status,
			SuggestedQty:     suggestedQty,
			LastSupplierID:   supRow.ID,
			LastSupplierName: supRow.Name,
		})
	}

	return items, nil
}
