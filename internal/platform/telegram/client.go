package telegram

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

type Client struct {
	token   string
	apiURL  string
	httpCli *http.Client
}

func NewClient(token string) *Client {
	return &Client{
		token:  token,
		apiURL: "https://api.telegram.org/bot" + token,
		httpCli: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

type SendMessageParams struct {
	ChatID    int64
	Text      string
	ParseMode string
}

func (c *Client) SendMessage(ctx context.Context, params SendMessageParams) (int64, error) {
	v := url.Values{}
	v.Set("chat_id", strconv.FormatInt(params.ChatID, 10))
	v.Set("text", params.Text)
	if params.ParseMode != "" {
		v.Set("parse_mode", params.ParseMode)
	}

	u := c.apiURL + "/sendMessage?" + v.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u, nil)
	if err != nil {
		return 0, fmt.Errorf("create request: %w", err)
	}

	resp, err := c.httpCli.Do(req)
	if err != nil {
		return 0, fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	var tgResp struct {
		OK          bool   `json:"ok"`
		Description string `json:"description"`
		Result      struct {
			MessageID int64 `json:"message_id"`
		} `json:"result"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&tgResp); err != nil {
		return 0, fmt.Errorf("decode response: %w", err)
	}

	if !tgResp.OK {
		return 0, fmt.Errorf("telegram error: %s", tgResp.Description)
	}

	return tgResp.Result.MessageID, nil
}
