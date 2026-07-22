package web_test

import (
	"errors"
	"html/template"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"uptime-pulse/internal/engine"
	"uptime-pulse/internal/web"
)

type MockEngine struct {
	MockViews  []engine.TargetPingsView
	MockErrors []error
}

func (m *MockEngine) GetLatestPings() ([]engine.TargetPingsView, []error) {
	return m.MockViews, m.MockErrors
}

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

func TestHandleDashboard_TemplateError(t *testing.T) {
	// 1. Create a server instance
	webServer := web.NewServer(nil, "views")

	// 2. Overwrite Templates with an empty template collection
	// (It doesn't have "layout.html" registered!)
	webServer.Templates = template.New("empty")

	request, _ := http.NewRequest(http.MethodGet, "/", nil)
	recorder := httptest.NewRecorder()

	// 3. Act
	webServer.HandleDashboard(recorder, request)
	response := recorder.Result()
	defer response.Body.Close()

	// 4. Assert that the fallback 500 Internal Server Error path executed
	if response.StatusCode != http.StatusInternalServerError {
		t.Errorf("Expected status code 500 on template failure, got: %d", response.StatusCode)
	}
}

func TestHandlePingsAPI(t *testing.T) {
	fakeData := []engine.TargetPingsView{
		{
			ID:         1,
			URL:        "https://www.google.com",
			StatusCode: 200,
			LatencyMs:  45,
			Error:      "",
			Timestamp:  "2026-06-18 15:00:00",
		},
	}
	mockEngine := &MockEngine{
		MockViews:  fakeData,
		MockErrors: nil,
	}

	webServer := web.NewServer(mockEngine, "views")

	request, err := http.NewRequest(http.MethodGet, "/api/pings", nil)
	if err != nil {
		log.Fatalf("Error while mocking GET request %v", err)
	}
	recorder := httptest.NewRecorder()
	webServer.HandlePingsAPI(recorder, request)
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
	log.Print(stringBodyBytes)
	if !strings.Contains(stringBodyBytes, "https://www.google.com") {
		log.Print(err)
		t.Errorf("Expected body to contain \"https://www.google.com\", got : %s", stringBodyBytes)
	}
	if !strings.Contains(stringBodyBytes, "45ms") { // Assuming your HTML maps standard suffix layouts
		t.Errorf("Expected metric cell string insertion missing. Output: %s", stringBodyBytes)
	}
}

func TestHandlePingsAPI_DatabaseFailure(t *testing.T) {
	mockFaultyEngine := &MockEngine{
		MockViews:  nil,
		MockErrors: []error{errors.New("sqlite: database is locked")},
	}

	webServer := web.NewServer(mockFaultyEngine, "views")

	request, _ := http.NewRequest(http.MethodGet, "/api/pings", nil)
	recorder := httptest.NewRecorder()

	webServer.HandlePingsAPI(recorder, request)
	response := recorder.Result()
	defer response.Body.Close()

	if response.StatusCode != http.StatusInternalServerError {
		t.Errorf("Expected status code 500 Internal Server Error, got: %d", response.StatusCode)
	}

	bodyBytes, _ := io.ReadAll(response.Body)
	stringBodyBytes := string(bodyBytes)
	if !strings.Contains(stringBodyBytes, "Internal Server Error") {
		t.Errorf("Expected fallback string response body error message, got: %s", stringBodyBytes)
	}
}

func TestHandlePingsAPI_PartialScanErrors(t *testing.T) {
	mockFaultyEngine := &MockEngine{
		MockViews: []engine.TargetPingsView{
			{URL: "https://success.com", StatusCode: 200},
		},
		MockErrors: []error{errors.New("row 2 failed to scan")},
	}

	webServer := web.NewServer(mockFaultyEngine, "views")

	request, _ := http.NewRequest(http.MethodGet, "/api/pings", nil)
	recorder := httptest.NewRecorder()

	webServer.HandlePingsAPI(recorder, request)
	response := recorder.Result()
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		t.Errorf("Expected status code 200 OK, got: %d", response.StatusCode)
	}

	bodyBytes, _ := io.ReadAll(response.Body)
	stringBodyBytes := string(bodyBytes)
	if !strings.Contains(stringBodyBytes, "https://success.com") {
		t.Errorf("Expected string to contain mock URL, instead got: %s", stringBodyBytes)
	}
}
