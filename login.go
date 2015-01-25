package main

import (
	"bufio"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"
)

var profileFolder string = "./"

func MakeUserProfile(tmpl, username, password string) string {
	path := profileFolder + tmpl + "/" + username
	os.RemoveAll(path)
	err := os.MkdirAll(path, 0755)
	if err != nil {
		log.Fatalln("can not create", path, err)
	}

	cookieFile, err := os.Create(path + "/cookie.txt")
	defer cookieFile.Close()
	cmd := exec.Command("casperjs", tmpl+".js", "--cookies-file="+path+"/cookie.txt", "--proxy=127.0.0.1:8080", "--proxy-type=http")

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Fatalln("can not get stdout pipe:", err)
	}
	bufout := bufio.NewReader(stdout)

	stdin, err := cmd.StdinPipe()
	if err != nil {
		log.Fatalln("can not get stdin pipe:", err)
	}
	bufin := bufio.NewWriter(stdin)

	if err := cmd.Start(); err != nil {
		log.Fatalln("can not start cmd:", err)
	}

	ret := ""
	for {
		line, err := bufout.ReadString('\n')
		if err != nil {
			break
		}
		log.Print(line)
		ret += line + "<br/>"
		if strings.Contains(line, "please input username:") {
			log.Println(username)
			bufin.WriteString(username)
			bufin.WriteRune('\n')
			bufin.Flush()
		}
		if strings.Contains(line, "please input password:") {
			log.Println(password)
			bufin.WriteString(password)
			bufin.WriteRune('\n')
			bufin.Flush()
		}
	}
	return ret
}

func HandleSubmit(w http.ResponseWriter, r *http.Request) {
	params := r.URL.Query()
	tmpl := params.Get("tmpl")
	username := params.Get("username")
	password := params.Get("password")
	out := MakeUserProfile(tmpl, username, password)
	ret := fmt.Sprintf("<p><a href=\"/result/%s/%s/\" target=_blank>Click to see results</a></p><h3>Logs</h3><p>%s</p>", tmpl, username, out)
	w.Header().Set("Content-Type", "text/html;charset=UTF-8")
	fmt.Fprint(w, ret)
}

func main() {
	http.HandleFunc("/submit", HandleSubmit)
	http.Handle("/result/", http.StripPrefix("/result/", http.FileServer(http.Dir("./"))))
	err := http.ListenAndServe(":8088", nil)
	if err != nil {
		log.Fatal(err.Error())
	}
}
