package service

import (
	"app/src/model"
	"app/src/utils"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type ActivityService interface {
	LogActivity(userID uuid.UUID, action, module, targetID, description, ip string)
	GetActivityLogs(c *fiber.Ctx, search, module, action string, page, limit int) ([]model.ActivityLog, int64, error)
}

type activityService struct {
	Log *logrus.Logger
	DB  *gorm.DB
}

func NewActivityService(db *gorm.DB) ActivityService {
	return &activityService{
		Log: utils.Log,
		DB:  db,
	}
}

// LogActivity is a fire-and-forget helper. Errors are logged but not returned.
func (s *activityService) LogActivity(userID uuid.UUID, action, module, targetID, description, ip string) {
	entry := model.ActivityLog{
		UserID:      userID,
		Action:      action,
		Module:      module,
		TargetID:    targetID,
		Description: description,
		IPAddress:   ip,
	}
	if err := s.DB.Create(&entry).Error; err != nil {
		s.Log.Errorf("Failed to write activity log: %v", err)
	}
}

// GetActivityLogs returns paginated, filtered activity logs.
func (s *activityService) GetActivityLogs(c *fiber.Ctx, search, module, action string, page, limit int) ([]model.ActivityLog, int64, error) {
	var logs []model.ActivityLog
	var total int64

	query := s.DB.WithContext(c.Context()).Model(&model.ActivityLog{})

	if module != "" {
		query = query.Where("module = ?", module)
	}
	if action != "" {
		query = query.Where("action = ?", action)
	}
	if search != "" {
		query = query.Where("description LIKE ? OR target_id LIKE ? OR module LIKE ?", "%"+search+"%", "%"+search+"%", "%"+search+"%")
	}

	query.Count(&total)

	offset := (page - 1) * limit
	if err := query.
		Preload("User").
		Order("created_at DESC").
		Offset(offset).
		Limit(limit).
		Find(&logs).Error; err != nil {
		s.Log.Errorf("Failed to query activity logs: %v", err)
		return nil, 0, fiber.NewError(fiber.StatusInternalServerError, "Database error")
	}

	return logs, total, nil
}

// LogCtxActivity is a helper function to log activities using the Fiber context.
func LogCtxActivity(db *gorm.DB, c *fiber.Ctx, action, module, targetID, description string) {
	userObj := c.Locals("user")
	user, ok := userObj.(*model.User)
	if !ok || user == nil {
		return
	}
	log := model.ActivityLog{
		UserID:      user.ID,
		Action:      action,
		Module:      module,
		TargetID:    targetID,
		Description: description,
		IPAddress:   c.IP(),
	}
	if err := db.Create(&log).Error; err != nil {
		utils.Log.Errorf("Failed to log activity: %v", err)
	}
}

