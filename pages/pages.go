package pages

import (
	"embed"
	"fmt"
	"html/template"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/IoIxD/LaffForum/database"
	"github.com/IoIxD/LaffForum/pages/funcmap"
)

//go:embed templates/*.*
var pages embed.FS
var tmpl *template.Template

type PageFunctionsStruct struct {
	sync.Mutex
	f map[string]func(w http.ResponseWriter, r *http.Request, info InfoStruct)
} 
var PageFunctions PageFunctionsStruct

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

	PageFunctions.f = make(map[string]func(w http.ResponseWriter, r *http.Request, info InfoStruct))
}

type InfoStruct struct {
	Values     			[]string
	Query      			url.Values
	Session    			*database.Session
	PostValues 			url.Values
	Request 			*http.Request
	ResponseWriter 		http.ResponseWriter
}

// Safely add a function to the page functions

func AddPageFunction(name string, f func(w http.ResponseWriter, r *http.Request, info InfoStruct)) {
	if(PageFunctions.f == nil) {
		go func() {
			for {
				time.Sleep(150 * time.Millisecond)
				if(PageFunctions.f == nil) {
					PageFunctions.Lock()
					PageFunctions.f[name] = f
					PageFunctions.Unlock()
				}
			}
		}()
	} else {
		PageFunctions.f[name] = f
	}
}

func (pagestruct *PageFunctionsStruct) Get(name string) (func(w http.ResponseWriter, r *http.Request, info InfoStruct), bool) {
	what, ok := pagestruct.f[name]
	return what, ok
}