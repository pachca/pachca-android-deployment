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

func TestPachcaNotifiesPromoteBuildButtonClicked(t *testing.T) {
	var viewCalls atomic.Int32

	mockPachca := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/views/open":
			viewCalls.Add(1)

			if r.Method != "POST" {
				t.Errorf("Expected POST method, got %s", r.Method)
			}

			var viewReq PachcaViewRequest
			json.NewDecoder(r.Body).Decode(&viewReq)

			if viewReq.TriggerID == "" {
				t.Error("Expected trigger_id to be non-empty")
			}

			if viewReq.Type != "view" {
				t.Errorf("Expected type 'view', got '%s'", viewReq.Type)
			}

			if viewReq.View.Title != "Promote Release" {
				t.Errorf("Expected title 'Promote Release', got '%s'", viewReq.View.Title)
			}

			if len(viewReq.View.Blocks) != 3 {
				t.Errorf("Expected 3 blocks, got %d", len(viewReq.View.Blocks))
			}

			if viewReq.View.Blocks[0].Type != "header" {
				t.Errorf("Expected block[0] type 'header', got '%s'", viewReq.View.Blocks[0].Type)
			}
			if viewReq.View.Blocks[0].Text == "" {
				t.Error("Expected header text to be non-empty")
			}

			rolloutBlock := viewReq.View.Blocks[1]
			if rolloutBlock.Type != "input" {
				t.Errorf("Expected block[1] type 'input', got '%s'", rolloutBlock.Type)
			}
			if rolloutBlock.Name != "rollout_percentage" {
				t.Errorf("Expected block[1] name 'rollout_percentage', got '%s'", rolloutBlock.Name)
			}
			if rolloutBlock.Label != "Rollout percentage" {
				t.Errorf("Expected block[1] label 'Rollout percentage', got '%s'", rolloutBlock.Label)
			}
			if !rolloutBlock.Required {
				t.Error("Expected rollout_percentage to be required")
			}
			if rolloutBlock.Multiline {
				t.Error("Expected rollout_percentage to be single-line")
			}
			if rolloutBlock.Hint == "" {
				t.Error("Expected rollout_percentage to have a hint")
			}

			notesBlock := viewReq.View.Blocks[2]
			if notesBlock.Type != "input" {
				t.Errorf("Expected block[2] type 'input', got '%s'", notesBlock.Type)
			}
			if notesBlock.Name != "release_notes" {
				t.Errorf("Expected block[2] name 'release_notes', got '%s'", notesBlock.Name)
			}
			if notesBlock.Label != "Release notes" {
				t.Errorf("Expected block[2] label 'Release notes', got '%s'", notesBlock.Label)
			}
			if !notesBlock.Multiline {
				t.Error("Expected release_notes to be multiline")
			}
			if !notesBlock.Required {
				t.Error("Expected release_notes to be required")
			}
			if notesBlock.MaxLength != 500 {
				t.Errorf("Expected release_notes max_length 500, got %d", notesBlock.MaxLength)
			}

			w.WriteHeader(http.StatusOK)
		default:
			t.Errorf("Unexpected path: %s", r.URL.Path)
		}
	}))
	defer mockPachca.Close()

	t.Setenv(shared.EnvPachcaUrl, mockPachca.URL)
	t.Setenv(shared.EnvPachcaKey, "test-api-key")

	pachcaPayload := map[string]any{
		"type":       "button",
		"event":      "click",
		"trigger_id": "550e8400-e29b-41d4-a716-446655440000",
		"data":       "promote:12345",
		"message_id": 194275,
		"user_id":    123,
		"chat_id":    198,
	}
	payloadBytes, _ := json.Marshal(pachcaPayload)

	req := httptest.NewRequest("POST", "/pachca/webhook", bytes.NewReader(payloadBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	HandlePachcaHook(w, req, mockPachca.Client())

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
	if viewCalls.Load() != 1 {
		t.Errorf("Expected 1 call to Pachca view API, got %d", viewCalls.Load())
	}
}

func TestPachcaNotifiesPromoteBuildFormFilled(t *testing.T) {

}

func TestPachcaNotifiesUpdateRolloutButtonClicked(t *testing.T) {

}

func TestPachcaNotifiesUpdateRolloutFormFilled(t *testing.T) {

}

func TestPachcaNotifiesReleaseToOtherStoresButtonClicked(t *testing.T) {

}

func TestPachcaNotifiesReleaseToOtherStoresFormFilled(t *testing.T) {

}
