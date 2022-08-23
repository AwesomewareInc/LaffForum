package pages

import (
	"bytes"
	"html/template"
	"net/http"

	"github.com/IoIxD/LaffForum/pages/funcmap"
)

func GenericTemplate(w http.ResponseWriter, r *http.Request, pagename string, data any) error {
	// redirect function
	// (todo: rewrite pages to not need these!)
	tempFuncMap := template.FuncMap{
		"Redirect": func(url string, code int) string {
			http.Redirect(w, r, url, code)
			return ""
		},
	}
	buf := bytes.NewBuffer(nil)
	if err := tmpl.Funcs(funcmap.FuncMap).Funcs(tempFuncMap).ExecuteTemplate(buf, pagename+".html", data); err != nil {
		return err
	}
	w.Write(buf.Bytes())
	return nil
}
