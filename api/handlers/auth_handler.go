package handlers

import (
	"fmt"

	"github.com/DaffaFA/counter-user_access_control/pkg/user"
	"github.com/DaffaFA/counter-user_access_control/utils"
	"github.com/bytedance/sonic"
	"github.com/gofiber/fiber/v2"
)

func AuthRequestHandler(service user.Service) fiber.Handler {
	return func(c *fiber.Ctx) error {
		ctx, span := utils.Tracer.Start(c.UserContext(), fmt.Sprintf("%s %s", c.Method(), c.OriginalURL()))
		defer span.End()

		sessionId := c.Cookies(SESSION_KEY, "-1")
		if sessionId == "-1" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Unauthorized",
			})
		}

		user, err := service.FetchUserSession(ctx, sessionId)
		if err != nil {
			span.RecordError(err)
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Unauthorized",
			})
		}

		userByte, err := sonic.Marshal(user)
		if err != nil {
			span.RecordError(err)
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": err.Error(),
			})
		}

		c.Set("user-data", string(userByte))

		return nil
	}
}
