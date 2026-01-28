package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"pachca.com/android-deployment/shared"
)

func TestGitlabNotifiesGooglePlayReleaseIsSuccessful(t *testing.T) {
	var pachcaCalls atomic.Int32

	gitlabPayload := map[string]any{
		"event":  "build",
		"result": "success",
		"data": map[string]any{
			"job_id":       12345,
			"version_code": 1001,
			"version_name": "1.0.1",
		},
	}
	payloadBytes, _ := json.Marshal(gitlabPayload)

	mockPachca := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		pachcaCalls.Add(1)

		if r.URL.Path != "/messages" {
			t.Errorf("Expected path /messages, got %s", r.URL.Path)
		}
		if r.Method != "POST" {
			t.Errorf("Expected POST method, got %s", r.Method)
		}
		var msg struct {
			Message struct {
				EntityType string `json:"entity_type"`
				EntityID   int    `json:"entity_id"`
				Content    string `json:"content"`
				Buttons    [][]struct {
					Text string `json:"text"`
					URL  string `json:"url"`
					Data string `json:"data"`
				} `json:"buttons"`
			} `json:"message"`
		}
		json.NewDecoder(r.Body).Decode(&msg)
		if msg.Message.EntityType != "discussion" {
			t.Errorf("Expected entity_type 'discussion', got '%s'", msg.Message.EntityType)
		}
		if msg.Message.EntityID != 198 {
			t.Errorf("Expected entity_id 198, got %d", msg.Message.EntityID)
		}
		if msg.Message.Content == "" {
			t.Error("Expected content to be non-empty")
		}
		if len(msg.Message.Buttons) == 0 {
			t.Error("Expected buttons to be present")
		}
		if len(msg.Message.Buttons[0]) == 0 {
			t.Error("Expected at least one button")
		}
		if msg.Message.Buttons[0][0].Text != "Promote release" {
			t.Errorf("Expected button text 'Promote release', got '%s'", msg.Message.Buttons[0][0].Text)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{
				"id":                 194275,
				"entity_type":        "discussion",
				"entity_id":          198,
				"chat_id":            198,
				"content":            msg.Message.Content,
				"user_id":            12,
				"created_at":         "2020-06-08T09:32:57.000Z",
				"url":                "https://app.pachca.com/chats/198?message=194275",
				"files":              []any{},
				"buttons":            msg.Message.Buttons,
				"thread":             nil,
				"forwarding":         nil,
				"parent_message_id":  nil,
				"display_avatar_url": nil,
				"display_name":       nil,
			},
		})
	}))
	defer mockPachca.Close()

	t.Setenv(shared.EnvPachcaUrl, mockPachca.URL)
	t.Setenv(shared.EnvPachcaKey, "test-api-key")
	t.Setenv(shared.EnvPachcaInternalChatId, "198")

	req := httptest.NewRequest("POST", "/gitlab/webhook", bytes.NewReader(payloadBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	HandleGitlabHook(w, req, mockPachca.Client())

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
	if pachcaCalls.Load() != 1 {
		t.Errorf("Expected 1 call to Pachca API, got %d", pachcaCalls.Load())
	}
}

func TestGitlabNotifiesGooglePlayBuildFailed(t *testing.T) {

}

func TestGitlabNotifiesPromotionIsSuccessful(t *testing.T) {

}

func TestGitlabNotifiesPromotionFailed(t *testing.T) {

}

func TestGitlabNotifiesRolloutUpdateIsSuccessful(t *testing.T) {

}

func TestGitlabNotifiesRolloutUpdateFailed(t *testing.T) {

}

func TestGitlabNotifiesOtherStoresReleaseIsSuccessful(t *testing.T) {

}

func TestGitlabNotifiesOtherStoresReleaseFailed(t *testing.T) {

}
