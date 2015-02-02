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

	http.HandleFunc("/form/init", func(w http.ResponseWriter, req *http.Request) {
		html := `
            <html>
                <head>
                    <title>Input Phone</title>
                </head>
                <body>
                    <form action="/form/phone">
                        <input type="text" id="phone" name="phone" />
                        <input type="submit" value="submit" id="submit" />
                    </form>
                </body>
            </html>
        `
		fmt.Fprint(w, html)
	})

	http.HandleFunc("/form/phone", func(w http.ResponseWriter, req *http.Request) {
		html := `
            <html>
                <head>
                    <title>Input Verify Code</title>
                </head>
                <body>
                    <form action="/form/verify_code">
                        <input type="text" id="verify_code" name="verify_code" />
                        <input type="submit" value="submit" id="submit" />
                    </form>
                </body>
            </html>
        `
		fmt.Fprint(w, html)
	})

	http.HandleFunc("/form/verify_code", func(w http.ResponseWriter, req *http.Request) {
		html := `
            <html>
                <head>
                    <title>Input Verify Code</title>
                </head>
                <body>
                    <h1>Thanks</h1>
                </body>
            </html>
        `
		fmt.Fprint(w, html)
	})
}
