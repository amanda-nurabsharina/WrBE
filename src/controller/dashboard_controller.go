package controller

import (
	"app/src/response"
	"app/src/service"

	"github.com/gofiber/fiber/v2"
)

type DashboardController struct {
	DashboardService service.DashboardService
}

func NewDashboardController(dashboardService service.DashboardService) *DashboardController {
	return &DashboardController{
		DashboardService: dashboardService,
	}
}

func (ctrl *DashboardController) GetDashboardData(c *fiber.Ctx) error {
	data, err := ctrl.DashboardService.GetDashboardData(c)
	if err != nil {
		return err
	}

	return c.Status(fiber.StatusOK).
		JSON(response.SuccessWithData[map[string]interface{}]{
			Code:    fiber.StatusOK,
			Status:  "success",
			Message: "Dashboard data retrieved successfully",
			Data:    data,
		})
}
