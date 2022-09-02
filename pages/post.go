package pages

import (
	"bytes"
	"fmt"
	"html/template"
	"net/http"

	str "strings"

	"github.com/IoIxD/LaffForum/database"
	"github.com/IoIxD/LaffForum/strings"
)

var Unescaper = str.NewReplacer(
	"\n", "<br>",
	"&amp;", "&",
	"&#39;", "'",
	"&#34;", "\"",
	"&lt;", "<pre>&lt;</pre>",
	"&gt;", "<pre>&gt;</pre>",
)

type PostPageVariables struct {
	PostSubject  string
	PostContents string
	CanReply     bool
	Author       string
	Timestamp    string

	Deleted   bool
	DeletedBy string

	PassiveErrorHeading     string
	PassiveErrorDescription string

	FatalError string

	PostFields []PostField

	BackTo    int
	PostID    int
	CanDelete bool
	ShowReply bool
}

type PostField struct {
	Author    string
	Timestamp string
	Contents  string
	Deleted   bool
	DeletedBy string

	ParentContents string

	BackTo    int
	PostID    int
	CanDelete bool
}

func init() {
	AddPageFunction("post", PostPageServe)
}

func PostPageServe(w http.ResponseWriter, r *http.Request, info InfoStruct) {
	// Buffer for the final page.
	buf := bytes.NewBuffer(nil)

	// header
	err := tmpl.ExecuteTemplate(buf, "header.html", info)
	if err != nil {
		tmpl.Execute(buf, err.Error())
	}

	// Get all the values to pass to the future templates.
	toPass := PostPageGen(w, r, info.Values, info)

	// if we got an error from that, do not continue.
	if toPass.FatalError != "" {
		buf.Write([]byte(`<span style="white-space: pre-wrap">` +
			template.HTMLEscaper(toPass.FatalError) +
			`</span>`))
		err = tmpl.ExecuteTemplate(buf, "footer.html", info)
		if err != nil {
			tmpl.Execute(buf, err.Error())
		}
	}

	err = tmpl.ExecuteTemplate(buf, "posts_part1", toPass)
	if err != nil {
		tmpl.Execute(buf, err.Error())
	}

	// The contents of the post are displayed here to bypass html/template's html escaping.
	// (template.HTML(...) doesn't work for us)
	// Also whitespace: pre-wrap ruins the padding for some reason so here is where we
	// replace newlines with <br>
	if !toPass.Deleted {
		contents := Unescaper.Replace(strings.Markdown(toPass.PostContents))
		buf.Write([]byte(contents))
	} else {
		if toPass.DeletedBy == toPass.Author {
			buf.Write([]byte("<em>[deleted]</em>"))
		} else {
			buf.Write([]byte("<em>[removed]</em>"))
		}

	}

	if info.Session.Username != "" {
		err = tmpl.ExecuteTemplate(buf, "actionsbox", toPass)
		if err != nil {
			tmpl.Execute(buf, err.Error())
		}
	}

	err = tmpl.ExecuteTemplate(buf, "posts_part2", toPass)
	if err != nil {
		tmpl.Execute(buf, err.Error())
	}

	for _, v := range toPass.PostFields {
		templateString, deletedClassString, deletedString := "", "", ""

		if v.Deleted {
			deletedClassString = " deleted"
		}

		if v.DeletedBy == v.Author {
			deletedString = "<em>[deleted]</em>"
		} else {
			deletedString = "<em>[removed]</em>"
		}

		templateString = `<tr><td class='from` + deletedClassString + `'>`

		if v.Author != "" && !v.Deleted {
			templateString += `<a href='/user/` + v.Author + `'>` + v.Author + `</a>`
		} else {
			templateString += deletedString
		}

		templateString += `</td><td class='contents` + deletedClassString + `'><b>` + v.Timestamp + `</b><br>`

		if v.ParentContents != "" {
			templateString += `<span class='original-post'>` + strings.Markdown(v.ParentContents) + `</span>`
		}

		if !v.Deleted {
			contents := Unescaper.Replace(strings.Markdown(v.Contents))
			templateString += contents
		} else {
			templateString += deletedString
		}
		buf.Write([]byte(templateString))

		if info.Session.Username != "" {
			err = tmpl.ExecuteTemplate(buf, "actionsbox", v)
			if err != nil {
				tmpl.Execute(buf, err.Error())
			}
		}
		buf.Write([]byte(`</td></tr>`))
	}

	err = tmpl.ExecuteTemplate(buf, "posts_part3", toPass)
	if err != nil {
		tmpl.Execute(buf, err.Error())
	}

	err = tmpl.ExecuteTemplate(buf, "footer.html", info)
	if err != nil {
		tmpl.Execute(buf, err.Error())
	}

	w.Write(buf.Bytes())
}

