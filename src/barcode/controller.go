package barcode

import (
	"app/src/model"
	"app/src/response"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type Controller struct {
	service Service
}

func NewController(service Service) *Controller {
	return &Controller{service: service}
}

func (h *Controller) Lookup(c *fiber.Ctx) error {
	barcodeStr := c.Query("barcode")
	if barcodeStr == "" {
		return fiber.NewError(fiber.StatusBadRequest, "Barcode parameter is required")
	}

	reg, entity, err := h.service.Lookup(barcodeStr)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, err.Error())
	}

	return c.Status(fiber.StatusOK).JSON(response.SuccessWithData[interface{}]{
		Code:    fiber.StatusOK,
		Status:  "success",
		Message: "Barcode lookup completed",
		Data: fiber.Map{
			"registry": reg,
			"entity":   entity,
		},
	})
}

type ValidateRequest struct {
	Barcode             string  `json:"barcode" validate:"required"`
	ExpectedWarehouseID *string `json:"expected_warehouse_id"`
	ExpectedLocationID  *string `json:"expected_location_id"`
}

func (h *Controller) Validate(c *fiber.Ctx) error {
	var req ValidateRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid request body")
	}

	var whID, locID *uuid.UUID
	if req.ExpectedWarehouseID != nil && *req.ExpectedWarehouseID != "" {
		id, err := uuid.Parse(*req.ExpectedWarehouseID)
		if err == nil {
			whID = &id
		}
	}
	if req.ExpectedLocationID != nil && *req.ExpectedLocationID != "" {
		id, err := uuid.Parse(*req.ExpectedLocationID)
		if err == nil {
			locID = &id
		}
	}

	reg, entity, err := h.service.Validate(req.Barcode, whID, locID)
	if err != nil {
		// Log the failed validation scan if user is authenticated
		if user, ok := c.Locals("user").(*model.User); ok {
			ip := c.IP()
			device := c.Get("User-Agent")
			_ = h.service.Scan(req.Barcode, user.ID, "VALIDATION", "FAILED", err.Error(), ip, device, nil)
		}
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	// Log successful validation
	if user, ok := c.Locals("user").(*model.User); ok {
		ip := c.IP()
		device := c.Get("User-Agent")
		_ = h.service.Scan(req.Barcode, user.ID, "VALIDATION", "SUCCESS", "Validated successfully", ip, device, nil)
	}

	return c.Status(fiber.StatusOK).JSON(response.SuccessWithData[interface{}]{
		Code:    fiber.StatusOK,
		Status:  "success",
		Message: "Barcode validated successfully",
		Data: fiber.Map{
			"registry": reg,
			"entity":   entity,
		},
	})
}

func (h *Controller) ValidatePick(c *fiber.Ctx) error {
	var req ValidateRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid request body")
	}

	var whID, locID *uuid.UUID
	if req.ExpectedWarehouseID != nil && *req.ExpectedWarehouseID != "" {
		id, err := uuid.Parse(*req.ExpectedWarehouseID)
		if err == nil {
			whID = &id
		}
	}
	if req.ExpectedLocationID != nil && *req.ExpectedLocationID != "" {
		id, err := uuid.Parse(*req.ExpectedLocationID)
		if err == nil {
			locID = &id
		}
	}

	reg, batch, err := h.service.ValidatePick(req.Barcode, whID, locID)
	user, _ := c.Locals("user").(*model.User)
	userID := uuid.Nil
	if user != nil {
		userID = user.ID
	}

	if err != nil {
		ip := c.IP()
		device := c.Get("User-Agent")
		_ = h.service.Scan(req.Barcode, userID, "PICKING_VALIDATION", "FAILED", err.Error(), ip, device, nil)
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	if user != nil {
		ip := c.IP()
		device := c.Get("User-Agent")
		_ = h.service.Scan(req.Barcode, userID, "PICKING_VALIDATION", "SUCCESS", "Pick barcode validated successfully", ip, device, nil)
	}

	return c.Status(fiber.StatusOK).JSON(response.SuccessWithData[interface{}]{
		Code:    fiber.StatusOK,
		Status:  "success",
		Message: "Barcode pick validated successfully",
		Data: fiber.Map{
			"registry": reg,
			"entity":   batch,
		},
	})
}

