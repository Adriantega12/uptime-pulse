package web_test

import (
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"uptime-pulse/internal/web"
)

func TestHandleDashboard(t *testing.T) {
	webServer := web.NewServer(nil, "views")

	request, err := http.NewRequest(http.MethodGet, "/", nil)
	if err != nil {
		log.Fatalf("Error while mocking GET request %v", err)
	}
	recorder := httptest.NewRecorder()
	webServer.HandleDashboard(recorder, request)
	response := recorder.Result()
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Errorf("Expected status code : %d, got : %d", http.StatusOK, response.StatusCode)
	}
	responseContentType := response.Header.Values("Content-Type")[0]
	if !strings.Contains(responseContentType, "text/html") {
		t.Errorf("Expected Content-Type to contain : %s, got : %s", "text/html", responseContentType)
	}
	bodyBytes, err := io.ReadAll(response.Body)
	stringBodyBytes := string(bodyBytes)
	if !strings.Contains(stringBodyBytes, "Uptime Pulse Monitor") {
		log.Print(err)
		t.Errorf("Expected body to contain \"Uptime Pulse Monitor\", got : %s", stringBodyBytes)
	}
}
