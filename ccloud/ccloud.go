package main

import (
	"flag"
	"fmt"
	"github.com/xlvector/caspercloud"
	_ "github.com/xlvector/caspercloud/ci"
	"log"
	"net"
	"net/http"
	_ "net/http/pprof"
	"runtime"
)

var health bool

func init() {
	health = true
}

func HandleHealth(w http.ResponseWriter, req *http.Request) {
	if health {
		fmt.Fprint(w, "yes")
	} else {
		http.Error(w, "no", http.StatusNotFound)
	}
}

func HandleStart(w http.ResponseWriter, req *http.Request) {
	health = true
	fmt.Fprint(w, "ok")
}

func HandleShutdown(w http.ResponseWriter, req *http.Request) {
	health = false
	fmt.Fprint(w, "ok")
}

func main() {
	runtime.GOMAXPROCS(6)
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
	port := flag.String("port", "8000", "port number")
	flag.Parse()

	service := caspercloud.NewCasperServer(&caspercloud.CasperCmdFactory{})
	http.Handle("/submit", service)
	http.HandleFunc("/start", HandleStart)
	http.HandleFunc("/shutdown", HandleShutdown)
	http.HandleFunc("/health", HandleHealth)
	http.Handle("/site/",
		http.StripPrefix("/site/",
			http.FileServer(http.Dir("./site"))))
	l, e := net.Listen("tcp", ":"+*port)
	if e != nil {
		log.Fatal("listen error:", e)
	}
	http.Serve(l, nil)
}
