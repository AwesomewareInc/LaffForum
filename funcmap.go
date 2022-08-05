package main

import "html/template"

var funcMap = template.FuncMap{
	"CreateUser": CreateUser,
	"SetSessionValue": SetSessionValue,
	"VerifyPassword": VerifyPassword,
}