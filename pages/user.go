package pages

import (
	"bytes"
	"fmt"
	"net/http"

	"github.com/IoIxD/LaffForum/database"
	"github.com/IoIxD/LaffForum/strings"
)

func init() {
	AddPageFunction("user", UserPageServe)
}

func UserPageServe(w http.ResponseWriter, r *http.Request, info InfoStruct) {
	// Buffer for the final page.
	buf := bytes.NewBuffer(nil)
	buf2 := bytes.NewBuffer(nil)
	hadError := false
	// header
	err := tmpl.ExecuteTemplate(buf, "header.html", info)
	if err != nil {
		buf2.Write([]byte(err.Error()))
		hadError = true
	}

	// Get all the values to pass to the future templates.
	toPass := UserPageGen(w, r, info.Values, info)
	err = tmpl.ExecuteTemplate(buf, "user.html", toPass)
	if err != nil {
		buf2.Write([]byte(err.Error()))
		hadError = true
	}

	err = tmpl.ExecuteTemplate(buf, "footer.html", info)
	if err != nil {
		buf2.Write([]byte(err.Error()))
		hadError = true
	}
	if hadError {
		w.Write(buf2.Bytes())
	} else {
		w.Write(buf.Bytes())
	}
}

type UserPageVariables struct {
	Session  *database.Session
	Name     string
	Pronouns string
	IsAdmin  bool

	Bio       string
	CreatedAt string

	CanEdit bool

	Error error

	Posts []UserPostField
}

type UserPostField struct {
	Topic   string
	Subject string
	ID      int
	Deleted bool
}

func UserPageGen(w http.ResponseWriter, r *http.Request, values []string, info InfoStruct) (toPass UserPageVariables) {
	if len(info.Values) <= 0 {
		toPass.Error = fmt.Errorf("no values")
	}
	toPass.Session = info.Session
	userid := values[1]
	user := database.GetUserInfo(userid)
	if user.Error() != nil {
		toPass.Error = user.Error()
	}
	if user.ID() == 0 {
		toPass.Error = fmt.Errorf("This user does not exist")
	}
	if user.Deleted() {
		toPass.Error = fmt.Errorf("This user has deactivated their account")
	}
	if user.Banned() {
		toPass.Error = fmt.Errorf("This user has been banned! " + user.BanReason())
	}

	if user.PrettyName() != "" {
		toPass.Name = user.PrettyName()
	} else {
		toPass.Name = user.Username()
	}

	toPass.IsAdmin = user.Admin()

	toPass.Bio = user.Bio()
	toPass.Pronouns = user.Pronouns()

	timestamp := strings.PrettyTime(user.Timestamp())
	if timestamp.Error != nil {
		toPass.CreatedAt = "Couldn't get timestamp; " + timestamp.Error.Error()
	} else {
		toPass.CreatedAt = timestamp.Result.(string)
	}
	toPass.CanEdit = false
	if user.Username() == info.Session.Username() {
		toPass.CanEdit = true
	}

	posts := database.GetPostsFromUser(user.Username())
	if posts.Error != nil {
		toPass.Error = posts.Error
	}
	toPass.Posts = make([]UserPostField, 0)
	for _, v := range posts.Posts {
		post := new(UserPostField)
		topic := database.GetSectionNameByID(v.Topic)
		if topic.Error != nil {
			post.Topic = "(couldn't get topic name; " + topic.Error.Error() + ")"
		} else {
			post.Topic = topic.Result.(string)
		}
		post.Subject = v.Subject
		post.ID = 0
		post.ID = v.ID

		toPass.Posts = append(toPass.Posts, *post)
	}
	return
}
