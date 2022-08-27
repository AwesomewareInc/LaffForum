package main

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	// crackhead trollface goes here
	"github.com/IoIxD/LaffForum/database"
	"github.com/IoIxD/LaffForum/debug"
	"github.com/IoIxD/LaffForum/pages"
)

func main() {
	// initialize a thread in the back that checks for "deactivated accounts" in the database that need to be scrubbed after three months.
	go database.DeletedAccountThread()

	// initialize the main server
	s := &http.Server{
		Addr:           ":8083",
		Handler:        http.HandlerFunc(handlerFunc),
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}

	// shut up, if the server crashes i don't want to wake up to several pings about it one morning.
	// TODO maybe implementing a system for logging to a file.
	defer func() {
		if what := recover(); what != nil {
			fmt.Println(what)
		}
	}()

	// Start the server.
	if err := s.ListenAndServe(); err != nil {
		fmt.Println(err)
		return
	}
}

func handlerFunc(w http.ResponseWriter, r *http.Request) {
	// Handle panics and send them to the user instead of sending them to me (same reason as above).
	defer func() {
		if what := recover(); what != nil {
			w.Header().Set("Content-Type", "text/html")
			w.Header().Set("Content-Name", "500.html")
			http.Error(w, fmt.Sprintf(`
				<h1>Error 500.</h1>
				There was a <b>fatal</b> error on the backend. There's not much else we can say about a fatal error, so please send this to a developer or our support email with a detailed description of what you were doing.<br>
				<hr>
				<pre>%v</pre>`, debug.PublicFacingErrorUnstripped(what.(error)).Error()),
				http.StatusInternalServerError)
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
	var err error

	// Before doing anything else we make a special case for rss, which isn't an internal page or regular page,
	// and it should be called before anything else is done to read/write objects.
	if pagename == "rss" {
		pages.RSSServe(w, r, values)
		return
	}

	// Check if it could refer to an internal page
	if file, err = os.Open("pages/templates/" + pagename + ".html"); err == nil {
		filename = "pages/templates/" + pagename + ".html"
		internal = true
	} else {
		// Otherwise, check if it could refer to a regular file.
		if file, err = os.Open("./" + pagename); err == nil {
			filename = "./" + pagename
		} else {
			// If all else fails, send a 404.
			http.Error(w, err.Error(), 404)
			return
		}
	}

	// relevant session
	sess := database.GetSession(r, w)
	if sess.Error != nil {
		http.Error(w, sess.Error.Error(), 500)
		return
	}
	// Post values
	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	// Get the session relating to the user
	info := new(pages.InfoStruct)
	info.Values = values
	info.Query = r.URL.Query()
	info.Session = sess.Session
	info.PostValues = r.PostForm
	info.Request = r
	info.ResponseWriter = w

	// Serve the file differently based on whether it's an internal page or not.
	if internal {
		f, ok := pages.PageFunctions[pagename]
		if ok {
			f(w, r, *info)
		} else {
			if err := pages.GenericTemplate(w, r, pagename, *info); err != nil {
				http.Error(w, err.Error(), 500)
				return
			}
		}
	} else {
		// get the mime-type.
		contentType, err := GetContentType(file)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		w.Header().Set("Content-Type", contentType)
		w.Header().Set("Content-Name", filename)
		w.WriteHeader(200)
		page, err := os.ReadFile(filename)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		w.Write(page)
	}
}

// Function for getting relevant values from a pagename.
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

// Function for auto-detecting the content type of a file.
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
