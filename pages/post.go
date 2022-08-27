package pages

import (
	"bytes"
	"fmt"
	"html/template"
	"net/http"

	"github.com/IoIxD/LaffForum/database"
	"github.com/IoIxD/LaffForum/strings"
)

type PostPageVariables struct {
	PostSubject string
	PostContents string 
	CanReply bool
	Author string
	Timestamp string

	PassiveErrorHeading string
	PassiveErrorDescription string

	FatalError string

	PostFields 	[]PostField

	BackTo int
	PostID int
	CanDelete bool
}

type PostField struct {
	Author 		string
	Timestamp 	string
	Contents 	string
	Deleted 	bool
	ParentContents string

	BackTo int
	PostID int
	CanDelete bool
}

func init() {
	PageFunctions["post"] = PostPageServe
}

func PostPageServe(w http.ResponseWriter, r *http.Request, info InfoStruct) {
	buf := bytes.NewBuffer(nil)

	err := tmpl.ExecuteTemplate(buf,"header.html",info)
	if(err != nil) {
		tmpl.Execute(buf,err.Error())
	}
	
	toPass := PostPageGen(w,r,info.Values,info)

	err = tmpl.ExecuteTemplate(buf,"posts_part1",toPass)
	if(err != nil) {
		tmpl.Execute(buf,err.Error())
	}

	if(info.Session.Username != "") {
		err = tmpl.ExecuteTemplate(buf,"actionsbox",toPass)
		if(err != nil) {
			tmpl.Execute(buf,err.Error())
		}
	}

	err = tmpl.ExecuteTemplate(buf,"posts_part2",toPass)
	if(err != nil) {
		tmpl.Execute(buf,err.Error())
	}

	for _, v := range toPass.PostFields {	
		
		templateString, deletedString := "", ""

		if(v.Deleted) {
			deletedString = " deleted"
		}

		templateString = "<tr><td class='from"+deletedString+"'>"
		if(v.Author != "") {
			templateString += "<a href='/user/"+v.Author+"'>"+v.Author+"</a>"
		} else {
			templateString += "<em>[deleted]</em>"
		}
		templateString += "</td>"
		templateString += "<td class='contents"+deletedString+"'>"

		templateString += "<b>"+v.Timestamp+"</b><br>"

		if(v.ParentContents != "") {
			templateString += "<span class='original-post'>"+strings.Markdown(v.ParentContents)+"</span>"
		}

		if(!v.Deleted) {
			templateString += strings.Markdown(v.Contents)
		} else {
			templateString += "<em>[deleted]</em>"
		}
		tmpl.Execute(buf,template.HTML(templateString))
		fmt.Print(templateString)
		if(info.Session.Username != "") {
			err = tmpl.ExecuteTemplate(buf,"actionsbox",v)
			if(err != nil) {
				tmpl.Execute(buf,err.Error())
			}
		}
		templateString = `</td></tr>`;
		fmt.Print(templateString)
		tmpl.Execute(buf,template.HTML(templateString))
		fmt.Print("\n")
	}

	err = tmpl.ExecuteTemplate(buf,"posts_part3",toPass)
	if(err != nil) {
		tmpl.Execute(buf,err.Error())
	}

	err = tmpl.ExecuteTemplate(buf,"footer.html",info)
	if(err != nil) {
		tmpl.Execute(buf,err.Error())
	}

	w.Write(buf.Bytes())
}

