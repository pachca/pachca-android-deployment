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

	"pachca.com/android-deployment/shared"
)

type GitlabPayload struct {
	Event  string          `json:"event"`
	Result string          `json:"result"`
	Data   json.RawMessage `json:"data"`
}

type GitlabBuildData struct {
	JobID       int    `json:"job_id"`
	VersionCode int    `json:"version_code"`
	VersionName string `json:"version_name"`
}

type Config struct {
	PachcaBaseURL string
	PachcaAPIKey  string
	ChatID        int
}

type PachcaMessageRequest struct {
	Message struct {
		EntityType string           `json:"entity_type"`
		EntityID   int              `json:"entity_id"`
		Content    string           `json:"content"`
		Buttons    [][]PachcaButton `json:"buttons"`
	} `json:"message"`
}

type PachcaButton struct {
	Text string `json:"text"`
	Data string `json:"data"`
}

type PachcaMessageResponse struct {
	Data struct {
		ID int `json:"id"`
	} `json:"data"`
}

func Handler(w http.ResponseWriter, r *http.Request) {
	HandleGitlabHook(w, r, http.DefaultClient)
}

func HandleGitlabHook(w http.ResponseWriter, r *http.Request, client *http.Client) {
	config, err := NewConfig()
	if err != nil {
		log.Printf("Config error: %s", err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	bodyBytes, _ := io.ReadAll(r.Body)
	log.Printf("Incoming Gitlab payload: %s", string(bodyBytes))

	var payload GitlabPayload
	if err := json.Unmarshal(bodyBytes, &payload); err != nil {
		http.Error(w, "Invalid payload", http.StatusBadRequest)
		return
	}

	if payload.Event != "build" || payload.Result != "success" {
		w.WriteHeader(http.StatusOK)
		return
	}

	err = HandleGitlabBuildSuccess(r.Context(), client, config, payload.Data)
	if err != nil {
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

	chatIDStr := os.Getenv(shared.EnvPachcaInternalChatId)
	if chatIDStr == "" {
		return nil, fmt.Errorf("ENV_PACHCA_INTERNAL_CHAT_ID not set")
	}

	chatID, err := strconv.Atoi(chatIDStr)
	if err != nil {
		return nil, fmt.Errorf("invalid ENV_PACHCA_INTERNAL_CHAT_ID")
	}

	return &Config{
		PachcaBaseURL: pachcaBaseURL,
		PachcaAPIKey:  pachcaAPIKey,
		ChatID:        chatID,
	}, nil
}

func HandleGitlabBuildSuccess(ctx context.Context, client *http.Client, config *Config, data json.RawMessage) error {
	var buildData GitlabBuildData
	if err := json.Unmarshal(data, &buildData); err != nil {
		return err
	}

	content := fmt.Sprintf(
		"Release %s (%d) uploaded to Google Play Internal. Built by job %d.",
		buildData.VersionName, buildData.VersionCode, buildData.JobID,
	)

	button := []PachcaButton{
		{
			Text: "Promote release",
			Data: fmt.Sprintf("promote:%d", buildData.JobID),
		},
	}
	buttons := [][]PachcaButton{button}

	var messageReq PachcaMessageRequest
	messageReq.Message.EntityType = "discussion"
	messageReq.Message.EntityID = config.ChatID
	messageReq.Message.Content = content
	messageReq.Message.Buttons = buttons

	messageID, err := sendMessage(ctx, client, config, messageReq)
	if err != nil {
		return err
	}

	if err := pinMessage(ctx, client, config, messageID); err != nil {
		return err
	}

	return nil
}

func sendMessage(ctx context.Context, client *http.Client, config *Config, messageReq PachcaMessageRequest) (int, error) {
	payloadBytes, err := json.Marshal(messageReq)
	if err != nil {
		return 0, err
	}

	log.Printf("Outgoing Pachca payload: %s", string(payloadBytes))

	req, err := http.NewRequestWithContext(ctx, "POST", config.PachcaBaseURL+"/messages", bytes.NewReader(payloadBytes))
	if err != nil {
		return 0, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+config.PachcaAPIKey)

	resp, err := client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	log.Printf("Pachca response: %s", string(respBody))

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("Pachca API returned status %d", resp.StatusCode)
	}

	var messageResp PachcaMessageResponse
	if err := json.Unmarshal(respBody, &messageResp); err != nil {
		return 0, err
	}

	return messageResp.Data.ID, nil
}

func pinMessage(ctx context.Context, client *http.Client, config *Config, messageID int) error {
	url := fmt.Sprintf("%s/messages/%d/pin", config.PachcaBaseURL, messageID)
	req, err := http.NewRequestWithContext(ctx, "POST", url, nil)
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", "Bearer "+config.PachcaAPIKey)

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	log.Printf("Pachca pin response: %s", string(respBody))

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Pachca pin API returned status %d", resp.StatusCode)
	}

	return nil
}
