package pages

import (
	"net/http"

	"github.com/IoIxD/LaffForum/pages/funcmap"
)

func GenericTemplate(w http.ResponseWriter, r *http.Request, pagename string, data InfoStruct) (error) {
	if err := tmpl.Funcs(funcmap.FuncMap).ExecuteTemplate(w, pagename+".html", data); err != nil {
		return err
	}
	return nil
}