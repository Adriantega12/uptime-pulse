package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

var getLatencyResponseCodeTests = []struct {
	name               string
	mockUrl            string
	mockStatusCode     int
	expectedStatusCode int
	timeoutInSeconds   int
}{
	{
		"200 OK",
		"",
		http.StatusOK,
		http.StatusOK,
		0,
	},
	{
		"500 Internal Server Error",
		"",
		http.StatusInternalServerError,
		http.StatusInternalServerError,
		0,
	},
	{
		"Server hangup",
		"",
		-1,
		-1,
		10,
	},
	{
		"DNS Lookup Failed",
		"1.2.3.4",
		-1,
		-1,
		0,
	},
}

func TestGetLatencyStatusCode(t *testing.T) {
	for _, tt := range getLatencyResponseCodeTests {

		mockUrl := tt.mockUrl
		mockStatusCode := tt.mockStatusCode
		expectedStatusCode := tt.expectedStatusCode
		timeoutInSeconds := tt.timeoutInSeconds
		t.Run(tt.name, func(t *testing.T) {

			server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Header info
				if mockStatusCode >= 0 {
					w.WriteHeader(mockStatusCode)
				}

				// Timeout
				timeoutDuration := time.Duration(timeoutInSeconds) * time.Second
				time.Sleep(timeoutDuration)
			}))

			// Need to start server, but only if no mock URL is given
			var url string
			if mockUrl == "" {
				// Start server
				server.Start()

				// Get relevant info from server
				url = server.URL
			} else {
				url = mockUrl
			}

			/*
			 * Have to close for all servers, even unstarted servers,
			 * due to underlying system descriptors and structures that might
			 * have been initialized
			 */
			defer server.Close()

			result := getLatency(url)
			actualStatusCode := result.StatusCode
			if actualStatusCode != expectedStatusCode {
				t.Errorf("Expected : %d, got : %d", expectedStatusCode, actualStatusCode)
			}
		})
	}
}
