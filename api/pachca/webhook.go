package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	"pachca.com/android-deployment/shared"
)

type PachcaButtonWebhookPayload struct {
	Type             string `json:"type"`
	Event            string `json:"event"`
	MessageID        int    `json:"message_id"`
	TriggerID        string `json:"trigger_id"`
	Data             string `json:"data"`
	UserID           int    `json:"user_id"`
	ChatID           int    `json:"chat_id"`
	WebhookTimestamp int    `json:"webhook_timestamp"`
}

type ButtonAction struct {
	Action string
	JobID  int
}

type PachcaViewRequest struct {
	TriggerID string `json:"trigger_id"`
	View      View   `json:"view"`
}

type View struct {
	Title  string      `json:"title"`
	Blocks []ViewBlock `json:"blocks"`
}

type ViewBlock struct {
	Type        string `json:"type"`
	Name        string `json:"name,omitempty"`
	Label       string `json:"label,omitempty"`
	Text        string `json:"text,omitempty"`
	Placeholder string `json:"placeholder,omitempty"`
	Multiline   bool   `json:"multiline,omitempty"`
	MinLength   int    `json:"min_length,omitempty"`
	MaxLength   int    `json:"max_length,omitempty"`
	Required    bool   `json:"required,omitempty"`
	Hint        string `json:"hint,omitempty"`
}

type Config struct {
	PachcaBaseURL string
	PachcaAPIKey  string
}

func Handler(w http.ResponseWriter, r *http.Request) {
	HandlePachcaHook(w, r, http.DefaultClient)
}

func HandlePachcaHook(w http.ResponseWriter, r *http.Request, client *http.Client) {
	config, err := NewConfig()
	if err != nil {
		log.Printf("Config error: %s", err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	bodyBytes, _ := io.ReadAll(r.Body)
	log.Printf("Incoming Pachca payload: %s", string(bodyBytes))

	var payload PachcaButtonWebhookPayload
	if err := json.Unmarshal(bodyBytes, &payload); err != nil {
		http.Error(w, "Invalid payload", http.StatusBadRequest)
		return
	}

	if payload.Type != "button" || payload.Event != "click" {
		w.WriteHeader(http.StatusOK)
		return
	}

	action, err := parseButtonData(payload.Data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if action.Action != "promote" {
		w.WriteHeader(http.StatusOK)
		return
	}

	err = openPromoteForm(r.Context(), client, config, payload.TriggerID, action.JobID)
	if err != nil {
		log.Printf("Error opening promote form: %s", err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func NewConfig() (*Config, error) {
	pachcaBaseURL := os.Getenv(shared.EnvPachcaUrl)
	if pachcaBaseURL == "" {
		return nil, fmt.Errorf("ENV_PACHCA_URL not set")
	}

	pachcaAPIKey := os.Getenv(shared.EnvPachcaKey)
	if pachcaAPIKey == "" {
		return nil, fmt.Errorf("ENV_PACHCA_KEY not set")
	}

	return &Config{
		PachcaBaseURL: pachcaBaseURL,
		PachcaAPIKey:  pachcaAPIKey,
	}, nil
}

func parseButtonData(data string) (*ButtonAction, error) {
	parts := strings.SplitN(data, ":", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid button data format")
	}

	jobID, err := strconv.Atoi(parts[1])
	if err != nil {
		return nil, fmt.Errorf("invalid job ID")
	}

	return &ButtonAction{
		Action: parts[0],
		JobID:  jobID,
	}, nil
}

func openPromoteForm(ctx context.Context, client *http.Client, config *Config, triggerID string, jobID int) error {
	viewReq := PachcaViewRequest{
		TriggerID: triggerID,
		View: View{
			Title: "Promote Release",
			Blocks: []ViewBlock{
				{
					Type: "header",
					Text: fmt.Sprintf("Promote release from job %d", jobID),
				},
				{
					Type:        "input",
					Name:        "rollout_percentage",
					Label:       "Rollout percentage",
					Placeholder: "Enter percentage (0-100)",
					MinLength:   1,
					MaxLength:   3,
					Required:    true,
					Hint:        "Percentage of users who will receive this update (0-100)",
				},
				{
					Type:        "input",
					Name:        "release_notes",
					Label:       "Release notes",
					Placeholder: "Enter release notes",
					Multiline:   true,
					MaxLength:   500,
					Required:    true,
				},
			},
		},
	}

	return sendView(ctx, client, config, viewReq)
}

func sendView(ctx context.Context, client *http.Client, config *Config, viewReq PachcaViewRequest) error {
	payloadBytes, err := json.Marshal(viewReq)
	if err != nil {
		return err
	}

	log.Printf("Outgoing Pachca view payload: %s", string(payloadBytes))

	req, err := http.NewRequestWithContext(ctx, "POST", config.PachcaBaseURL+"/views/open", bytes.NewReader(payloadBytes))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+config.PachcaAPIKey)

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	log.Printf("Pachca view response: %s", string(respBody))

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("Pachca view API returned status %d", resp.StatusCode)
	}

	return nil
}
