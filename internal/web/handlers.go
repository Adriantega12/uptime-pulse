package web

import (
	"html/template"
	"log"
	"fmt"

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
