package handlers

import (
	"embed"
	"fmt"
	"github.com/charlieegan3/music/pkg/tool/utils"
	"github.com/foolin/goview"
	"github.com/gosimple/slug"
	"html/template"
	"path/filepath"
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
			"artist_slug": func(artistName string) string {
				return fmt.Sprintf(
					"%s-%s",
					utils.CRC32Hash(artistName),
					slug.Make(artistName),
				)
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
