package main

import "html/template"

var funcMap = template.FuncMap{
	"SetSessionValue": SetSessionValue,

	"Capitalize": Capitalize,

	"CreateUser": CreateUser,
	"VerifyPassword": VerifyPassword,

	"GetSections": GetSections,
	"GetPostsBySectionName": GetPostsBySectionName,
	"GetUsernameByID": 	GetUsernameByID,
	"GetPostInfo":	GetPostInfo,

	"SubmitPost": SubmitPost,
}