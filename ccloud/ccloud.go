package main

import (
	"flag"
	"github.com/xlvector/caspercloud"
	_ "github.com/xlvector/caspercloud/ci"
	"golang.org/x/net/websocket"
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
	host := flag.String("host", "127.0.0.1", "host")
	flag.Parse()

	service := caspercloud.NewCasperServer(*host)
	http.Handle("/submit", service)
	http.Handle("/start", HandleStart)
	http.Handle("/shutdown", HandleShutdown)
	http.Handle("/health", HandleHealth)
	http.Handle("/ws/submit", websocket.Handler(service.ServeWebSocket))
	http.Handle("/images/",
		http.StripPrefix("/images/",
			http.FileServer(http.Dir("./images"))))
	http.Handle("/site/",
		http.StripPrefix("/site/",
			http.FileServer(http.Dir("./site"))))
	l, e := net.Listen("tcp", ":"+*port)
	if e != nil {
		log.Fatal("listen error:", e)
	}
	http.Serve(l, nil)
}
