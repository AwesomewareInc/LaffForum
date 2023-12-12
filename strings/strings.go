package strings

import (
	"bytes"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"text/template"
	"time"

	"github.com/IoIxD/LaffForum/database"

	"github.com/dyatlov/go-opengraph/opengraph"
	"github.com/gomarkdown/markdown"
	"github.com/gomarkdown/markdown/html"
	"github.com/gomarkdown/markdown/parser"
)

// cached opengraph
// todo: put this all into a database for permenant caching
var cachedOpenGraph = make(map[string]*opengraph.OpenGraph)
var regexLinkFinder = regexp.MustCompile(`\[(.*?)\]\((.*?)\)`)
var regexRawLinkFinder = regexp.MustCompile(`https?:\/\/(www\.)?[-a-zA-Z0-9@:%._\+~#=]{1,256}\.[a-zA-Z0-9()]{1,6}\b([^\<]*)`)

// Capitalize a string
func Capitalize(value string) string {
	// Treat dashes as spaces
	value = strings.Replace(value, "-", " ", 99)
	valuesplit := strings.Split(value, " ")
	var result string
	for _, v := range valuesplit {
		if len(v) <= 0 {
			continue
		}
		result += strings.ToUpper(v[:1])
		result += v[1:] + " "
	}
	return result
}

// Trim a string to 128 characters, for meta tags.
func TrimForMeta(value string) string {
	if len(value) <= 127 {
		return value
	}
	return value[:128] + "..."
}

// Print the server date three months from now
func PrintThreeMonthsFromNow() string {
	future := time.Now().Add(time.Hour * 2190)
	return future.Format("Jan 02 2006, 03:04:05PM -0700")
}

// Parsing a markdown string.

func Markdown(val string) string {
	// we can't make these static thanks gomarkdown
	var p = parser.NewWithExtensions(parser.CommonExtensions | parser.AutoHeadingIDs | parser.NoEmptyLineBeforeBlock)
	var renderer = html.NewRenderer(html.RendererOptions{Flags: html.CommonFlags | html.SkipLinks | html.SkipHTML})

	// basic shit
	val = template.HTMLEscapeString(val)
	val = string(markdown.Render(p.Parse([]byte(val)), renderer))

	val = strings.ReplaceAll(val, "<p>", "")
	val = strings.ReplaceAll(val, "</p>", "")

	val = strings.ReplaceAll(val, "<br>", "")

	val = strings.ReplaceAll(val, "&ndash;", "--")
	// quake
	val = strings.Replace(val, "{{QUAKE}}",
		`<a href='/WebQuake/Client/index.htm'>
			<iframe width='1024' height='768' src='/WebQuake/Client/index.htm' class='quake-iframe'></iframe>
		</a>`, 1)

	// custom link handling
	rawLinks := regexRawLinkFinder.FindAllString(val, -1)
	for _, link := range rawLinks {
		val = strings.ReplaceAll(val, link, "["+link+"]("+link+")")
	}

	links := regexLinkFinder.FindAllString(val, -1)

	for _, link := range links {
		parts := regexLinkFinder.FindSubmatch([]byte(link))
		var title, href []byte
		if len(parts) >= 1 {
			title = parts[1]
		}
		if len(parts) >= 1 {
			href = parts[2]
		}

		var og *opengraph.OpenGraph
		if _, ok := cachedOpenGraph[string(href)]; ok {
			og = cachedOpenGraph[string(href)]
		} else {
			og = getOpenGraph(href)
			cachedOpenGraph[string(href)] = og
		}

		// edge case: if the href is drive.google.com, embed it using the preview url
		if strings.Contains(string(href), "drive.google.com") {
			newHref := strings.ReplaceAll(string(href), "/view", "/preview")
			newHref = strings.ReplaceAll(newHref, "&ndash", "-")
			val = strings.ReplaceAll(val, link, "<iframe src='"+newHref+"' width='320' height='240' allow=autoplay></iframe>")
			continue
		}

		newVal := bytes.NewBuffer(nil)
		if og.Title == "" && og.Description == "" {
			val = strings.ReplaceAll(val, link, `<a href="`+string(href)+`">`+string(title)+`</a>`)
			continue
		}

		newVal.Write([]byte(`
			<a href="` + string(href) + `">` + string(title) + `</a>
			<a href="` + string(href) + `">
			<div class='opengraph'>`,
		))
		if og.Title != "" {
			newVal.Write([]byte(fmt.Sprintf("<h3>%v %v</h3>", og.Determiner, og.Title)))
		}
		if og.Description != "" {
			newVal.Write([]byte(fmt.Sprintf("<small>%v</small><br>", og.Description)))
		}
		if og.Images != nil {
			for _, image := range og.Images {
				newVal.Write([]byte(fmt.Sprintf("<img src='%v'>", image.URL)))
			}
		}
		if og.Audios != nil {
			for _, audio := range og.Audios {
				newVal.Write([]byte(fmt.Sprintf("<audio src='%v'>", audio.SecureURL)))
			}
		}

		newVal.Write([]byte(`</a></div>`))

		val = strings.ReplaceAll(val, link, newVal.String())
	}

	return val
}

func HTMLEscape(val string) string {
	return template.HTMLEscapeString(val)
}

// Function for formatting a timestamp as "x hours ago"
func PrettyTime(unixTime int) (result database.GenericResult) {
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

func getOpenGraph(href []byte) *opengraph.OpenGraph {
	resp, err := http.Get(string(href))
	if err != nil {
		return &opengraph.OpenGraph{
			Title:       "Could not get URL.",
			Description: err.Error(),
		}
	}
	o := opengraph.NewOpenGraph()
	o.ProcessHTML(resp.Body)
	return o
}
