package router

import (
	"app/src/controller"
	m "app/src/middleware"
	"app/src/service"
	"app/src/validation"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

func IMSRoutes(
	v1 fiber.Router,
	u service.UserService,
	prodService service.ProductService,
	batchService service.BatchService,
	txService service.TransactionService,
	dashService service.DashboardService,
	db *gorm.DB,
) {
	prodController := controller.NewProductController(prodService)
	batchController := controller.NewBatchController(batchService)
	txController := controller.NewTransactionController(txService)
	dashController := controller.NewDashboardController(dashService)
	metaController := controller.NewMetaController(db)
	uploadController := controller.NewUploadController()

	// Initialize Supplier service & controller directly inside IMSRoutes
	validate := validation.Validator()
	supService := service.NewSupplierService(db, validate)
	supController := controller.NewSupplierController(supService)

	// Initialize Packaging service & controller directly inside IMSRoutes
	packService := service.NewPackagingService(db, validate)
	packController := controller.NewPackagingController(packService)

	// Initialize Customer service & controller directly inside IMSRoutes
	custService := service.NewCustomerService(db, validate)
	custController := controller.NewCustomerController(custService)

	// Initialize Order service & controller directly inside IMSRoutes
	orderService := service.NewOrderService(db, validate)
	orderController := controller.NewOrderController(orderService)

	// Initialize Activity service & controller
	activityService := service.NewActivityService(db)
	activityController := controller.NewActivityController(activityService)

	// Initialize Report service & controller directly inside IMSRoutes
	repService := service.NewReportService(db)
	repController := controller.NewReportController(repService)

	// Authorization middleware check helper
	auth := m.Auth(u)

	// Products
	products := v1.Group("/products", auth)
	products.Get("/", prodController.GetProducts)
	products.Post("/", prodController.CreateProduct)
	products.Get("/:id", prodController.GetProductByID)
	products.Put("/:id", prodController.UpdateProduct)
	products.Delete("/:id", prodController.DeleteProduct)

	// Suppliers
	suppliers := v1.Group("/suppliers", auth)
	suppliers.Get("/", supController.GetSuppliers)
	suppliers.Post("/", supController.CreateSupplier)
	suppliers.Get("/:id", supController.GetSupplierByID)
	suppliers.Put("/:id", supController.UpdateSupplier)
	suppliers.Delete("/:id", supController.DeleteSupplier)

	// Customers
	customers := v1.Group("/customers", auth)
	customers.Get("/", custController.GetCustomers)
	customers.Post("/", custController.CreateCustomer)
	customers.Get("/:id", custController.GetCustomerByID)
	customers.Put("/:id", custController.UpdateCustomer)
	customers.Delete("/:id", custController.DeleteCustomer)

	// Packaging Units
	packagingUnits := v1.Group("/packaging-units", auth)
	packagingUnits.Get("/", packController.GetPackagingUnits)
	packagingUnits.Post("/", packController.CreatePackagingUnit)
	packagingUnits.Get("/:id", packController.GetPackagingUnitByID)
	packagingUnits.Put("/:id", packController.UpdatePackagingUnit)
	packagingUnits.Delete("/:id", packController.DeletePackagingUnit)

	// Batches
	batches := v1.Group("/inventory/batches", auth)
	batches.Get("/", batchController.GetBatches)
	batches.Get("/:id", batchController.GetBatchByID)
	batches.Put("/:id/status", batchController.UpdateBatchStatus)

	// Inward Stock (Barang Masuk)
	v1.Post("/inventory/in", auth, txController.CreateInwardTransaction)
	v1.Get("/inventory/in", auth, txController.GetTransactions) // Retrieves transactions filtered by type

	// Outward Stock FEFO (Barang Keluar)
	v1.Post("/inventory/out", auth, txController.CreateOutwardTransaction)
	v1.Get("/inventory/out", auth, txController.GetTransactions)

	// Expiry Alerts
	v1.Get("/inventory/expiry-alerts", auth, batchController.GetExpiryAlerts)

	// B3 Inward Batch Approval
	v1.Put("/inventory/batches/:id/approve", auth, txController.ApproveB3Inward)

	// Purchase Orders
	v1.Get("/orders/po", auth, orderController.GetPurchaseOrders)
	v1.Post("/orders/po", auth, orderController.CreatePurchaseOrder)
	v1.Put("/orders/po/:id/approve", auth, orderController.ApprovePurchaseOrder)

	// Sales Orders
	v1.Get("/orders/so", auth, orderController.GetSalesOrders)
	v1.Post("/orders/so", auth, orderController.CreateSalesOrder)
	v1.Put("/orders/so/:id/approve", auth, orderController.ApproveSalesOrder)
	v1.Patch("/orders/so/:id/payment", auth, orderController.UpdateSalesOrderPaymentStatus)

	// File Upload
	v1.Post("/upload", auth, uploadController.UploadFile)

	// Transaction Edit / Completion
	v1.Put("/inventory/transactions/:id", auth, txController.UpdateTransaction)
	v1.Put("/inventory/transactions/:id/complete", auth, txController.CompleteTransaction)
	v1.Post("/inventory/transactions/:id/confirm-pick", auth, txController.ConfirmPick)

	// Stock Opname
	v1.Post("/stock-opname", auth, txController.CreateStockOpname)
	v1.Get("/inventory/adjustment", auth, txController.GetTransactions)

	// Activity Log (Audit Trail)
	v1.Get("/activity-logs", auth, activityController.GetActivityLogs)

	// Dashboard
	v1.Get("/dashboard", auth, dashController.GetDashboardData)

	// Reports
	reports := v1.Group("/reports", auth)
	reports.Get("/inventory-value", repController.GetInventoryValueReport)
	reports.Get("/stock-aging", repController.GetStockAgingReport)
	reports.Get("/stock-mutation", repController.GetStockMutationReport)
	reports.Get("/distribution", repController.GetDistributionReport)
	reports.Get("/reorder-point", repController.GetReorderPointReport)

	// Metadata Dropdowns
	v1.Get("/warehouses", auth, metaController.GetWarehouses)
	v1.Get("/locations", auth, metaController.GetLocations)
}
