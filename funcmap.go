package main

import (
	"html/template"
	texttemplate "text/template"
)

// function map
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

	"String":   String,
	"IsString": IsString,
	"IsInt":    IsInt,

	"HTMLEscape": HTMLEscape,
	"Markdown":   Markdown,

	"VerifyCaptcha": VerifyCaptcha,

	"DeletePost":  DeletePost,
	"RestorePost": RestorePost,
}

// function map for post.html which needs unescaped html
var textTemplateFuncMap = texttemplate.FuncMap{
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

	"String":   String,
	"IsString": IsString,
	"IsInt":    IsInt,

	"HTMLEscape": HTMLEscape,
	"Markdown":   Markdown,

	"VerifyCaptcha": VerifyCaptcha,

	"DeletePost":  DeletePost,
	"RestorePost": RestorePost,
}
