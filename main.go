package main

import (
	"bytes"
	"embed"
	"fmt"
	"html/template"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"runtime/debug"
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

	// initialize text/template alongside it for markdown.
	texttmpl = texttemplate.New("")
	texttmpl.Funcs(textTemplateFuncMap)

	// Parse the templates.
	_, err := tmpl.ParseFS(pages, "pages/*")
	if err != nil {
		fmt.Println(err)
		return
	}
	// (re)parse post.html as an unescaped template
	_, err = texttmpl.ParseFS(pages, "pages/post.html")
	if err != nil {
		fmt.Println(err)
		return
	}

	// initialize a thread in the back that checks for "deactivated accounts" in the database that need to be scrubbed after three months.
	go DeletedAccountThread()

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
				<pre>%v</pre>`,PublicFacingErrorUnstripped(what.(error)).Error()), 
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

	// Before doing anything else we make a special case for rss, which isn't an internal page or regular page,
	// and it should be called before anything else is done to read/write objects. 
	if(pagename == "rss") {
		RSSServe(w,r,values)
		return
	}

	// Check if it could refer to an internal page
	if file, err = os.Open("pages/" + pagename + ".html"); err == nil {
		filename = "pages/" + pagename + ".html"
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

	// Get the session relating to the user
	var Info struct {
		Values     	[]string
		Query      	url.Values
		Session    	*Session
		PostValues 	url.Values
		Request 	*http.Request 			// TODO: we could probably merge this into the session object.
		ResponseWriter http.ResponseWriter
	}

	// url values sepereated by /
	Info.Values = values
	// url queries that come ater ?
	Info.Query = r.URL.Query()

	// the arguments going to this function.
	Info.Request = r
	Info.ResponseWriter = w

	// relevant session
	sess := GetSession(r)
	if(sess.Error != nil) {
		http.Error(w, sess.Error.Error(), 500)
		return
	}
	Info.Session = sess.Session

	// Post values
	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	Info.PostValues = r.PostForm

	// Lastly, we want the redirect function; annoyingly we have to declare that right here, inline,
	// because apparently "we're not supposed to do this in templating"
	// (too bad, it's slower to do this with javascript)
	tempFuncMap := template.FuncMap{
		"Redirect": func(url string, code int) (string) {
			http.Redirect(w,r,url,code)
			return ""
		},
	}

	// Serve the file differently based on whether it's an internal page or not.
	if internal {
		// On some post.html escaping needs to be disabled so that Markdown can be displayed.
		switch pagename {
		case "post":
			// By writing it to a buffer first and then writing it to the page, 
			// instead of writing it directly to the writer, any redirect that we
			// should do get processed before headers are set.
			b := bytes.NewBuffer(nil)
			if err := texttmpl.Funcs(tempFuncMap).ExecuteTemplate(b, pagename+".html", &Info); err != nil {
				http.Error(w, err.Error(), 500)
				return
			}
			w.Write(b.Bytes())
		// Every other page is normal.
		default:
			// By writing it to a buffer first and then writing it to the page, 
			// instead of writing it directly to the writer, any redirect that we
			// should do get processed before headers are set.
			b := bytes.NewBuffer(nil)
			if err := tmpl.Funcs(tempFuncMap).ExecuteTemplate(b, pagename+".html", &Info); err != nil {
				http.Error(w, err.Error(), 500)
				return
			}
			w.Write(b.Bytes())
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

// Function for returning debug information about a function when
// templating fails us; we strip it to the information we really need.
// We also strip folder names out in the (admittedly rare) case that this
// could give an attacker a hint on how to attack us.
func PublicFacingError(msg string, err error) error {
	// stack trace
	stacktrace := string(debug.Stack())

	pc, filename_, line, _ := runtime.Caller(1)

	// manipulate the stacktrace.
	stacktraceParts := strings.Split(stacktrace, "\n")[3:] // the first three lines are guaranteed to be part of this call.
	var relevant bool                                      // whether we've begun encountering lines that are part of this project
	var maxStackDetail int                                 // the point at which we stop encountering those lines.
	// for each part of the stacktrace...
	for i, v := range stacktraceParts {
		// does it many slashes in it?
		if strings.Count(v, string(os.PathSeparator)) >= 2 {
			// how many tabs in it?
			tabcount := strings.Count(v, "	")
			// split it into parts and filter the line to only the last part
			stacktracePartParts := strings.Split(v, string(os.PathSeparator))
			// make sure it retains the amount of tabs
			var newString string
			for i := 0; i < tabcount; i++ {
				newString += "	"
			}
			newString += stacktracePartParts[len(stacktracePartParts)-1]
			stacktraceParts[i] = newString
		}
		if strings.Contains(v, "LaffForum") {
			if relevant == false {
				relevant = true
			} else {
				maxStackDetail = i + 3
				break
			}
		}
	}

	// and reduce the stacktrace to fit in the scope we want.
	stacktrace = strings.Join(stacktraceParts[0:maxStackDetail], "\n")
	stacktrace += "\n(...continues entering system files...)"
	filenameParts := strings.Split(filename_, "/")
	filename := filenameParts[len(filenameParts)-1]

	funcname_ := runtime.FuncForPC(pc).Name()
	funcnames := strings.Split(funcname_, ".")
	funcname := funcnames[len(funcnames)-1]

	return fmt.Errorf("%v at %v:%v in %v(), %v. \n\n%v", msg, filename, line, funcname, err.Error(), stacktrace)
}


// Same as above, but we don't strip the stacktrace. Useful for
// panic recovery where the entire stacktrace is important.
func PublicFacingErrorUnstripped(err error) error {
	// stack trace
	stacktrace := string(debug.Stack())

	// filename, line, and information we'll use later to get the scope.
	pc, filename_, line, _ := runtime.Caller(1)

	// manipulate the stacktrace.
	stacktraceParts := strings.Split(stacktrace, "\n")
	// for each part of the stacktrace...
	for i, v := range stacktraceParts {
		// does it many slashes in it?
		if strings.Count(v, string(os.PathSeparator)) >= 2 {
			// how many tabs in it?
			tabcount := strings.Count(v, "	")
			// split it into parts and filter the line to only the last part
			stacktracePartParts := strings.Split(v, string(os.PathSeparator))
			// make sure it retains the amount of tabs
			var newString string
			for i := 0; i < tabcount; i++ {
				newString += "	"
			}
			newString += stacktracePartParts[len(stacktracePartParts)-1]
			stacktraceParts[i] = newString
		}
	}

	// reduce the filename to the part we care about.
	filenameParts := strings.Split(filename_, "/")
	filename := filenameParts[len(filenameParts)-1]

	// get the function name
	funcname_ := runtime.FuncForPC(pc).Name()
	funcnames := strings.Split(funcname_, ".")
	funcname := funcnames[len(funcnames)-1]

	return fmt.Errorf("At %v:%v in %v():\n%v. \n\n%v", filename, line, funcname, err.Error(), stacktrace)

}
