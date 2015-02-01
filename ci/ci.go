package ci

import (
	"fmt"
	"net/http"
)

func init() {
	http.HandleFunc("/hello", func(w http.ResponseWriter, req *http.Request) {
		query := req.URL.Query().Get("query")
		html := fmt.Sprintf("<html><head><title>Hello World</title></head><body><h1>Hello World</h1><p id=\"query\">%s</p></body></html>\n", query)
		fmt.Fprint(w, html)
	})
}
