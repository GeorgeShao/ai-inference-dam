package api

import (
	"github.com/gofiber/fiber/v2"

	"github.com/georgeshao/ai-inference-dam/internal/dispatcher"
	"github.com/georgeshao/ai-inference-dam/internal/storage"
)

func SetupRoutes(app *fiber.App, store storage.Store, d *dispatcher.Dispatcher) {
	h := NewHandler(store, d)

	app.Post("/namespaces", h.CreateNamespace)
	app.Get("/namespaces", h.ListNamespaces)
	app.Get("/namespaces/:name", h.GetNamespace)
	app.Patch("/namespaces/:name", h.UpdateNamespace)
	app.Delete("/namespaces/:name", h.DeleteNamespace)

	app.Get("/requests", h.ListRequests)
	app.Get("/requests/:id", h.GetRequest)

	app.Post("/dispatch", h.TriggerDispatch)

	app.Post("/v1/chat/completions", h.QueueChatCompletion)

	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok"})
	})
}
