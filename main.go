package main

import (
	"embed"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	texttemplate "text/template"
	"time"
)

//go:embed pages/*.*
var pages embed.FS
var tmpl *template.Template
var texttmpl *texttemplate.Template

func main() {
	// initialize the template shit
	tmpl = template.New("")
	tmpl.Funcs(funcMap) // "FuncMap" refers to a template.FuncMap in another file, that isn't included in this one.

	// initialize text/template too
	texttmpl = texttemplate.New("")
	texttmpl.Funcs(textTemplateFuncMap) 

	_, err := tmpl.ParseFS(pages, "pages/*")
	if err != nil {
		log.Println(err)
		return
	}
	_, err = texttmpl.ParseFS(pages, "pages/*")
	if err != nil {
		log.Println(err)
		return
	}
	// initialize the main server
	s := &http.Server{
		Addr:           ":8083",
		Handler:        http.HandlerFunc(handlerFunc),
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}
	// If panics can't be handled by the handler function then something is very wrong.
	// But we don't want the server to go down because of it, so we have to ignore it.
	defer func() {
		if recover() != nil {
			fmt.Println(recover())
		}
	}()
	if err := s.ListenAndServe(); err != nil {
		log.Fatalln(err)
	}
}

func handlerFunc(w http.ResponseWriter, r *http.Request) {
	// Handle panics and send them to the user instead of sending them to me.
	defer func() {
		if recover() != nil {
			http.Error(w, recover().(string), http.StatusInternalServerError)
			return
		}
	}()
	// How are we trying to access the site?
	switch r.Method {
	case http.MethodGet, http.MethodHead, http.MethodPost: // These methods are allowed. continue.
	default: // Send them an error for other ones.
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}

	// Get the pagename.
	pagename, values := getPagename(r.URL.EscapedPath())

	var internal bool
	var filename string

	var file *os.File

	// Check if it could refer to an internal page
	if file, err = os.Open("pages/" + pagename + ".html"); err == nil {
		filename = "pages/" + pagename + ".html"
		internal = true
		// Otherwise, check if it could refer to a regular file.
	} else {
		if file, err = os.Open("./" + pagename); err == nil {
			filename = "./" + pagename
		} else {
			// If all else fails, send a 404.
			http.Error(w, err.Error(), 404)
			return
		}
	}

	// get the mime-type.
	contentType, err := GetContentType(file)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Name", filename)
	w.WriteHeader(200)

	// Get the session relating to the user
	var Info struct {
		Values 			[]string
		Query  			url.Values
		Global 			GlobalValues
		PostValues 		url.Values
	}
	Info.Values = values
	Info.Query = r.URL.Query()
	Info.Global.Session = getSession(r)

	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), 500)
	} 

	Info.PostValues = r.PostForm

	// Serve the file differently based on whether it's an internal page or not.
	if internal {
		// On some pages, html escaping needs to be disabled.
		switch(pagename) {
			case "post":
				if err := texttmpl.ExecuteTemplate(w, pagename+".html", &Info); err != nil {
					http.Error(w, err.Error(), 500)
				}
			default:
				if err := tmpl.ExecuteTemplate(w, pagename+".html", &Info); err != nil {
					http.Error(w, err.Error(), 500)
				}
		} 

	} else {
		page, err := os.ReadFile(filename)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		w.Write(page)
	}
}

func getPagename(fullpagename string) (string, []string) {
	// Split the pagename into sections
	if fullpagename[0] == '/' && len(fullpagename) > 1 {
		fullpagename = fullpagename[1:]
	}
	values := strings.Split(fullpagename, "/")

	// Then try and get the relevant pagename from that, accounting for many specifics.
	pagename := values[0]
	switch pagename {
	// If it's blank, set it to the default page.
	case "":
		return "index", values
	// If the first part is resources, then treat the rest of the url normally
	case "resources":
		return fullpagename, values
	}
	return pagename, values
}

func GetContentType(output *os.File) (string, error) {
	ext := filepath.Ext(output.Name())
	file := make([]byte, 1024)
	switch ext {
	case ".svg":
		return "image/svg+xml", nil
	case ".htm", ".html":
		return "text/html", nil
	case ".css":
		return "text/css", nil
	case ".js":
		return "application/javascript", nil
	default:
		_, err := output.Read(file)
		if err != nil {
			return "", err
		}
		return http.DetectContentType(file), nil
	}
}

func PrettyTime(unixTime int) (result GenericResult) {
	unixTimeDur, err := time.ParseDuration(fmt.Sprintf("%vs", time.Now().Unix()-int64(unixTime)))
	if err != nil {
		result.Error = err
		return
	}

	if unixTimeDur.Hours() >= 8760 {
		result.Result = fmt.Sprintf("%0.f years ago", unixTimeDur.Hours()/8760)
		return
	}
	if unixTimeDur.Hours() >= 730 {
		result.Result = fmt.Sprintf("%0.f months ago", unixTimeDur.Hours()/730)
		return
	}
	if unixTimeDur.Hours() >= 168 {
		result.Result = fmt.Sprintf("%0.f weeks ago", unixTimeDur.Hours()/168)
		return
	}
	if unixTimeDur.Hours() >= 24 {
		result.Result = fmt.Sprintf("%0.f days ago", unixTimeDur.Hours()/24)
		return
	}
	if unixTimeDur.Hours() >= 1 {
		result.Result = fmt.Sprintf("%0.f hours ago", unixTimeDur.Hours())
		return
	}
	if unixTimeDur.Minutes() >= 1 {
		result.Result = fmt.Sprintf("%0.f minutes ago", unixTimeDur.Minutes())
		return
	}
	result.Result = fmt.Sprintf("%0.f seconds ago", unixTimeDur.Seconds())
	return
}
