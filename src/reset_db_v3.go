//go:build ignore

package main

import (
	"app/src/config"
	"app/src/database"
	"app/src/model"
	"fmt"
	"gorm.io/gorm"
)

func main() {
	db := database.Connect(config.DBHost, config.DBName)
	
	fmt.Println("Resetting database tables for PO/SO and B3 updates...")
	
	// Delete in reverse order of foreign keys
	db.Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&model.StockTransaction{})
	db.Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&model.InventoryBatch{})
	db.Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&model.SalesOrderItem{})
	db.Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&model.SalesOrder{})
	db.Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&model.PurchaseOrderItem{})
	db.Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&model.PurchaseOrder{})
	db.Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&model.Product{})
	db.Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&model.Supplier{})
	db.Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&model.Customer{})
	db.Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&model.PackagingUnit{})

	fmt.Println("Database tables deleted successfully!")
}
