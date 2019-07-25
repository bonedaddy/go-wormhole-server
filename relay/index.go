package relay

import (
	"net/http"
    "text/template"
    "io/ioutil"
)

var indexTemplate *template.Template

func init() {
    tmpl, err := ioutil.ReadFile("index.html")
    if err != nil {
        panic(err)
    }

    indexTemplate = template.Must(template.New("").Parse(string(tmpl)))
}

func handleIndex(w http.ResponseWriter, r *http.Request) {
	indexTemplate.Execute(w, "ws://"+r.Host+"/v1")
}