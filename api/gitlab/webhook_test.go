package handler

import (
	"testing"
	//"net/http"
	//"net/http/httptest"
)

// mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
//        w.WriteHeader(http.StatusOK)
//    }))
//    defer mockServer.Close()

//    // Test with mock server
//    req := httptest.NewRequest("GET", "/api/fetch", nil)
//    w := httptest.NewRecorder()

//    HandleGitlabHook(w, req, mockServer.Client())

//    if w.Code != http.StatusNoContent {
//        t.Errorf("Expected 204, got %d", w.Code)
//    }

func TestGitlabNotifiesGooglePlayReleaseIsSuccessful(t *testing.T) {

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
