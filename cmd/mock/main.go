package main

import (
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"
)

type MensagemEnviada struct {
	ScenarioID string `json:"scenario_id"`
	Texto      string `json:"texto"`
	Timestamp  string `json:"timestamp"`
	Para       string `json:"para"`
}

type Inbox struct {
	mu        sync.Mutex
	mensagens map[string][]MensagemEnviada
}

func main() {
	port := os.Getenv("MOCK_PORT")
	if port == "" {
		port = "9000"
	}

	inbox := &Inbox{mensagens: make(map[string][]MensagemEnviada)}

	app := fiber.New(fiber.Config{
		DisableStartupMessage: true,
	})

	app.Post("/v:version/:phoneID/messages", func(c *fiber.Ctx) error {
		var body struct {
			MessagingProduct string `json:"messaging_product"`
			To               string `json:"to"`
			Type             string `json:"type"`
			Text             struct {
				Body string `json:"body"`
			} `json:"text"`
		}
		if err := c.BodyParser(&body); err != nil {
			return c.Status(400).JSON(fiber.Map{"error": err.Error()})
		}

		scenarioID := c.Get("X-Scenario-ID")
		if scenarioID == "" {
			scenarioID = "_global"
		}

		inbox.mu.Lock()
		inbox.mensagens[scenarioID] = append(inbox.mensagens[scenarioID], MensagemEnviada{
			ScenarioID: scenarioID,
			Texto:      body.Text.Body,
			Timestamp:  time.Now().Format(time.RFC3339),
			Para:       body.To,
		})
		inbox.mu.Unlock()

		log.Printf("[mock] scenario=%s → %.80s", scenarioID, body.Text.Body)

		return c.Status(200).JSON(fiber.Map{
			"messaging_product": "whatsapp",
			"contacts":          []map[string]string{{"input": body.To, "wa_id": body.To}},
			"messages":          []map[string]string{{"id": fmt.Sprintf("wamid.mock.%d", time.Now().UnixNano())}},
		})
	})

	app.Get("/inbox", func(c *fiber.Ctx) error {
		sid := c.Query("scenario_id", "_global")
		inbox.mu.Lock()
		msgs := append([]MensagemEnviada(nil), inbox.mensagens[sid]...)
		inbox.mu.Unlock()
		return c.JSON(fiber.Map{
			"scenario_id": sid,
			"total":       len(msgs),
			"mensagens":   msgs,
		})
	})

	app.Delete("/inbox", func(c *fiber.Ctx) error {
		sid := c.Query("scenario_id", "_global")
		inbox.mu.Lock()
		delete(inbox.mensagens, sid)
		inbox.mu.Unlock()
		return c.JSON(fiber.Map{"ok": true})
	})

	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok"})
	})

	log.Printf("[mock] ouvindo em :%s", port)
	log.Fatal(app.Listen("0.0.0.0:" + port))
}