func PostPageGen(w http.ResponseWriter, r *http.Request, values []string, info InfoStruct) (toPass PostPageVariables) {
	username := info.Session.Username
	isadmin := info.Session.Me().Admin()

	if len(values) <= 0 {
		toPass.FatalError = "No post ID given."
		return
	}

	postid := template.HTMLEscaper(values[1])
	post := database.GetPostInfo(postid)
	if post.Error != nil {
		toPass.FatalError = post.Error.Error()
		return
	}

	if post.ReplyTo != 0 {
		http.Redirect(w, r, fmt.Sprintf("/post/%v#%v", post.ReplyTo, post.ID), 303)
	}

	if post.ID == 0 {
		toPass.FatalError = "Non-existent post."
		return
	}

	toPass.Deleted = post.Deleted()

	// redundant check to make sure that if the post is deleted, nothing is even *processed*
	if toPass.Deleted {
		return
	}

	toPass.PostContents = post.Contents
	toPass.PostSubject = post.Subject

	userid := database.GetUserIDByName(username).Result
	sectioninf := database.GetSectionInfo(post.Topic)
	if sectioninf.Error != nil {
		w.Write([]byte(sectioninf.Error.Error()))
		return
	}
	content := []byte(template.HTMLEscaper(info.PostValues.Get("contents")))
	if len(content) >= 1 {
		if sectioninf.AdminOnly == 2 {
			if !isadmin {
				toPass.FatalError = `Permission denied.`
				return
			}
		}

		reply := info.Session.SubmitPost(post.Topic, template.HTMLEscaper("RE: "+post.Subject), string(content), post.ID)
		if reply.Error != nil {
			toPass.PassiveErrorHeading = `Error while submitting your post.`
			toPass.PassiveErrorDescription = reply.Error.Error()
		} else {
			http.Redirect(w, r, fmt.Sprintf("/post/%v", reply.ID), 303)
		}
	}

	author := database.GetUsernameByID(post.Author)
	if author.Error != nil {
		toPass.PassiveErrorHeading = `Could not get author.`
		toPass.PassiveErrorDescription = author.Error.Error()
	} else {
		if author.Result != "" {
			toPass.Author = author.Result.(string)
		}
	}

	timestamp := strings.PrettyTime(post.Timestamp)
	if timestamp.Error != nil {
		toPass.Timestamp = `Couldn't get timestamp; ` + timestamp.Error.Error()
	} else {
		toPass.Timestamp = timestamp.Result.(string)
	}

	toPass.CanReply = false

	if sectioninf.AdminOnly == 2 {
		if isadmin {
			toPass.CanReply = true
		}
	} else if info.Session.Username != "" {
		toPass.CanReply = true
	}

	postFields := make([]PostField, 0)

	posts := database.GetPostsInReplyTo(post.ID)
	if posts.Error != nil {
		toPass.FatalError = fmt.Sprintf("Could not get posts; %v", posts.Error.Error())
		return
	} else {
		var postField PostField
		for _, n := range posts.Posts {
			postField.Deleted = n.Deleted()
			postField.DeletedBy = n.DeletedBy()
			postField.Contents = n.Contents
			postField.BackTo = post.ID
			postField.PostID = n.ID
			postField.ParentContents = ""

			postField.CanDelete = false
			if (postField.Author == info.Session.Username) ||
				(postField.Deleted && postField.Author == postField.DeletedBy) ||
				isadmin {
				postField.CanDelete = true
			}

			// Only calculate the following if it's a visible post.
			if !postField.Deleted || isadmin || userid == n.Author {
				// Poster name
				if !postField.Deleted || isadmin {
					author := database.GetUsernameByID(n.Author)
					if author.Error != nil {
						postField.Author = `Could not get author; ` + author.Error.Error()
					} else {
						postField.Author = author.Result.(string)
					}
				}
				// Timestamp
				timestamp := strings.PrettyTime(n.Timestamp)
				if timestamp.Error != nil {
					postField.Timestamp = `Could not parse; ` + timestamp.Error.Error()
				} else {
					postField.Timestamp = timestamp.Result.(string)
				}
				// Parent Post
				if n.ReplyTo != post.ID {
					post_ := database.GetPostInfo(n.ReplyTo)
					if post_.Error == nil {
						if !post_.Deleted() || isadmin {
							postField.ParentContents = post_.Contents
						} else {
							postField.ParentContents = "[deleted]"
						}
					} else {
						toPass.FatalError = post_.Error.Error()
						return
					}
				}
			}

			postFields = append(postFields, postField)
		}
	}

	toPass.PostFields = postFields
	toPass.BackTo = post.ID
	toPass.PostID = post.ID
	toPass.DeletedBy = post.DeletedBy()
	toPass.CanDelete = false
	if (toPass.Author == info.Session.Username) ||
		(toPass.Deleted && toPass.Author == toPass.DeletedBy) ||
		isadmin {
		toPass.CanDelete = true
	}

	return
}
