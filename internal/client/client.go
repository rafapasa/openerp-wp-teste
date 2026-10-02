package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"rafapasa/openerp-wp-teste/internal/dto"
	"time"
)

type Client struct {
	cfg  *dto.Config
	http *http.Client
}

func New(cfg *dto.Config) *Client {
	return &Client{
		cfg:  cfg,
		http: &http.Client{Timeout: 30 * time.Second},
	}
}

func (c *Client) EnviarWebhook(scenarioID, from, texto string) (int, error) {
	body := map[string]interface{}{
		"object": "whatsapp_business_account",
		"entry": []map[string]interface{}{
			{
				"id": c.cfg.PhoneNumberID,
				"changes": []map[string]interface{}{
					{
						"field": "messages",
						"value": map[string]interface{}{
							"messaging_product": "whatsapp",
							"metadata": map[string]string{
								"display_phone_number": "15550000000",
								"phone_number_id":      c.cfg.PhoneNumberID,
							},
							"contacts": []map[string]interface{}{
								{
									"profile": map[string]string{"name": c.cfg.ClienteNome + " " + scenarioID},
									"wa_id":   from,
								},
							},
							"messages": []map[string]interface{}{
								{
									"from":      from,
									"id":        fmt.Sprintf("wamid.%s.%d", scenarioID, time.Now().UnixNano()),
									"timestamp": fmt.Sprintf("%d", time.Now().Unix()),
									"type":      "text",
									"text":      map[string]string{"body": texto},
								},
							},
						},
					},
				},
			},
		},
	}
	data, _ := json.Marshal(body)
	req, _ := http.NewRequest("POST", c.cfg.ServerURL+"/webhook", bytes.NewReader(data))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Scenario-ID", scenarioID)
	resp, err := c.http.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)
	return resp.StatusCode, nil
}

func (c *Client) BuscarInbox(phone string) (*dto.InboxResponse, error) {
	resp, err := c.http.Get(c.cfg.MockURL + "/inbox?phone=" + phone)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("mock retornou %d", resp.StatusCode)
	}
	data, _ := io.ReadAll(resp.Body)
	var out dto.InboxResponse
	if err := json.Unmarshal(data, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) LimparInbox(phone string) error {
	req, _ := http.NewRequest("DELETE", c.cfg.MockURL+"/inbox?phone="+phone, nil)
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	resp.Body.Close()
	return nil
}
