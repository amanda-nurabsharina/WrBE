package middleware

import (
	"app/src/config"
	"app/src/database"
	"app/src/service"
	"app/src/utils"
	"encoding/json"
	"strings"

	"github.com/gofiber/fiber/v2"
)

func Auth(userService service.UserService, requiredRights ...string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		authHeader := c.Get("Authorization")
		token := strings.TrimSpace(strings.TrimPrefix(authHeader, "Bearer "))

		if token == "" {
			return fiber.NewError(fiber.StatusUnauthorized, "Please authenticate")
		}

		userID, err := utils.VerifyToken(token, config.JWTSecret, config.TokenTypeAccess)
		if err != nil {
			return fiber.NewError(fiber.StatusUnauthorized, "Please authenticate")
		}

		user, err := userService.GetUserByID(c, userID)
		if err != nil || user == nil {
			return fiber.NewError(fiber.StatusUnauthorized, "Please authenticate")
		}

		c.Locals("user", user)

		if len(requiredRights) > 0 {
			// Super admin has access to all resources
			if user.Role == "super_admin" || user.Role == "super admin" {
				return c.Next()
			}

			// Query role from database
			var accessibleMenusJSON []byte
			var userRights []string
			
			errDb := database.DB.Table("roles").
				Select("accessible_menus").
				Where("name = ? AND deleted_at IS NULL", user.Role).
				Row().Scan(&accessibleMenusJSON)

			if errDb == nil {
				json.Unmarshal(accessibleMenusJSON, &userRights)
			}

			// Always merge static configuration rights if they exist
			if staticRights, ok := config.RoleRights[user.Role]; ok {
				existingRights := make(map[string]bool)
				for _, r := range userRights {
					existingRights[r] = true
				}
				for _, r := range staticRights {
					if !existingRights[r] {
						userRights = append(userRights, r)
						existingRights[r] = true
					}
				}
			}

			if !hasAllRights(userRights, requiredRights) && c.Params("userId") != userID {
				return fiber.NewError(fiber.StatusForbidden, "You don't have permission to access this resource")
			}
		}

		return c.Next()
	}
}

func hasAllRights(userRights, requiredRights []string) bool {
	rightSet := make(map[string]struct{}, len(userRights))
	for _, right := range userRights {
		rightSet[right] = struct{}{}
	}

	for _, right := range requiredRights {
		if _, exists := rightSet[right]; !exists {
			return false
		}
	}
	return true
}
