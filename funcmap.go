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

	// working with user data
	"CreateUser":     				CreateUser,
	"VerifyPassword": 				VerifyPassword,
	"GetUsernameByID":				GetUsernameByID,
	"GetUserInfo":      			GetUserInfo,

	// working with posts
	"GetPostsBySectionName": 		GetPostsBySectionName,
	"GetPostsFromUser":      		GetPostsFromUser,
	"GetPostsInReplyTo":     		GetPostsInReplyTo,
	"GetPostInfo":           		GetPostInfo,
	"GetLastTenPostsMadeAtAll": 	GetLastTenPostsMadeAtAll,

	// working with sections/topics
	"GetSections":           		GetSections,
	"GetSectionInfo":        		GetSectionInfo,

	// working with sessions
	"NewSession": 					NewSession,

	// type conversion/checking 
	"String":   					String,
	"IsString": 					IsString,
	"IsInt":    					IsInt,

	// misc
	"VerifyCaptcha": 				VerifyCaptcha,
	"Redirect": 					func() (string) {
		return "lol"
	},
}

// function map for post.html which needs unescaped html
var textTemplateFuncMap texttemplate.FuncMap

func init() {
	textTemplateFuncMap = make(texttemplate.FuncMap)
	for k, v := range funcMap {
		textTemplateFuncMap[k] = v
	}
}
