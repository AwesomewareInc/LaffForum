package main

import "html/template"

var funcMap = template.FuncMap{
	"SetSessionValue": SetSessionValue,

	"Capitalize": Capitalize,

	"CreateUser": CreateUser,
	"VerifyPassword": VerifyPassword,

	"GetSections": GetSections,
	"GetPostsBySectionName": GetPostsBySectionName,
	"GetPostsFromUser": GetPostsFromUser,
	"GetUsernameByID": 	GetUsernameByID,
	"GetPostInfo":	GetPostInfo,
	"GetUserInfo": GetUserInfo,
	"GetSectionInfo": GetSectionInfo,
	
	"SubmitPost": SubmitPost,

	"PrettyTime": PrettyTime,
}