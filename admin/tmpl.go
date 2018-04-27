package admin

import (
	"fmt"
	"html/template"
	"net/http"
	"path/filepath"
)

var (
	templates map[string]*template.Template
)

func render(w http.ResponseWriter, name string, data interface{}) {
	tmpl, ok := templates[name]
	if !ok {
		http.Error(w, fmt.Sprintf("The template %s does not exist.", name), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	err := tmpl.Execute(w, data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func initTemplates() error {
	skeletons, err := filepath.Glob("admin/tmpl/_*.html")
	if err != nil {
		return fmt.Errorf("error loading skeletons: %v", err)
	}
	tmplFiles, err := filepath.Glob("admin/tmpl/*.html")
	if err != nil {
		return fmt.Errorf("error loading templates: %v", err)
	}
	mainTmpl, err := template.New("main").Parse(`{{define "main" }}{{ template "base" . }}{{ end }}`)
	if err != nil {
		return fmt.Errorf("error parsing main template: %v", err)
	}

	templates = make(map[string]*template.Template)
	for _, tmplFile := range tmplFiles {
		fileName := filepath.Base(tmplFile)
		files := append(skeletons, tmplFile)
		tmpl, err := mainTmpl.Clone()
		if err != nil {
			return fmt.Errorf("here %v", err)
		}
		templates[fileName] = template.Must(tmpl.ParseFiles(files...))
	}

	return nil
}
