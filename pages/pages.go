package pages

import (
	"embed"
	"fmt"
	"net/http"
	"net/url"
	"html/template"

	"github.com/IoIxD/LaffForum/database"
	"github.com/IoIxD/LaffForum/pages/funcmap"
)

//go:embed templates/*.*
var pages embed.FS
var tmpl *template.Template
var PageFunctions map[string]func(w http.ResponseWriter, r *http.Request, info InfoStruct)

func init() {
	// initialize the template shit
	tmpl = template.New("")
	tmpl.Funcs(funcmap.FuncMap) // "FuncMap" refers to a template.FuncMap in another file, that isn't included in this one.

	// Parse the templates.
	_, err := tmpl.ParseFS(pages, "templates/*")
	if err != nil {
		fmt.Println(err)
		return
	}

	PageFunctions = make(map[string]func(w http.ResponseWriter, r *http.Request, info InfoStruct))
}

type InfoStruct struct {
	Values     			[]string
	Query      			url.Values
	Session    			*database.Session
	PostValues 			url.Values
	Request 			*http.Request
	ResponseWriter 		http.ResponseWriter
}