type ScanLogRequest struct {
	Barcode   string  `json:"barcode" validate:"required"`
	Action    string  `json:"action" validate:"required"` // e.g. RECEIVING, PUT_AWAY, PICKING, OPNAME
	Status    string  `json:"status" validate:"required"` // SUCCESS, FAILED
	Message   string  `json:"message"`
	SessionID *string `json:"session_id"`
}

func (h *Controller) RecordScan(c *fiber.Ctx) error {
	var req ScanLogRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid request body")
	}

	user, ok := c.Locals("user").(*model.User)
	if !ok {
		return fiber.NewError(fiber.StatusUnauthorized, "Unauthorized")
	}

	var sessionID *uuid.UUID
	if req.SessionID != nil && *req.SessionID != "" {
		id, err := uuid.Parse(*req.SessionID)
		if err == nil {
			sessionID = &id
		}
	}

	ip := c.IP()
	device := c.Get("User-Agent")

	err := h.service.Scan(req.Barcode, user.ID, req.Action, req.Status, req.Message, ip, device, sessionID)
	if err != nil {
		return err
	}

	return c.Status(fiber.StatusOK).JSON(response.SuccessWithData[interface{}]{
		Code:    fiber.StatusOK,
		Status:  "success",
		Message: "Scan logged successfully",
		Data:    nil,
	})
}

type PrintLogRequest struct {
	Barcode   string `json:"barcode" validate:"required"`
	LabelType string `json:"label_type" validate:"required"` // PRODUCT, BATCH
	Qty       int    `json:"qty" validate:"required,min=1"`
	Reason    string `json:"reason" validate:"required"`     // Initial Print, Reprint, Replacement
}

func (h *Controller) RecordPrint(c *fiber.Ctx) error {
	var req PrintLogRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid request body")
	}

	user, ok := c.Locals("user").(*model.User)
	if !ok {
		return fiber.NewError(fiber.StatusUnauthorized, "Unauthorized")
	}

	err := h.service.Print(req.Barcode, req.LabelType, user.ID, req.Qty, req.Reason)
	if err != nil {
		return err
	}

	return c.Status(fiber.StatusOK).JSON(response.SuccessWithData[interface{}]{
		Code:    fiber.StatusOK,
		Status:  "success",
		Message: "Barcode print event logged successfully",
		Data:    nil,
	})
}

type StartSessionRequest struct {
	SessionType string `json:"session_type" validate:"required"`
}

func (h *Controller) StartSession(c *fiber.Ctx) error {
	var req StartSessionRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid request body")
	}

	user, ok := c.Locals("user").(*model.User)
	if !ok {
		return fiber.NewError(fiber.StatusUnauthorized, "Unauthorized")
	}

	db := h.service.(*service).repo.GetDB()
	session := model.ScanSession{
		SessionType: req.SessionType,
		UserID:      user.ID,
		Status:      "IN_PROGRESS",
	}

	if err := db.Create(&session).Error; err != nil {
		return err
	}

	return c.Status(fiber.StatusOK).JSON(response.SuccessWithData[interface{}]{
		Code:    fiber.StatusOK,
		Status:  "success",
		Message: "Scan session started",
		Data: fiber.Map{
			"session_id": session.ID,
		},
	})
}

type PutAwayRequest struct {
	BatchBarcode    string `json:"batch_barcode" validate:"required"`
	LocationBarcode string `json:"location_barcode" validate:"required"`
}

func (h *Controller) PutAway(c *fiber.Ctx) error {
	var req PutAwayRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid request body")
	}

	user, ok := c.Locals("user").(*model.User)
	if !ok {
		return fiber.NewError(fiber.StatusUnauthorized, "Unauthorized")
	}

	batch, err := h.service.PutAway(user.ID, req.BatchBarcode, req.LocationBarcode)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	return c.Status(fiber.StatusOK).JSON(response.SuccessWithData[interface{}]{
		Code:    fiber.StatusOK,
		Status:  "success",
		Message: "Put away completed successfully",
		Data:    batch,
	})
}
