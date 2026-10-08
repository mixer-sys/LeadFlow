package bitrix24

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type Client struct {
	webhookURL string
	httpClient *http.Client
}

type CreateLeadParams struct {
	Title    string
	Name     string
	LastName string
	Phone    string
	Email    string
	Comments string
}

type CreateLeadResponse struct {
	Result int64 `json:"result"`
}

func NewClient(webhookURL string) *Client {
	return &Client{
		webhookURL: webhookURL,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (c *Client) CreateLead(ctx context.Context, params CreateLeadParams) (int64, error) {
	url := c.webhookURL + "crm.lead.add.json"

	payload := map[string]interface{}{
		"fields": map[string]interface{}{
			"TITLE":     params.Title,
			"NAME":      params.Name,
			"LAST_NAME": params.LastName,
			"COMMENTS":  params.Comments,
		},
	}

	if params.Phone != "" {
		payload["fields"].(map[string]interface{})["PHONE"] = []map[string]string{
			{"VALUE": params.Phone, "VALUE_TYPE": "wORK"},
		}
	}

	if params.Email != "" {
		payload["fields"].(map[string]interface{})["EMAIL"] = []map[string]string{
			{"VALUE": params.Email, "VALUE_TYPE": "WORK"},
		}
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return 0, fmt.Errorf("marshal payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(body))
	if err != nil {
		return 0, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return 0, fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("unexpected status: %s", resp.Status)
	}

	var result CreateLeadResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return 0, fmt.Errorf("decode response: %w", err)
	}

	return result.Result, nil
}
