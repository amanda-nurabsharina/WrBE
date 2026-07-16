package database

import (
	"app/src/config"
	"app/src/model"
	"app/src/utils"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/driver/postgres"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

func Connect(dbHost, dbName string) *gorm.DB {
	var db *gorm.DB
	var err error

	logMode := logger.Info
	if config.IsProd {
		logMode = logger.Error
	}

	// 1. Try PostgreSQL using DBConfig if Host is configured
	if config.DBConfig.Host != "" {
		dsn := fmt.Sprintf(
			"host=%s user=%s password=%s dbname=%s port=%s sslmode=%s TimeZone=Asia/Shanghai",
			config.DBConfig.Host, config.DBConfig.User, config.DBConfig.Password, config.DBConfig.DBName, config.DBConfig.Port, config.DBConfig.SSLMode,
		)
		utils.Log.Infof("Attempting to connect to PostgreSQL: %s:%s/%s", config.DBConfig.Host, config.DBConfig.Port, config.DBConfig.DBName)
		db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{
			Logger:                 logger.Default.LogMode(logMode),
			SkipDefaultTransaction: true,
			PrepareStmt:            true,
			TranslateError:         true,
		})
	}

	// 2. Fall back to SQLite local database if Postgres connection fails or isn't used
	if db == nil || err != nil {
		if err != nil {
			utils.Log.Warnf("PostgreSQL connection failed: %v. Falling back to SQLite.", err)
		} else {
			utils.Log.Info("Using SQLite local database (warehouse.db)")
		}

		db, err = gorm.Open(sqlite.Open("warehouse.db"), &gorm.Config{
			Logger:                 logger.Default.LogMode(logMode),
			SkipDefaultTransaction: true,
			PrepareStmt:            true,
			TranslateError:         true,
		})
		if err != nil {
			utils.Log.Fatalf("Failed to initialize SQLite database: %v", err)
		}
	}

	sqlDB, errDB := db.DB()
	if errDB != nil {
		utils.Log.Errorf("Failed to connect to database: %+v", errDB)
	} else {
		// Config connection pooling
		sqlDB.SetMaxIdleConns(10)
		sqlDB.SetMaxOpenConns(100)
		sqlDB.SetConnMaxLifetime(60 * time.Minute)
	}

	// Run AutoMigrate
	utils.Log.Info("Running database migrations...")
	if err := db.AutoMigrate(
		&model.User{}, &model.Token{}, &model.Role{},
		&model.PackagingUnit{}, &model.Product{}, &model.Warehouse{}, &model.Location{},
		&model.Supplier{}, &model.Customer{},
		&model.PurchaseOrder{}, &model.PurchaseOrderItem{},
		&model.SalesOrder{}, &model.SalesOrderItem{},
		&model.InventoryBatch{}, &model.StockTransaction{},
		&model.ActivityLog{},
	); err != nil {
		utils.Log.Errorf("Failed to auto-migrate tables: %v", err)
	} else {
		utils.Log.Info("Database migrations completed successfully")
		// Seed default data if not present
		seedDatabase(db)
	}

	DB = db
	return db
}