func PostPageGen(w http.ResponseWriter, r *http.Request, values []string, info InfoStruct) (toPass PostPageVariables) {
	username := info.Session.Username
	isadmin := info.Session.Me().Admin()

	if(len(values) <= 0) {
		toPass.FatalError = "No post ID given."
		return
	}

	postid := template.HTMLEscaper(values[1])
	post := database.GetPostInfo(postid)
	if (post.Error != nil) {
		toPass.FatalError = post.Error.Error()
		return
	}

	if (post.ReplyTo != 0) {
		http.Redirect(w,r,fmt.Sprintf("/post/%v%v",post.ReplyTo,post.ID), 303)
	}

	if (post.ID == 0) {
		toPass.FatalError = "Non-existent post."
		return
	}
	
	toPass.PostSubject = post.Subject
	toPass.PostContents = post.Contents

	userid := database.GetUserIDByName(username).Result
	sectioninf := database.GetSectionInfo(post.Topic)
	if(sectioninf.Error != nil) {
		w.Write([]byte(sectioninf.Error.Error()))
		return
	}
	content := []byte(template.HTMLEscaper(info.PostValues.Get("contents")))
	if(len(content) >= 1) {
		if(sectioninf.AdminOnly == 2) {
			if(!isadmin) {
				toPass.FatalError = `Permission denied.`
				return
			}
		}

		reply := info.Session.SubmitPost(post.Topic,template.HTMLEscaper("RE: "+post.Subject),string(content),post.ID)
		if(reply.Error != nil) {
			toPass.PassiveErrorHeading = `Error while submitting your post.`;
			toPass.PassiveErrorDescription = reply.Error.Error()
		} else {
			http.Redirect(w,r,fmt.Sprintf("/post/%v",reply.ID), 303)
		}
	}

	author := database.GetUsernameByID(post.Author)
	if(author.Error != nil) {
		toPass.PassiveErrorHeading = `Could not get author.`;
		toPass.PassiveErrorDescription = author.Error.Error();
	} else {
		if(author.Result != "") {
			toPass.Author = author.Result.(string)
		}
	}

	timestamp := strings.PrettyTime(post.Timestamp)
	if (timestamp.Error != nil) {
		toPass.Timestamp = `Couldn't get timestamp; `+timestamp.Error.Error()
	} else {
		toPass.Timestamp = timestamp.Result.(string)
	}

	if(sectioninf.AdminOnly == 2) {
		if(isadmin) {
			toPass.CanReply = true
		} else {
			toPass.CanReply = false
		}
	}


	postFields := make([]PostField, 0)

	posts := database.GetPostsInReplyTo(post.ID)
	if (posts.Error != nil) {
		toPass.FatalError = fmt.Sprintf("Could not get posts; %v",posts.Error.Error())
		return
	} else {
		var postField PostField
		for _, n := range posts.Posts {
			postField.Deleted = n.Deleted()
			if (!postField.Deleted || isadmin) {
				author := database.GetUsernameByID(n.Author)
				if(author.Error != nil) {
					postField.Author = `Could not get author; `+author.Error.Error()
				} else {
					postField.Author = author.Result.(string);
				}
			}
			if((!postField.Deleted) || isadmin || (userid == n.Author)) {
				timestamp := strings.PrettyTime(n.Timestamp)
				if(timestamp.Error != nil) {
					postField.Timestamp = `Could not parse; `+timestamp.Error.Error()
				} else {
					postField.Timestamp = timestamp.Result.(string)
				}
				if (n.ReplyTo != post.ID) {
					post_ := database.GetPostInfo(n.ReplyTo)
					if(post_.Error == nil) {
						if ((!post_.Deleted()) || isadmin ) {
							postField.ParentContents = post_.Contents
						} else {
							postField.ParentContents = `Deleted by a moderator`;
						}
					}
				}
			}

			postField.Contents = n.Contents

			postField.BackTo = post.ID
			postField.PostID = n.ID

			if(postField.Author == info.Session.Username || info.Session.Me().Admin()) {
				postField.CanDelete = true
			}
		}
		postFields = append(postFields,postField)
	}

	toPass.PostFields = postFields
	toPass.BackTo = post.ID
	toPass.PostID = post.ID
	if(toPass.Author == info.Session.Username || info.Session.Me().Admin()) {
		toPass.CanDelete = true
	}
	return
}
