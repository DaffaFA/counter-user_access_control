package routes

import (
	"github.com/DaffaFA/counter-server/api/handlers"
	"github.com/DaffaFA/counter-server/pkg/user"
	"github.com/gofiber/fiber/v2"
)

func UserRouter(app fiber.Router, service user.Service) {
	app.Get("/user", handlers.GetUser(service))
	app.Post("/signin", handlers.SignIn(service))
	app.Post("/register", handlers.Register(service))
	app.Post("/logout", handlers.SignOut(service))

	app.Post("/_auth", handlers.AuthRequestHandler(service))
}