func seedDatabase(db *gorm.DB) {
	// 1. Seed/Update Roles
	utils.Log.Info("Seeding roles...")
	for _, roleCfg := range config.DefaultRoles {
		var existing model.Role
		err := db.Where("name = ?", roleCfg.Name).First(&existing).Error
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				newRole := model.Role{
					Name:            roleCfg.Name,
					DisplayName:     roleCfg.DisplayName,
					Description:     roleCfg.Description,
					AccessibleMenus: model.StringArray(roleCfg.AccessibleMenus),
					Permissions:     model.PermissionMap(roleCfg.Permissions),
					CreatedAt:       time.Now(),
					UpdatedAt:       time.Now(),
				}
				if err := db.Create(&newRole).Error; err != nil {
					utils.Log.Errorf("Failed to seed role %s: %v", roleCfg.Name, err)
				} else {
					utils.Log.Infof("Seeded role: %s", roleCfg.Name)
				}
			} else {
				utils.Log.Errorf("Error checking role %s: %v", roleCfg.Name, err)
			}
		} else {
			// Update permissions and accessible menus if they are empty/null or out of sync
			needsUpdate := false
			if len(existing.Permissions) == 0 && len(roleCfg.Permissions) > 0 {
				existing.Permissions = model.PermissionMap(roleCfg.Permissions)
				needsUpdate = true
			}
			// Update accessible menus if they are shorter or mismatch
			if len(existing.AccessibleMenus) < len(roleCfg.AccessibleMenus) {
				existing.AccessibleMenus = model.StringArray(roleCfg.AccessibleMenus)
				needsUpdate = true
			}
			if needsUpdate {
				existing.UpdatedAt = time.Now()
				if err := db.Save(&existing).Error; err != nil {
					utils.Log.Errorf("Failed to update role %s: %v", roleCfg.Name, err)
				} else {
					utils.Log.Infof("Updated existing role: %s with new permissions/menus", roleCfg.Name)
				}
			}
		}
	}

	// 2. Seed Users from config (superadmin, etc.)
	utils.Log.Info("Seeding default users...")
	var seededSuperAdmin model.User
	for _, userCfg := range config.DefaultUsers {
		var count int64
		db.Model(&model.User{}).Where("email = ?", userCfg.Email).Count(&count)
		if count == 0 {
			hashedPassword, err := utils.HashPassword(userCfg.Password)
			if err != nil {
				utils.Log.Errorf("Failed to hash password for user %s: %v", userCfg.Username, err)
				continue
			}
			newUser := model.User{
				Name:          userCfg.FirstName + " " + userCfg.LastName,
				Email:         userCfg.Email,
				Password:      hashedPassword,
				Role:          userCfg.Role,
				Department:    "Executive",
				Position:      "Super Administrator",
				Status:        "active",
				VerifiedEmail: true,
			}
			if err := db.Create(&newUser).Error; err != nil {
				utils.Log.Errorf("Failed to seed user %s: %v", userCfg.Email, err)
			} else {
				utils.Log.Infof("Successfully seeded user: %s (role: %s)", userCfg.Email, userCfg.Role)
				seededSuperAdmin = newUser
			}
		} else {
			db.Where("email = ?", userCfg.Email).First(&seededSuperAdmin)
		}
	}

	// 3. Seed legacy admin if no user exists at all
	var userCount int64
	db.Model(&model.User{}).Count(&userCount)
	if userCount == 0 || userCount == 1 { // if only super_admin exists or 0 users
		var legacyCount int64
		db.Model(&model.User{}).Where("email = ?", "admin@warehouse.com").Count(&legacyCount)
		if legacyCount == 0 {
			utils.Log.Info("Seeding database with legacy admin user...")
			hashedPassword, err := utils.HashPassword("admin123")
			if err != nil {
				utils.Log.Errorf("Failed to hash admin password for seeding: %v", err)
				return
			}
			admin := model.User{
				Name:          "System Admin",
				Email:         "admin@warehouse.com",
				Password:      hashedPassword,
				Role:          "admin",
				Department:    "IT & Security",
				Position:      "System Administrator",
				Status:        "active",
				VerifiedEmail: true,
			}
			if err := db.Create(&admin).Error; err != nil {
				utils.Log.Errorf("Failed to seed admin user: %v", err)
			} else {
				utils.Log.Info("Successfully seeded legacy admin user: admin@warehouse.com / admin123")
			}
		}
	}

	// 4. Seed Warehouse
	utils.Log.Info("Seeding default warehouses...")
	var warehouse model.Warehouse
	var whCount int64
	db.Model(&model.Warehouse{}).Count(&whCount)
	if whCount == 0 {
		warehouse = model.Warehouse{
			Code: "WH-MAIN",
			Name: "Main Warehouse",
		}
		if err := db.Create(&warehouse).Error; err != nil {
			utils.Log.Errorf("Failed to seed warehouse: %v", err)
		} else {
			utils.Log.Info("Successfully seeded default warehouse: WH-MAIN")
		}
	} else {
		db.First(&warehouse)
	}

	// 5. Seed Rack Locations
	utils.Log.Info("Seeding default rack locations...")
	var rack1, rack2, rack3 model.Location
	var locCount int64
	db.Model(&model.Location{}).Count(&locCount)
	if locCount == 0 && warehouse.ID != uuid.Nil {
		rack1 = model.Location{WarehouseID: warehouse.ID, Rack: "Rack-A1"}
		rack2 = model.Location{WarehouseID: warehouse.ID, Rack: "Rack-A2"}
		rack3 = model.Location{WarehouseID: warehouse.ID, Rack: "Rack-B1"}
		db.Create(&rack1)
		db.Create(&rack2)
		db.Create(&rack3)
		utils.Log.Info("Successfully seeded default rack locations (Rack-A1, Rack-A2, Rack-B1)")
	} else {
		db.Where("rack = ?", "Rack-A1").First(&rack1)
		db.Where("rack = ?", "Rack-A2").First(&rack2)
		db.Where("rack = ?", "Rack-B1").First(&rack3)
	}

	// 5.5. Seed Packaging Units
	utils.Log.Info("Seeding default packaging units...")
	var packCount int64
	db.Model(&model.PackagingUnit{}).Count(&packCount)
	if packCount == 0 {
		units := []model.PackagingUnit{
			{Code: "DUS", Name: "Dus", Description: "Kemasan karton dus"},
			{Code: "KRG", Name: "Karung", Description: "Kemasan karung sak"},
			{Code: "BTL", Name: "Botol", Description: "Kemasan botol plastik/kaca"},
			{Code: "JRG", Name: "Jerigen", Description: "Kemasan jerigen plastik"},
			{Code: "DRM", Name: "Drum", Description: "Kemasan drum besi/plastik"},
			{Code: "PCS", Name: "Pcs", Description: "Kemasan satuan pieces"},
		}
		for _, u := range units {
			db.Create(&u)
		}
		utils.Log.Info("Successfully seeded default packaging units")
	}

	var dusPack, krgPack model.PackagingUnit
	db.Where("code = ?", "DUS").First(&dusPack)
	db.Where("code = ?", "KRG").First(&krgPack)

	// 6. Seed Suppliers
	utils.Log.Info("Seeding default suppliers...")
	var supCount int64
	db.Model(&model.Supplier{}).Count(&supCount)
	if supCount == 0 {
		suppliers := []model.Supplier{
			{
				Name:        "SANTANI Agro Mandiri",
				Phone:       "021-543210",
				Email:       "contact@santani.co.id",
				PIC:         "Ir. Hermawan",
				Address:     "Gedung Santani, Jl. Jend. Sudirman Kav. 21, Jakarta",
				NPWP:        "01.111.222.3-444.000",
				PaymentTerm: 45,
			},
			{
				Name:        "Syngenta Indonesia",
				Phone:       "021-300488",
				Email:       "info.indonesia@syngenta.com",
				PIC:         "Rian Hidayat",
				Address:     "Cilandak Commercial Estate, Building 100, Jakarta",
				NPWP:        "02.222.333.4-555.000",
				PaymentTerm: 30,
			},
		}
		for _, sup := range suppliers {
			db.Create(&sup)
		}
		utils.Log.Info("Successfully seeded default suppliers")
	}

	// 6.5. Seed Customers
	utils.Log.Info("Seeding default customers...")
	var custCount int64
	db.Model(&model.Customer{}).Count(&custCount)
	if custCount == 0 {
		customers := []model.Customer{
			{
				Name:        "Toko Tani Makmur",
				Phone:       "0812-3456-7890",
				Email:       "makmur@tokotani.com",
				PIC:         "Haji Makmur",
				Address:     "Jl. Raya Pertanian No. 12, Sleman, Yogyakarta",
				NPWP:        "01.234.567.8-901.000",
				PaymentTerm: 30,
				PriceTier:   "retail",
			},
			{
				Name:        "Distributor Tani Nusantara",
				Phone:       "021-7654-3210",
				Email:       "contact@taninusantara.com",
				PIC:         "Budi Santoso",
				Address:     "Kawasan Industri Cikarang Blok B-3, Bekasi",
				NPWP:        "02.345.678.9-012.000",
				PaymentTerm: 60,
				PriceTier:   "distributor",
			},
		}
		for _, cust := range customers {
			db.Create(&cust)
		}
		utils.Log.Info("Successfully seeded default customers")
	}

	// 7. Seed Products
	utils.Log.Info("Seeding default products...")
	var p1, p2, p3 model.Product
	var prodCount int64
	db.Model(&model.Product{}).Count(&prodCount)
	if prodCount == 0 {
		p1 = model.Product{
			Code:             "NPK-15",
			Barcode:          "8991234567890",
			Name:             "Santani NPK 15-15-15",
			CategoryID:       "Pupuk",
			SubCategory:      "Pupuk",
			RegCategory:      "non-B3",
			Unit:             "Kg",
			PackagingUnitID:  krgPack.ID,
			ConversionRatio:  50,
			PurchasePrice:    450000,
			PriceDistributor: 480000,
			PriceRetail:      500000,
			MinimumStock:     100,
		}
		p2 = model.Product{
			Code:             "GLY-480",
			Barcode:          "8991234567891",
			Name:             "Santani Glyphosate 480 SL",
			CategoryID:       "Pestisida",
			SubCategory:      "Herbisida",
			RegCategory:      "B3",
			Unit:             "Liter",
			PackagingUnitID:  dusPack.ID,
			ConversionRatio:  20,
			PurchasePrice:    950000,
			PriceDistributor: 1050000,
			PriceRetail:      1100000,
			MinimumStock:     50,
		}
		p3 = model.Product{
			Code:             "CAR-50",
			Barcode:          "8991234567892",
			Name:             "Santani Carbendazim 50 WP",
			CategoryID:       "Pestisida",
			SubCategory:      "Fungisida",
			RegCategory:      "B3",
			Unit:             "Kg",
			PackagingUnitID:  dusPack.ID,
			ConversionRatio:  10,
			PurchasePrice:    650000,
			PriceDistributor: 720000,
			PriceRetail:      750000,
			MinimumStock:     30,
		}
		db.Create(&p1)
		db.Create(&p2)
		db.Create(&p3)
		utils.Log.Info("Successfully seeded default products")
	} else {
		db.Where("code = ?", "NPK-15").First(&p1)
		db.Where("code = ?", "GLY-480").First(&p2)
		db.Where("code = ?", "CAR-50").First(&p3)
	}

	// 8. Seed Batches to demonstrate FEFO and warning categories
	utils.Log.Info("Seeding default inventory batches...")
	var batchCount int64
	db.Model(&model.InventoryBatch{}).Count(&batchCount)
	if batchCount == 0 && p1.ID != uuid.Nil && warehouse.ID != uuid.Nil && rack1.ID != uuid.Nil {
		now := time.Now()
		// Red: Expired
		b1 := model.InventoryBatch{
			ProductID:     p1.ID,
			BatchNumber:   "B-NPK-EXP",
			ExpiredDate:   now.AddDate(0, 0, -10), // expired 10 days ago
			Qty:           20,
			WarehouseID:   warehouse.ID,
			LocationID:    rack1.ID,
			PurchasePrice: 450000,
			Status:        "expired",
		}
		// Orange: Near-Expired (<30 days)
		b2 := model.InventoryBatch{
			ProductID:     p1.ID,
			BatchNumber:   "B-NPK-30D",
			ExpiredDate:   now.AddDate(0, 0, 15), // expires in 15 days
			Qty:           50,
			WarehouseID:   warehouse.ID,
			LocationID:    rack1.ID,
			PurchasePrice: 450000,
			Status:        "active",
		}
		// Yellow: Hampir Expired (30-90 days)
		b3 := model.InventoryBatch{
			ProductID:     p1.ID,
			BatchNumber:   "B-NPK-90D",
			ExpiredDate:   now.AddDate(0, 0, 60), // expires in 60 days
			Qty:           30,
			WarehouseID:   warehouse.ID,
			LocationID:    rack2.ID,
			PurchasePrice: 450000,
			Status:        "active",
		}
		// Green: Safe (>90 days)
		b4 := model.InventoryBatch{
			ProductID:     p1.ID,
			BatchNumber:   "B-NPK-SAFE",
			ExpiredDate:   now.AddDate(0, 0, 180), // expires in 180 days
			Qty:           100,
			WarehouseID:   warehouse.ID,
			LocationID:    rack2.ID,
			PurchasePrice: 450000,
			Status:        "active",
		}

		// Glyphosate batches
		b5 := model.InventoryBatch{
			ProductID:     p2.ID,
			BatchNumber:   "B-GLY-SAFE",
			ExpiredDate:   now.AddDate(0, 0, 120), // expires in 120 days
			Qty:           80,
			WarehouseID:   warehouse.ID,
			LocationID:    rack3.ID,
			PurchasePrice: 950000,
			Status:        "active",
		}

		db.Create(&b1)
		db.Create(&b2)
		db.Create(&b3)
		db.Create(&b4)
		db.Create(&b5)

		// Create corresponding IN stock transactions
		if seededSuperAdmin.ID != uuid.Nil {
			db.Create(&model.StockTransaction{BatchID: b1.ID, TransactionType: "IN", Qty: 20, ReferenceNo: "TX-IN-001", UserID: seededSuperAdmin.ID})
			db.Create(&model.StockTransaction{BatchID: b2.ID, TransactionType: "IN", Qty: 50, ReferenceNo: "TX-IN-001", UserID: seededSuperAdmin.ID})
			db.Create(&model.StockTransaction{BatchID: b3.ID, TransactionType: "IN", Qty: 30, ReferenceNo: "TX-IN-002", UserID: seededSuperAdmin.ID})
			db.Create(&model.StockTransaction{BatchID: b4.ID, TransactionType: "IN", Qty: 100, ReferenceNo: "TX-IN-002", UserID: seededSuperAdmin.ID})
			db.Create(&model.StockTransaction{BatchID: b5.ID, TransactionType: "IN", Qty: 80, ReferenceNo: "TX-IN-003", UserID: seededSuperAdmin.ID})
		}
		utils.Log.Info("Successfully seeded default inventory batches and transaction logs")
	}
}
