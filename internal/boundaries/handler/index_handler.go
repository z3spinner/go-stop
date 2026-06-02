package handler

import (
	"html/template"
	"net/http"

	"github.com/gin-gonic/gin"
)

// IndexHandler serves index.html with the build version injected for cache busting.
type IndexHandler struct {
	tmpl    *template.Template
	version string
}

func NewIndexHandler(htmlPath, version string) (*IndexHandler, error) {
	tmpl, err := template.ParseFiles(htmlPath)
	if err != nil {
		return nil, err
	}
	return &IndexHandler{tmpl: tmpl, version: version}, nil
}

func (h *IndexHandler) Serve(c *gin.Context) {
	c.Status(http.StatusOK)
	c.Header("Content-Type", "text/html; charset=utf-8")
	_ = h.tmpl.Execute(c.Writer, map[string]string{"Version": h.version})
}
