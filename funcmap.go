package main

import (
	"html/template"
	texttemplate "text/template"
)

// function map
var funcMap = template.FuncMap{
	// working with strings
	"Capitalize": 					Capitalize,
	"HTMLEscape": 					HTMLEscape,
	"Markdown":   					Markdown,
	"PrettyTime": 					PrettyTime,
	"TrimForMeta": 					TrimForMeta,

	// working with user data
	"CreateUser":     				CreateUser,
	"VerifyPassword": 				VerifyPassword,
	"GetIDByUsername":				GetUserIDByName,
	"GetUserIDByName": 				GetUserIDByName,
	"GetUsernameByID":				GetUsernameByID,
	"GetUserInfo":      			GetUserInfo,

	// working with posts
	"GetPostsBySectionName": 		GetPostsBySectionName,
	"GetPostsFromUser":      		GetPostsFromUser,
	"GetPostsInReplyTo":     		GetPostsInReplyTo,
	"GetPostInfo":           		GetPostInfo,
	"GetLastFivePosts": 			GetLastFivePosts,
	"GetUnreadReplyingTo": 			GetUnreadReplyingTo,
	"GetReadReplyingTo": 			GetReadReplyingTo,

	// working with sections/topics
	"GetSections":           		GetSections,
	"GetSectionInfo":        		GetSectionInfo,
	"GetSectionNameByID": 			GetSectionNameByID,

	// working with sessions
	"NewSession": 					NewSession,

	// type conversion/checking 
	"String":   					String,
	"IsString": 					IsString,
	"IsInt":    					IsInt,

	// misc
	"VerifyCaptcha": 				VerifyCaptcha,
	// this is a dummy function that gets overriden in the handler method
	"Redirect":						func(url string, code int) (string) {return ""},
	"PrintThreeMonthsFromNow": 		PrintThreeMonthsFromNow,
}

// function map for post.html which needs unescaped html
var textTemplateFuncMap texttemplate.FuncMap

func init() {
	textTemplateFuncMap = make(texttemplate.FuncMap)
	for k, v := range funcMap {
		textTemplateFuncMap[k] = v
	}
}
