package api

import (
	"github.com/gofiber/fiber/v2"

	"github.com/georgeshao/ai-inference-dam/internal/dispatcher"
	"github.com/georgeshao/ai-inference-dam/internal/storage"
)

func SetupRoutes(app *fiber.App, store storage.Store, d *dispatcher.Dispatcher) {
	h := NewHandler(store, d)

	v1 := app.Group("/v1")

	v1.Post("/namespaces", h.CreateNamespace)
	v1.Get("/namespaces", h.ListNamespaces)
	v1.Get("/namespaces/:name", h.GetNamespace)
	v1.Patch("/namespaces/:name", h.UpdateNamespace)
	v1.Delete("/namespaces/:name", h.DeleteNamespace)

	v1.Post("/chat/completions", h.QueueChatCompletion)

	v1.Get("/requests", h.ListRequests)
	v1.Get("/requests/:id", h.GetRequest)

	v1.Post("/dispatch", h.TriggerDispatch)

	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok"})
	})
}
