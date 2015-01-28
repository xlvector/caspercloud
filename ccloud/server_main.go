package main

import (
	"crawler/casper-cloud"
	"flag"
	"log"
	"net"
	"net/http"
	_ "net/http/pprof"
	"runtime"
)

func main() {
	runtime.GOMAXPROCS(6)
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
	port := flag.String("port", "9000", "port number")
	flag.Parse()

	service := caspercloud.NewCasperServer()
	http.Handle("/submit", service)
	l, e := net.Listen("tcp", *port)
	if e != nil {
		log.Fatal("listen error:", e)
	}
	http.Serve(l, nil)
}
