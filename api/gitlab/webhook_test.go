package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"pachca.com/android-deployment/shared"
)

func TestGitlabNotifiesGooglePlayReleaseIsSuccessful(t *testing.T) {
	var messageCalls atomic.Int32
	var pinCalls atomic.Int32

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
		switch r.URL.Path {
		case "/messages":
			messageCalls.Add(1)

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
			buttonData := msg.Message.Buttons[0][0].Data
			if !strings.HasPrefix(buttonData, "promote|") {
				t.Errorf("Expected button data to start with 'promote|', got '%s'", buttonData)
			}
			var releaseInfo shared.ReleaseInfo
			if err := json.Unmarshal([]byte(strings.TrimPrefix(buttonData, "promote|")), &releaseInfo); err != nil {
				t.Errorf("Failed to unmarshal button data: %v", err)
			}
			if releaseInfo.JobID != 12345 {
				t.Errorf("Expected button data job_id 12345, got %d", releaseInfo.JobID)
			}
			if releaseInfo.VersionCode != 1001 {
				t.Errorf("Expected button data version_code 1001, got %d", releaseInfo.VersionCode)
			}
			if releaseInfo.VersionName != "1.0.1" {
				t.Errorf("Expected button data version_name '1.0.1', got '%s'", releaseInfo.VersionName)
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]any{
				"data": map[string]any{
					"id": 194275,
				},
			})
		case "/messages/194275/pin":
			pinCalls.Add(1)

			if r.Method != "POST" {
				t.Errorf("Expected POST method, got %s", r.Method)
			}
			w.WriteHeader(http.StatusCreated)
		default:
			t.Errorf("Unexpected path: %s", r.URL.Path)
		}
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
	if messageCalls.Load() != 1 {
		t.Errorf("Expected 1 call to Pachca message API, got %d", messageCalls.Load())
	}
	if pinCalls.Load() != 1 {
		t.Errorf("Expected 1 call to Pachca pin API, got %d", pinCalls.Load())
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
