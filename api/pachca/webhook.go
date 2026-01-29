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

type PachcaViewSubmitPayload struct {
	Type             string         `json:"type"`
	Event            string         `json:"event"`
	PrivateMetadata  string         `json:"private_metadata"`
	CallbackID       string         `json:"callback_id"`
	UserID           int            `json:"user_id"`
	Data             map[string]any `json:"data"`
	WebhookTimestamp int            `json:"webhook_timestamp"`
}

type PachcaViewRequest struct {
	Type            string `json:"type"`
	TriggerID       string `json:"trigger_id"`
	CallbackID      string `json:"callback_id"`
	PrivateMetadata string `json:"private_metadata"`
	View            View   `json:"view"`
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

type FormValidationErrorsResponse struct {
	Errors map[string]string `json:"errors"`
}

type PromoteFormData struct {
	RolloutPercentage int
	ReleaseNotes      string
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

	var basePayload struct {
		Type  string `json:"type"`
		Event string `json:"event"`
	}
	if err := json.Unmarshal(bodyBytes, &basePayload); err != nil {
		http.Error(w, "Invalid payload", http.StatusBadRequest)
		return
	}

	switch basePayload.Type {
	case "button":
		if basePayload.Event == "click" {
			handleButtonClick(w, r, client, config, bodyBytes)
		}
	case "view":
		if basePayload.Event == "submit" {
			handleViewSubmit(w, r, client, config, bodyBytes)
		}
	default:
		w.WriteHeader(http.StatusOK)
	}
}

func handleButtonClick(w http.ResponseWriter, r *http.Request, client *http.Client, config *Config, bodyBytes []byte) {
	var payload PachcaButtonWebhookPayload
	if err := json.Unmarshal(bodyBytes, &payload); err != nil {
		http.Error(w, "Invalid payload", http.StatusBadRequest)
		return
	}

	releaseInfo, err := parseButtonData(payload.Data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if releaseInfo.JobID == 0 {
		w.WriteHeader(http.StatusOK)
		return
	}

	err = openPromoteForm(r.Context(), client, config, payload.TriggerID, releaseInfo)
	if err != nil {
		log.Printf("Error opening promote form: %s", err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func handleViewSubmit(w http.ResponseWriter, r *http.Request, client *http.Client, config *Config, bodyBytes []byte) {
	var payload PachcaViewSubmitPayload
	if err := json.Unmarshal(bodyBytes, &payload); err != nil {
		http.Error(w, "Invalid payload", http.StatusBadRequest)
		return
	}

	if payload.CallbackID != "promote" {
		w.WriteHeader(http.StatusOK)
		return
	}

	var releaseInfo shared.ReleaseInfo
	if err := json.Unmarshal([]byte(payload.PrivateMetadata), &releaseInfo); err != nil {
		http.Error(w, "Invalid private_metadata", http.StatusBadRequest)
		return
	}

	errors := validatePromoteForm(payload.Data)
	if len(errors) > 0 {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(FormValidationErrorsResponse{Errors: errors})
		return
	}

	var formData PromoteFormData
	rolloutStr := payload.Data["rollout_percentage"].(string)
	formData.RolloutPercentage, _ = strconv.Atoi(rolloutStr)
	formData.ReleaseNotes = payload.Data["release_notes"].(string)

	log.Printf("Promote form submitted: job=%d, version=%s (%d), rollout=%d%%, notes=%s",
		releaseInfo.JobID, releaseInfo.VersionName, releaseInfo.VersionCode,
		formData.RolloutPercentage, formData.ReleaseNotes)

	w.WriteHeader(http.StatusOK)
}

func validatePromoteForm(data map[string]any) map[string]string {
	errors := make(map[string]string)

	rolloutStr, ok := data["rollout_percentage"].(string)
	if !ok || rolloutStr == "" {
		errors["rollout_percentage"] = "Rollout percentage is required"
	} else {
		rollout, err := strconv.Atoi(rolloutStr)
		if err != nil {
			errors["rollout_percentage"] = "Rollout percentage must be a number"
		} else if rollout < 0 || rollout > 100 {
			errors["rollout_percentage"] = "Rollout percentage must be between 0 and 100"
		}
	}

	notes, ok := data["release_notes"].(string)
	if !ok || notes == "" {
		errors["release_notes"] = "Release notes are required"
	} else if len(notes) > 500 {
		errors["release_notes"] = "Release notes must be 500 characters or less"
	}

	return errors
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

func parseButtonData(data string) (*shared.ReleaseInfo, error) {
	parts := strings.SplitN(data, "|", 2)
	if len(parts) != 2 || parts[0] != "promote" {
		return nil, fmt.Errorf("invalid button data format")
	}

	var releaseInfo shared.ReleaseInfo
	if err := json.Unmarshal([]byte(parts[1]), &releaseInfo); err != nil {
		return nil, fmt.Errorf("invalid button data json")
	}

	return &releaseInfo, nil
}

func openPromoteForm(ctx context.Context, client *http.Client, config *Config, triggerID string, releaseInfo *shared.ReleaseInfo) error {
	privateMetadata, _ := json.Marshal(releaseInfo)

	viewReq := PachcaViewRequest{
		Type:            "modal",
		TriggerID:       triggerID,
		CallbackID:      "promote",
		PrivateMetadata: string(privateMetadata),
		View: View{
			Title: "Promote Release",
			Blocks: []ViewBlock{
				{
					Type: "header",
					Text: fmt.Sprintf("Promote %s (%d) from job %d", releaseInfo.VersionName, releaseInfo.VersionCode, releaseInfo.JobID),
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
