package handlers

import (
	"fmt"
	"time"

	"github.com/DaffaFA/counter-user_access_control/pkg/entities"
	"github.com/DaffaFA/counter-user_access_control/pkg/user"
	"github.com/DaffaFA/counter-user_access_control/utils"
	"github.com/gofiber/fiber/v2"
)

const SESSION_KEY = "x-m-session"

func GetUser(service user.Service) fiber.Handler {
	return func(c *fiber.Ctx) error {
		ctx, span := utils.Tracer.Start(c.UserContext(), fmt.Sprintf("%s %s", c.Method(), c.OriginalURL()))
		defer span.End()

		session := c.Cookies(SESSION_KEY, "-1")
		if session == "-1" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Unauthorized",
			})
		}

		user, err := service.FetchUserSession(ctx, session)
		if err != nil {
			span.RecordError(err)
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": err.Error(),
			})
		}

		return c.JSON(user)
	}
}

func SignIn(service user.Service) fiber.Handler {
	return func(c *fiber.Ctx) error {
		ctx, span := utils.Tracer.Start(c.UserContext(), fmt.Sprintf("%s %s", c.Method(), c.OriginalURL()))
		defer span.End()

		var user entities.User

		if err := c.BodyParser(&user); err != nil {
			span.RecordError(err)
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": err.Error(),
			})
		}

		session, userData, sessionExpired, err := service.SignIn(ctx, user)
		if err != nil {
			span.RecordError(err)
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": err.Error(),
			})
		}

		c.Cookie(&fiber.Cookie{
			Name:     SESSION_KEY,
			Value:    session,
			Expires:  time.Now().Add(sessionExpired),
			HTTPOnly: true,
			SameSite: "Strict",
			Domain:   "",
			Path:     "/",
		})

		return c.JSON(userData)
	}
}

func Register(service user.Service) fiber.Handler {
	return func(c *fiber.Ctx) error {
		ctx, span := utils.Tracer.Start(c.UserContext(), fmt.Sprintf("%s %s", c.Method(), c.OriginalURL()))
		defer span.End()

		var user entities.User

		if err := c.BodyParser(&user); err != nil {
			span.RecordError(err)
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": err.Error(),
			})
		}

		if err := service.Register(ctx, user); err != nil {
			span.RecordError(err)
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": err.Error(),
			})
		}

		return c.Status(fiber.StatusCreated).JSON(fiber.Map{
			"message": "User created successfully",
		})
	}
}

func SignOut(service user.Service) fiber.Handler {
	return func(c *fiber.Ctx) error {
		ctx, span := utils.Tracer.Start(c.UserContext(), fmt.Sprintf("%s %s", c.Method(), c.OriginalURL()))
		defer span.End()

		session := c.Cookies(SESSION_KEY)

		if err := service.SignOut(ctx, session); err != nil {
			span.RecordError(err)
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": err.Error(),
			})
		}

		c.ClearCookie(SESSION_KEY)

		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"message": "User signed out successfully",
		})
	}
}
