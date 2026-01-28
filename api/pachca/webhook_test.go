package handler

import (
	"testing"
	"net/http"
	"net/http/httptest"
)

func TestFetchAndRespond(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.WriteHeader(http.StatusOK)
    }))
    defer mockServer.Close()

    // Test with mock server
    req := httptest.NewRequest("GET", "/api/fetch", nil)
    w := httptest.NewRecorder()

    HandlePachcaHook(w, req, mockServer.Client())

    if w.Code != http.StatusNoContent {
        t.Errorf("Expected 204, got %d", w.Code)
    }
}
