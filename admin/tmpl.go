package admin

import (
	"fmt"
	"html/template"
	"io/fs"
	"log"
	"net/http"
	"path/filepath"
	"strings"
)

var (
	templates map[string]*template.Template
)

func render(w http.ResponseWriter, name string, data interface{}) error {
	tmpl, ok := templates[name]
	if !ok {
		return fmt.Errorf("The template %s does not exist.", name)
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	err := tmpl.Execute(w, data)
	if err != nil {
		return err
	}

	return nil
}

func initTemplates() error {
	skeletons, err := filepath.Glob("admin/tmpl/_*.html")
	if err != nil {
		return fmt.Errorf("error loading skeletons: %v", err)
	}
	mainTmpl, err := template.New("main").Parse(`{{define "main" }}{{ template "base" . }}{{ end }}`)
	if err != nil {
		return fmt.Errorf("error parsing main template: %v", err)
	}

	templates = make(map[string]*template.Template)
	err = filepath.Walk("admin/tmpl/", func(path string, info fs.FileInfo, err error) error {
		if strings.HasPrefix(filepath.Base(path), "_") {
			return nil
		}
		if filepath.Ext(path) != ".html" {
			return nil
		}

		name, err := filepath.Rel("admin/tmpl", path)
		if err != nil {
			return err
		}
		// On windows, the path will have \\ but we want consistent naming on all platforms.
		name = strings.ReplaceAll(name, "\\", "/")
		files := append(skeletons, path)
		tmpl, err := mainTmpl.Clone()
		if err != nil {
			return err
		}

		templates[name] = template.Must(tmpl.ParseFiles(files...))
		return nil
	})
	if err != nil {
		return err
	}

	log.Printf("Loaded %d admin templates", len(templates))
	return nil
}
