package pages

import (
	"bytes"
	"fmt"
	"net/http"

	"github.com/IoIxD/LaffForum/database"
	"github.com/IoIxD/LaffForum/strings"

	str "strings"
)

func RSSServe(w http.ResponseWriter, r *http.Request, values []string) {
	if len(values) <= 1 {
		return
	}
	w.Header().Set("Content-Type", "text/xml")
	w.Header().Set("Content-Name", values[1]+".xml")

	buf_ := make([]byte, 0)
	buf := bytes.NewBuffer(buf_)

	buf.Write([]byte(`<?xml version="1.0" encoding="UTF-8" ?>
		<rss version="2.0">
		<channel>
			<link>https://` + r.Host + `</link>
		`))

	switch values[1] {
	case "topic":
		if len(values) <= 2 {
			buf.Write([]byte(`<title>Must specify topic name</title>`))
			break
		}
		buf.Write([]byte(`<title>` + strings.Capitalize(values[2]) + `</title>`))
		buf.Write([]byte(`<description> Posts in ` + strings.Capitalize(values[2]) + `</description>`))
		result := database.GetPostsBySectionName(values[2])
		if result.Error != nil {
			buf.Write([]byte(`<item> 
									<title>Error</title>
									<description>` + result.Error.Error() + `</description>
								</item>`))
			break
		}
		for _, v := range result.Posts {
			subj := str.Replace(v.Subject, "&", "and", 99)
			buf.Write([]byte(`
					<item>
						<title>` + subj + `</title>
						<description>` + strings.TrimForMeta(v.Contents) + `</description>
						<link>https://` + r.Host + `/post/` + fmt.Sprint(v.ID) + `</link>
					</item>
					`))
		}
	case "post":
		// Try and get the post information.
		result := database.GetPostInfo(values[2])
		if result.Error != nil {
			buf.Write([]byte(`<item> 
									<title>Error</title>
									<description>` + result.Error.Error() + `</description>
								</item>`))
			break
		}

		buf.Write([]byte(`<title>` + result.Subject + `</title>`))
		buf.Write([]byte(`<description>"` + result.Subject + `" and its replies</description>`))

		// Show the original post as the first result.
		buf.Write(XMLShowPost(`https://`+r.Host, result))

		replies := database.GetPostsInReplyTo(result.ID)
		if replies.Error != nil {
			buf.Write([]byte(`<item> 
									<title>Error</title>
									<description>` + replies.Error.Error() + `</description>
								</item>`))
			break
		}
		for _, v := range replies.Posts {
			buf.Write(XMLShowPost(`https://`+r.Host, v))
		}
	}
	buf.Write([]byte(`
			</channel>
		</rss>`))
	w.Write(buf.Bytes())
}

func XMLShowPost(url string, post database.Post) []byte {
	author := database.GetUsernameByID(post.Author)
	var authorname string
	if author.Error != nil {
		authorname = author.Error.Error()
	} else {
		if author.Result != nil {
			authorname = author.Result.(string)
		} else {
			authorname = "[deleted]"
		}
	}
	return []byte(`
		<item>
			<title>` + authorname + `: "` + post.Subject + `"</title>
			<description>` + strings.TrimForMeta(post.Contents) + `</description>
			<link>` + url + `/post/` + fmt.Sprint(post.ID) + `</link>
		</item>
	`)
}
