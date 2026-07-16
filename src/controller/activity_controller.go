package controller

import (
	"app/src/model"
	"app/src/response"
	"app/src/service"
	"strconv"

	"github.com/gofiber/fiber/v2"
)

type ActivityController struct {
	ActivityService service.ActivityService
}

func NewActivityController(activityService service.ActivityService) *ActivityController {
	return &ActivityController{
		ActivityService: activityService,
	}
}

func (ctrl *ActivityController) GetActivityLogs(c *fiber.Ctx) error {
	search := c.Query("search", "")
	module := c.Query("module", "")
	action := c.Query("action", "")
	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "50"))
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 200 {
		limit = 50
	}

	logs, total, err := ctrl.ActivityService.GetActivityLogs(c, search, module, action, page, limit)
	if err != nil {
		return err
	}

	return c.Status(fiber.StatusOK).JSON(response.SuccessWithData[map[string]interface{}]{
		Code:    fiber.StatusOK,
		Status:  "success",
		Message: "Activity logs retrieved successfully",
		Data: map[string]interface{}{
			"logs":  logs,
			"total": total,
			"page":  page,
			"limit": limit,
		},
	})
}

// Reusable empty response type for the SuccessWithData generic
type ActivityLogsResponse struct {
	Logs  []model.ActivityLog `json:"logs"`
	Total int64               `json:"total"`
	Page  int                 `json:"page"`
	Limit int                 `json:"limit"`
}
