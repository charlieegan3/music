package handlers

import (
	"embed"
	"html/template"
	"path/filepath"

	"github.com/foolin/goview"

	"github.com/charlieegan3/music/pkg/tool/utils"
)

//go:embed views/*
var views embed.FS

var gv *goview.ViewEngine

func init() {
	cnfg := goview.Config{
		Root:      "views",
		Extension: ".html",
		Master:    "layouts/master",
		Funcs: template.FuncMap{
			"name_slug": utils.NameSlug,
			"add": func(a, b int) int {
				return a + b
			},
		},
	}

	gv = goview.New(cnfg)
	gv.SetFileHandler(func(config goview.Config, tmpl string) (string, error) {
		path := filepath.Join(config.Root, tmpl)
		bytes, err := views.ReadFile(path + config.Extension)
		return string(bytes), err
	})
}
