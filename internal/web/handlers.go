package web

import (
	"html/template"
	"log"

	"uptime-pulse/internal/engine"
)

type Server struct {
	Engine    *engine.UptimeEngine
	Templates *template.Template
}

func NewServer(uptimeEngine engine.UptimeEngine) *Server {
	templates, err := template.ParseFiles("layout.html", "rows.html")
	if err != nil {
		log.Fatalf("Error creating HTML templates : %x", err)
	}
	return &Server{
		&uptimeEngine,
		templates,
	}
}
