package main

import (
	"fmt"
	"log"
	"os"
	"strings"
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
	app := fiber.New(fiber.Config{DisableStartupMessage: true})

	app.Post("/v:version/:phoneID/messages", func(c *fiber.Ctx) error {
		var body map[string]interface{}
		if err := c.BodyParser(&body); err != nil {
			return c.Status(400).JSON(fiber.Map{"error": err.Error()})
		}
		to, _ := body["to"].(string)
		texto := textoSaida(body)
		chave := to
		if chave == "" {
			chave = "_global"
		}
		inbox.mu.Lock()
		inbox.mensagens[chave] = append(inbox.mensagens[chave], MensagemEnviada{
			ScenarioID: chave,
			Texto:      texto,
			Timestamp:  time.Now().Format(time.RFC3339),
			Para:       to,
		})
		inbox.mu.Unlock()
		log.Printf("[mock] para=%s → %.80s", to, texto)
		return c.Status(200).JSON(fiber.Map{
			"messaging_product": "whatsapp",
			"contacts":          []map[string]string{{"input": to, "wa_id": to}},
			"messages":          []map[string]string{{"id": fmt.Sprintf("wamid.mock.%d", time.Now().UnixNano())}},
		})
	})

	app.Get("/inbox", func(c *fiber.Ctx) error {
		sid := c.Query("phone")
		if sid == "" {
			sid = c.Query("scenario_id", "_global")
		}
		inbox.mu.Lock()
		msgs := append([]MensagemEnviada(nil), inbox.mensagens[sid]...)
		inbox.mu.Unlock()
		return c.JSON(fiber.Map{"scenario_id": sid, "total": len(msgs), "mensagens": msgs})
	})

	app.Delete("/inbox", func(c *fiber.Ctx) error {
		sid := c.Query("phone")
		if sid == "" {
			sid = c.Query("scenario_id", "_global")
		}
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

func textoSaida(body map[string]interface{}) string {
	var partes []string
	if text, ok := body["text"].(map[string]interface{}); ok {
		if b, ok := text["body"].(string); ok {
			partes = append(partes, b)
		}
	}
	if inter, ok := body["interactive"].(map[string]interface{}); ok {
		if bodyMap, ok := inter["body"].(map[string]interface{}); ok {
			if b, ok := bodyMap["text"].(string); ok {
				partes = append(partes, b)
			}
		}
		if action, ok := inter["action"].(map[string]interface{}); ok {
			if botoes, ok := action["buttons"].([]interface{}); ok {
				for _, raw := range botoes {
					btn, ok := raw.(map[string]interface{})
					if !ok {
						continue
					}
					reply, _ := btn["reply"].(map[string]interface{})
					if title, ok := reply["title"].(string); ok {
						partes = append(partes, title)
					}
				}
			}
		}
	}
	return strings.TrimSpace(strings.Join(partes, "\n"))
}
