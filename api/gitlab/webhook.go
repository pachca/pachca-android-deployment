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
	URL  string `json:"url"`
	Data string `json:"data"`
}

type PachcaMessageResponse struct {
	Data struct {
		ID               int     `json:"id"`
		EntityType       string  `json:"entity_type"`
		EntityID         int     `json:"entity_id"`
		ChatID           int     `json:"chat_id"`
		Content          string  `json:"content"`
		UserID           int     `json:"user_id"`
		CreatedAt        string  `json:"created_at"`
		URL              string  `json:"url"`
		Files            []any   `json:"files"`
		Buttons          []any   `json:"buttons"`
		Thread           *any    `json:"thread"`
		Forwarding       *any    `json:"forwarding"`
		ParentMessageID  *int    `json:"parent_message_id"`
		DisplayAvatarURL *string `json:"display_avatar_url"`
		DisplayName      *string `json:"display_name"`
	} `json:"data"`
}

func Handler(w http.ResponseWriter, r *http.Request) {
	HandleGitlabHook(w, r, http.DefaultClient)
}

func HandleGitlabHook(w http.ResponseWriter, r *http.Request, client *http.Client) {
	config, err := NewConfig()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var payload GitlabPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "Invalid payload", http.StatusBadRequest)
		return
	}

	if payload.Event != "build" || payload.Result != "success" {
		w.WriteHeader(http.StatusOK)
		return
	}

	messageID, err := HandleGitlabBuildSuccess(r.Context(), client, config, payload.Data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"message_id": messageID,
	}
	json.NewEncoder(w).Encode(response)
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

func HandleGitlabBuildSuccess(ctx context.Context, client *http.Client, config *Config, data json.RawMessage) (int, error) {
	var buildData GitlabBuildData
	if err := json.Unmarshal(data, &buildData); err != nil {
		return 0, err
	}

	log.Printf("Build data: job_id=%d, version_code=%d, version_name=%s", buildData.JobID, buildData.VersionCode, buildData.VersionName)

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

	payloadBytes, err := json.Marshal(messageReq)
	if err != nil {
		return 0, err
	}

	log.Printf("Sending to Pachca: %s", string(payloadBytes))

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

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return 0, fmt.Errorf("Pachca API returned status %d: %s", resp.StatusCode, string(body))
	}

	var response PachcaMessageResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return 0, err
	}

	return response.Data.ID, nil
}
