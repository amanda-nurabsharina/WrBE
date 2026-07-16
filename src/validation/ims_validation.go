package validation

type CreateProduct struct {
	Code             string  `json:"code" validate:"required,min=3,max=50"`
	Barcode          string  `json:"barcode" validate:"max=50"`
	Name             string  `json:"name" validate:"required,min=3,max=100"`
	CategoryID       string  `json:"category_id" validate:"max=50"`
	Unit             string  `json:"unit" validate:"required,max=20"`
	MinimumStock     int     `json:"minimum_stock" validate:"min=0"`
	RegCategory      string  `json:"reg_category" validate:"required,max=20"`
	KementanRegNo    string  `json:"kementan_reg_no" validate:"max=100"`
	MSDSReference    string  `json:"msds_reference" validate:"max=255"`
	SubCategory      string  `json:"sub_category" validate:"required,max=50"`
	PackagingUnitID  string  `json:"packaging_unit_id" validate:"required,uuid"`
	ConversionRatio  int     `json:"conversion_ratio" validate:"required,min=1"`
	PurchasePrice    float64 `json:"purchase_price" validate:"min=0"`
	PriceDistributor float64 `json:"price_distributor" validate:"min=0"`
	PriceRetail      float64 `json:"price_retail" validate:"min=0"`
}

type UpdateProduct struct {
	Barcode          string   `json:"barcode" validate:"max=50"`
	Name             string   `json:"name" validate:"min=3,max=100"`
	CategoryID       string   `json:"category_id" validate:"max=50"`
	Unit             string   `json:"unit" validate:"max=20"`
	MinimumStock     *int     `json:"minimum_stock" validate:"omitempty,min=0"`
	RegCategory      string   `json:"reg_category" validate:"max=20"`
	KementanRegNo    string   `json:"kementan_reg_no" validate:"max=100"`
	MSDSReference    string   `json:"msds_reference" validate:"max=255"`
	SubCategory      string   `json:"sub_category" validate:"max=50"`
	PackagingUnitID  string   `json:"packaging_unit_id" validate:"omitempty,uuid"`
	ConversionRatio  *int     `json:"conversion_ratio" validate:"omitempty,min=1"`
	PurchasePrice    *float64 `json:"purchase_price" validate:"omitempty,min=0"`
	PriceDistributor *float64 `json:"price_distributor" validate:"omitempty,min=0"`
	PriceRetail      *float64 `json:"price_retail" validate:"omitempty,min=0"`
}

type CreatePackagingUnit struct {
	Code        string `json:"code" validate:"required,min=2,max=50"`
	Name        string `json:"name" validate:"required,min=2,max=100"`
	Description string `json:"description" validate:"max=255"`
}

type UpdatePackagingUnit struct {
	Name        string `json:"name" validate:"required,min=2,max=100"`
	Description string `json:"description" validate:"max=255"`
}

type InwardRequest struct {
	SupplierID    string  `json:"supplier_id" validate:"required,uuid"`
	InvoiceNo     string  `json:"invoice_no" validate:"required,min=3,max=50"`
	ProductID     string  `json:"product_id" validate:"required,uuid"`
	BatchNumber   string  `json:"batch_number" validate:"required,min=2,max=50"`
	ExpiredDate   string  `json:"expired_date" validate:"required"` // Format: YYYY-MM-DD
	Qty           int     `json:"qty" validate:"required,gt=0"`
	PurchasePrice float64 `json:"purchase_price" validate:"required,gt=0"`
	WarehouseID   string  `json:"warehouse_id" validate:"required,uuid"`
	LocationID    string  `json:"location_id" validate:"required,uuid"`
	POID          string  `json:"po_id" validate:"omitempty,uuid"`
}

