package web

import (
	"fmt"
	"html/template"
	"log"
	"net/http"

	"uptime-pulse/internal/engine"
)

type Server struct {
	Engine    *engine.UptimeEngine
	Templates *template.Template
}

func NewServer(uptimeEngine *engine.UptimeEngine) *Server {
	viewsDirectory := "internal/web/views"
	layoutFilePath := fmt.Sprintf("%s/layout.html", viewsDirectory)
	rowsFilePath := fmt.Sprintf("%s/rows.html", viewsDirectory)
	templates, err := template.ParseFiles(layoutFilePath, rowsFilePath)
	if err != nil {
		log.Fatalf("Error creating HTML templates : %v", err)
	}
	return &Server{
		uptimeEngine,
		templates,
	}
}

func (s *Server) HandleDashboard(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	err := s.Templates.ExecuteTemplate(w, "layout.html", nil)
	if err != nil {
		log.Printf("Error executing template : %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

func (s *Server) HandlePingsAPI(w http.ResponseWriter, r *http.Request) {
	targetPingsView, scanErrors := s.Engine.GetLatestPings()

	// Error scanning latest ping
	if targetPingsView != nil && len(scanErrors) > 0 {
		for _, scanError := range scanErrors {
			log.Printf("Error getting latest ping of a given target : %v", scanError)
		}
	}

	// Error querying database
	if targetPingsView == nil && len(scanErrors) > 0 {
		log.Printf("Error getting latest pings : %v", scanErrors[0])
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	err := s.Templates.ExecuteTemplate(w, "rows.html", targetPingsView)
	if err != nil {
		log.Printf("Error executing template : %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}
