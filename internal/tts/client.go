// Copyright (C) 2026 Joey Kot <joey.kot.x@gmail.com>
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed WITHOUT ANY WARRANTY; without even the
// implied warranty of MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.
// See <https://www.gnu.org/licenses/> for more details.

package tts

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Client calls an OpenAI-compatible text-to-speech endpoint.
type Client struct {
	BaseURL    string
	Token      string
	Model      string
	HTTPClient *http.Client
}

// New returns a TTS client with a request timeout.
func New(baseURL, token, model string, timeoutSeconds int) *Client {
	if timeoutSeconds <= 0 {
		timeoutSeconds = 120
	}
	return &Client{
		BaseURL: strings.TrimSpace(baseURL),
		Token:   strings.TrimSpace(token),
		Model:   strings.TrimSpace(model),
		HTTPClient: &http.Client{
			Timeout: time.Duration(timeoutSeconds) * time.Second,
		},
	}
}

// Synthesize sends text and voiceJSON to the configured endpoint and returns raw audio bytes.
func (c *Client) Synthesize(text string, voiceJSON string) ([]byte, error) {
	return c.SynthesizeContext(context.Background(), text, voiceJSON)
}

// SynthesizeContext is the context-aware form of Synthesize.
func (c *Client) SynthesizeContext(ctx context.Context, text string, voiceJSON string) ([]byte, error) {
	if c == nil {
		return nil, fmt.Errorf("tts client is nil")
	}
	if strings.TrimSpace(c.BaseURL) == "" {
		return nil, fmt.Errorf("tts base url is empty")
	}
	if strings.TrimSpace(c.Model) == "" {
		return nil, fmt.Errorf("tts model is empty")
	}
	text = strings.TrimSpace(text)
	if text == "" {
		return nil, fmt.Errorf("tts text is empty")
	}

	body := map[string]any{}
	if err := json.Unmarshal([]byte(voiceJSON), &body); err != nil {
		return nil, fmt.Errorf("voice json: %w", err)
	}
	body["model"] = c.Model
	body["input"] = text

	payload, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "audio/mpeg, audio/*, application/octet-stream")
	if c.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.Token)
	}

	httpClient := c.HTTPClient
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		limited, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return nil, fmt.Errorf("tts request failed: %s: %s", resp.Status, strings.TrimSpace(string(limited)))
	}
	audio, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if len(audio) == 0 {
		return nil, fmt.Errorf("tts response body is empty")
	}
	return audio, nil
}
