package main

import (
	"html/template"
	"strings"

	"github.com/gomarkdown/markdown"
)

var funcMap = template.FuncMap{
	"Capitalize": Capitalize,

	"CreateUser":     CreateUser,
	"VerifyPassword": VerifyPassword,

	"GetSections":           GetSections,
	"GetPostsBySectionName": GetPostsBySectionName,
	"GetPostsFromUser":      GetPostsFromUser,
	"GetPostsInReplyTo":     GetPostsInReplyTo,
	"GetUsernameByID":       GetUsernameByID,
	"GetPostInfo":           GetPostInfo,
	"GetUserInfo":           GetUserInfo,
	"GetSectionInfo":        GetSectionInfo,

	"SubmitPost": SubmitPost,

	"PrettyTime": PrettyTime,

	"String": String,
	"IsString": IsString,
	"IsInt": IsInt,

	"Markdown": func(val string) (string) {
		val = strings.Replace(val,"<","\\<",99)
		val = strings.Replace(val,">","\\>",99)
		return string(markdown.ToHTML([]byte(val),nil,nil))
	},
}
