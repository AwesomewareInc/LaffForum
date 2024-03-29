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
	"&amp;", "&",
	"&#39;", "'",
	"&#34;", "\"",
	"&lt;", "<pre>&lt;</pre>",
	"&gt;", "<pre>&gt;</pre>",
)

type PostPageVariables struct {
	PostSubject  string
	PostContents string
	PostEdited   bool
	CanReply     bool
	Author       string
	Timestamp    string
	Pronouns     string

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
	CanEdit   bool
}

type PostField struct {
	Author     string
	Timestamp  string
	Contents   string
	Deleted    bool
	DeletedBy  string
	Pronouns   string
	BeenEdited bool

	ParentContents string

	BackTo    int
	PostID    int
	CanDelete bool
	CanEdit   bool
	CanReply  bool
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

	if !toPass.Deleted {
		contents := Unescaper.Replace(strings.Markdown(toPass.PostContents))
		buf.Write([]byte("<span class='box'>" + contents + "</span>"))
	} else {
		if toPass.DeletedBy == toPass.Author {
			buf.Write([]byte("<em>[deleted]</em>"))
		} else {
			buf.Write([]byte("<em>[removed]</em>"))
		}

	}

	if info.Session.Username() != "" {
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
			var editedString string
			if v.BeenEdited {
				editedString = `  <em class='edited'>[edited]</em>`
			}
			templateString += `
			<span class="hbox">
				<span class="unbox box">
					<a class='username' href='/user/` + v.Author + `'>` + v.Author + `</a>
				</span>
				<span class="tmbox box">
					<em class='timestamp'> ` + v.Timestamp + `</em>
				</span>
				<span class="prbox box">
					<em class='pronouns'> ` + v.Pronouns + `</em>` + editedString + `
				</span>`
		} else {
			templateString += deletedString
		}

		templateString += `</td><td class='contents` + deletedClassString + `'>`

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

		if info.Session.Username() != "" {
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
	username := info.Session.Username()
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
		http.Redirect(w, r, fmt.Sprintf("/post/%v#%v", post.ReplyTo, post.ID), http.StatusSeeOther)
	}

	if post.ID == 0 {
		toPass.FatalError = "Non-existent post."
		return
	}

	toPass.BackTo = post.ID
	toPass.PostID = post.ID
	toPass.DeletedBy = post.DeletedBy()

	author := database.GetUserInfo(post.Author)
	if author.Error() != nil {
		toPass.PassiveErrorHeading = `Could not get author.`
		toPass.PassiveErrorDescription = author.Error().Error()
	} else {
		toPass.Author = author.Username()
		toPass.Pronouns = author.Pronouns()
	}

	toPass.CanDelete = false
	toPass.CanEdit = false
	toPass.CanReply = true
	if (!toPass.Deleted && toPass.Author == info.Session.Username()) ||
		(toPass.Deleted && toPass.DeletedBy == info.Session.Username()) ||
		isadmin {
		toPass.CanDelete = true
	}

	if info.Session.Me().Banned() {
		toPass.CanDelete = false
		toPass.CanEdit = false
		toPass.CanReply = false
	}

	if toPass.Author == info.Session.Username() {
		toPass.CanEdit = true
	}
	toPass.Deleted = post.Deleted()

	// redundant check to make sure that if the post is deleted, nothing is even *processed*
	if toPass.Deleted {
		return
	}

	toPass.PostContents = post.Contents
	toPass.PostSubject = post.Subject
	toPass.PostEdited = post.BeenEdited == 1

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

	timestamp := strings.PrettyTime(post.Timestamp)
	if timestamp.Error != nil {
		toPass.Timestamp = `Couldn't get timestamp; ` + timestamp.Error.Error()
	} else {
		toPass.Timestamp = timestamp.Result.(string)
	}

	if !info.Session.Me().Banned() {
		if sectioninf.AdminOnly == 2 {
			if isadmin {
				toPass.CanReply = true
			} else {
				toPass.CanReply = false
			}
		} else if info.Session.Username() != "" {
			toPass.CanReply = true
		}
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
			postField.BeenEdited = n.BeenEdited == 1

			// Only calculate the following if it's a visible post.
			if !postField.Deleted || isadmin || userid == n.Author {
				// Poster name/pronouns
				if !postField.Deleted || isadmin {
					author := database.GetUserInfo(n.Author)
					if author.Error() != nil {
						postField.Author = `Could not get author; ` + author.Error().Error()
						postField.Pronouns = "what"
					} else {
						postField.Author = author.Username()
						postField.Pronouns = author.Pronouns()
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
				postField.CanDelete = false
				postField.CanEdit = false
				postField.CanReply = true

				if postField.Author == info.Session.Username() {
					postField.CanEdit = true
				}

				if (!postField.Deleted && postField.Author == info.Session.Username()) ||
					(postField.Deleted && info.Session.Username() == postField.DeletedBy) ||
					isadmin {
					postField.CanDelete = true
				}

				if info.Session.Me().Banned() {
					postField.CanDelete = false
					postField.CanEdit = false
					postField.CanReply = false
				}

			}

			postFields = append(postFields, postField)
		}
	}
	toPass.PostFields = postFields

	return
}
