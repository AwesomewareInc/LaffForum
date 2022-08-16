package main

import (
	"strings"

	"github.com/gomarkdown/markdown"
)

func Markdown(val string) (string) {
	val = HTMLEscape(val)
	return string(markdown.ToHTML([]byte(val),nil,nil))
}

func HTMLEscape(val string) (string) {
	val = strings.Replace(val,"<","\\<",99)
	val = strings.Replace(val,">","\\>",99)
	return val
}