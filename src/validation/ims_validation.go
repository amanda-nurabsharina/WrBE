package validation

type CreateProduct struct {
	Code         string `json:"code" validate:"required,min=3,max=50"`
	Barcode      string `json:"barcode" validate:"max=50"`
	Name         string `json:"name" validate:"required,min=3,max=100"`
	CategoryID   string `json:"category_id" validate:"max=50"`
	Unit         string `json:"unit" validate:"required,max=20"`
	MinimumStock int    `json:"minimum_stock" validate:"min=0"`
}

type UpdateProduct struct {
	Barcode      string `json:"barcode" validate:"max=50"`
	Name         string `json:"name" validate:"min=3,max=100"`
	CategoryID   string `json:"category_id" validate:"max=50"`
	Unit         string `json:"unit" validate:"max=20"`
	MinimumStock *int   `json:"minimum_stock" validate:"omitempty,min=0"`
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
}

type OutwardRequest struct {
	ProductID   string `json:"product_id" validate:"required,uuid"`
	Qty         int    `json:"qty" validate:"required,gt=0"`
	Purpose     string `json:"purpose" validate:"max=100"`
	Description string `json:"description" validate:"max=255"`
}

type StockOpnameRequest struct {
	BatchID     string `json:"batch_id" validate:"required,uuid"`
	PhysicalQty int    `json:"physical_qty" validate:"min=0"`
	Description string `json:"description" validate:"max=255"`
}

type CreateSupplier struct {
	Name  string `json:"name" validate:"required,min=2,max=100"`
	Phone string `json:"phone" validate:"max=20"`
	Email string `json:"email" validate:"omitempty,email,max=100"`
}

type UpdateSupplier struct {
	Name  string `json:"name" validate:"min=2,max=100"`
	Phone string `json:"phone" validate:"max=20"`
	Email string `json:"email" validate:"omitempty,email,max=100"`
}
