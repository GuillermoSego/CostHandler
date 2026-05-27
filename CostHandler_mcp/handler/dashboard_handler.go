package handler

import (
	"html/template"
	"io/fs"
	"net/http"

	"github.com/GuillermoSego/costhandler/mcp/service"
	"github.com/GuillermoSego/costhandler/mcp/web"
)

type DashboardHandler struct {
	tmpl *template.Template
}

func NewDashboardHandler() (*DashboardHandler, error) {
	tmpl, err := template.ParseFS(web.TemplateFS, "templates/dashboard.html")
	if err != nil {
		return nil, err
	}
	return &DashboardHandler{tmpl: tmpl}, nil
}

func (d *DashboardHandler) HandlePage(w http.ResponseWriter, r *http.Request) {
	data := map[string]any{
		"Title":      "CostHandler Dashboard",
		"Categories": service.ValidCategories(),
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	d.tmpl.Execute(w, data)
}

func (d *DashboardHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /dashboard", d.HandlePage)

	staticSub, _ := fs.Sub(web.StaticFS, "static")
	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServerFS(staticSub)))
}
