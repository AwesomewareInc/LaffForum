package main

import "html/template"

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
}
