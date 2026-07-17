package controller

import (
	"app/src/response"
	"app/src/service"
	"strconv"

	"github.com/gofiber/fiber/v2"
)

type ReportController struct {
	ReportService service.ReportService
}

func NewReportController(reportService service.ReportService) *ReportController {
	return &ReportController{
		ReportService: reportService,
	}
}

func (ctrl *ReportController) GetInventoryValueReport(c *fiber.Ctx) error {
	categoryID := c.Query("category_id", "")
	warehouseID := c.Query("warehouse_id", "")

	data, err := ctrl.ReportService.GetInventoryValueReport(c, categoryID, warehouseID)
	if err != nil {
		return err
	}

	return c.Status(fiber.StatusOK).
		JSON(response.SuccessWithData[interface{}]{
			Code:    fiber.StatusOK,
			Status:  "success",
			Message: "Inventory value report retrieved successfully",
			Data:    data,
		})
}

func (ctrl *ReportController) GetStockAgingReport(c *fiber.Ctx) error {
	categoryID := c.Query("category_id", "")
	warehouseID := c.Query("warehouse_id", "")
	productID := c.Query("product_id", "")

	data, err := ctrl.ReportService.GetStockAgingReport(c, categoryID, warehouseID, productID)
	if err != nil {
		return err
	}

	return c.Status(fiber.StatusOK).
		JSON(response.SuccessWithData[interface{}]{
			Code:    fiber.StatusOK,
			Status:  "success",
			Message: "Stock aging report retrieved successfully",
			Data:    data,
		})
}

func (ctrl *ReportController) GetStockMutationReport(c *fiber.Ctx) error {
	startDate := c.Query("start_date", "")
	endDate := c.Query("end_date", "")
	productID := c.Query("product_id", "")
	warehouseID := c.Query("warehouse_id", "")

	data, err := ctrl.ReportService.GetStockMutationReport(c, startDate, endDate, productID, warehouseID)
	if err != nil {
		return err
	}

	return c.Status(fiber.StatusOK).
		JSON(response.SuccessWithData[interface{}]{
			Code:    fiber.StatusOK,
			Status:  "success",
			Message: "Stock mutation report retrieved successfully",
			Data:    data,
		})
}

func (ctrl *ReportController) GetDistributionReport(c *fiber.Ctx) error {
	categoryID := c.Query("category_id", "")
	subCategory := c.Query("sub_category", "")
	startDate := c.Query("start_date", "")
	endDate := c.Query("end_date", "")

	data, err := ctrl.ReportService.GetDistributionReport(c, categoryID, subCategory, startDate, endDate)
	if err != nil {
		return err
	}

	return c.Status(fiber.StatusOK).
		JSON(response.SuccessWithData[interface{}]{
			Code:    fiber.StatusOK,
			Status:  "success",
			Message: "Distribution report retrieved successfully",
			Data:    data,
		})
}

func (ctrl *ReportController) GetReorderPointReport(c *fiber.Ctx) error {
	leadTimeStr := c.Query("lead_time", "7")
	leadTime, err := strconv.Atoi(leadTimeStr)
	if err != nil || leadTime < 0 {
		leadTime = 7
	}

	data, err := ctrl.ReportService.GetReorderPointReport(c, leadTime)
	if err != nil {
		return err
	}

	return c.Status(fiber.StatusOK).
		JSON(response.SuccessWithData[interface{}]{
			Code:    fiber.StatusOK,
			Status:  "success",
			Message: "Reorder point report retrieved successfully",
			Data:    data,
		})
}
