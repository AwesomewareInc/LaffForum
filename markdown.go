package main

import (
	"text/template"

	"github.com/gomarkdown/markdown"
)

func Markdown(val string) (string) {
	val = template.HTMLEscapeString(val)
	return string(markdown.ToHTML([]byte(val),nil,nil))
}

func HTMLEscape(val string) (string) {
	return template.HTMLEscapeString(val)
}