type OutwardRequest struct {
	ProductID    string  `json:"product_id" validate:"required,uuid"`
	Qty          int     `json:"qty" validate:"required,gt=0"`
	Purpose      string  `json:"purpose" validate:"max=100"`
	Description  string  `json:"description" validate:"max=255"`
	SOID         string  `json:"so_id" validate:"omitempty,uuid"`
	SellingPrice float64 `json:"selling_price" validate:"omitempty,min=0"`
}

type StockOpnameRequest struct {
	BatchID     string `json:"batch_id" validate:"required,uuid"`
	PhysicalQty int    `json:"physical_qty" validate:"min=0"`
	Description string `json:"description" validate:"max=255"`
}

type CreateSupplier struct {
	Name        string `json:"name" validate:"required,min=2,max=100"`
	Phone       string `json:"phone" validate:"max=20"`
	Email       string `json:"email" validate:"omitempty,email,max=100"`
	PIC         string `json:"pic" validate:"max=100"`
	Address     string `json:"address" validate:"max=500"`
	NPWP        string `json:"npwp" validate:"max=50"`
	PaymentTerm int    `json:"payment_term" validate:"min=0"`
}

type UpdateSupplier struct {
	Name        string `json:"name" validate:"omitempty,min=2,max=100"`
	Phone       string `json:"phone" validate:"max=20"`
	Email       string `json:"email" validate:"omitempty,email,max=100"`
	PIC         string `json:"pic" validate:"max=100"`
	Address     string `json:"address" validate:"max=500"`
	NPWP        string `json:"npwp" validate:"max=50"`
	PaymentTerm *int   `json:"payment_term" validate:"omitempty,min=0"`
}

type CreateCustomer struct {
	Name        string `json:"name" validate:"required,min=2,max=100"`
	Phone       string `json:"phone" validate:"max=20"`
	Email       string `json:"email" validate:"omitempty,email,max=100"`
	PIC         string `json:"pic" validate:"max=100"`
	Address     string `json:"address" validate:"max=500"`
	NPWP        string `json:"npwp" validate:"max=50"`
	PaymentTerm int    `json:"payment_term" validate:"min=0"`
	PriceTier   string `json:"price_tier" validate:"required,oneof=distributor retail"`
}

type UpdateCustomer struct {
	Name        string `json:"name" validate:"omitempty,min=2,max=100"`
	Phone       string `json:"phone" validate:"max=20"`
	Email       string `json:"email" validate:"omitempty,email,max=100"`
	PIC         string `json:"pic" validate:"max=100"`
	Address     string `json:"address" validate:"max=500"`
	NPWP        string `json:"npwp" validate:"max=50"`
	PaymentTerm *int   `json:"payment_term" validate:"omitempty,min=0"`
	PriceTier   string `json:"price_tier" validate:"omitempty,oneof=distributor retail"`
}

type PurchaseOrderItemReq struct {
	ProductID string  `json:"product_id" validate:"required,uuid"`
	Qty       int     `json:"qty" validate:"required,gt=0"`
	Price     float64 `json:"price" validate:"required,min=0"`
}

type CreatePurchaseOrder struct {
	SupplierID string                 `json:"supplier_id" validate:"required,uuid"`
	PONumber   string                 `json:"po_number" validate:"required,min=3,max=50"`
	OrderDate  string                 `json:"order_date" validate:"required"` // Format: YYYY-MM-DD
	Items      []PurchaseOrderItemReq `json:"items" validate:"required,min=1,dive"`
}

type SalesOrderItemReq struct {
	ProductID string  `json:"product_id" validate:"required,uuid"`
	Qty       int     `json:"qty" validate:"required,gt=0"`
	Price     float64 `json:"price" validate:"required,min=0"`
}

type CreateSalesOrder struct {
	CustomerID string              `json:"customer_id" validate:"required,uuid"`
	SONumber   string              `json:"so_number" validate:"required,min=3,max=50"`
	OrderDate  string              `json:"order_date" validate:"required"` // Format: YYYY-MM-DD
	Items      []SalesOrderItemReq `json:"items" validate:"required,min=1,dive"`
}
