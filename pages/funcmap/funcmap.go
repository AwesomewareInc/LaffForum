package funcmap

import (
	"html/template"
	texttemplate "text/template"

	"github.com/IoIxD/LaffForum/database"
	"github.com/IoIxD/LaffForum/strings"
)

// function map
var FuncMap = template.FuncMap{
	// working with strings
	"Capitalize": 					strings.Capitalize,
	"HTMLEscape": 					strings.HTMLEscape,
	"Markdown":   					strings.Markdown,
	"PrettyTime": 					strings.PrettyTime,
	"TrimForMeta": 					strings.TrimForMeta,

	// working with user data
	"CreateUser":     				database.CreateUser,
	"VerifyPassword": 				database.VerifyPassword,
	"GetIDByUsername":				database.GetUserIDByName,
	"GetUserIDByName": 				database.GetUserIDByName,
	"GetUsernameByID":				database.GetUsernameByID,
	"GetUserInfo":      			database.GetUserInfo,

	// working with posts
	"GetPostsBySectionName": 		database.GetPostsBySectionName,
	"GetPostsFromUser":      		database.GetPostsFromUser,
	"GetPostsInReplyTo":     		database.GetPostsInReplyTo,
	"GetPostInfo":           		database.GetPostInfo,
	"GetLastFivePosts": 			database.GetLastFivePosts,
	"GetUnreadReplyingTo": 			database.GetUnreadReplyingTo,
	"GetReadReplyingTo": 			database.GetReadReplyingTo,

	// working with sections/topics
	"GetSections":           		database.GetSections,
	"GetSectionInfo":        		database.GetSectionInfo,
	"GetSectionNameByID": 			database.GetSectionNameByID,

	// working with sessions
	"NewSession": 					database.NewSession,

	// misc
	"VerifyCaptcha": 				database.VerifyCaptcha,
	// this is a dummy function that gets overriden in the handler method
	"Redirect":						func(url string, code int) (string) {return ""},
	"PrintThreeMonthsFromNow": 		strings.PrintThreeMonthsFromNow,
}

// function map for post.html which needs unescaped html
var textTemplateFuncMap texttemplate.FuncMap

func init() {
	textTemplateFuncMap = make(texttemplate.FuncMap)
	for k, v := range FuncMap {
		textTemplateFuncMap[k] = v
	}
}
