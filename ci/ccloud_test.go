package ci

import (
	"flag"
	"github.com/xlvector/caspercloud"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os/exec"
	//"runtime"
	"testing"
)

var httpClient *http.Client

func init() {
	go ccloudServer()
	httpClient = &http.Client{}
}

func ccloudServer() {
	//runtime.GOMAXPROCS(6)
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
	port := flag.String("port", "8000", "port number")
	flag.Parse()

	service := caspercloud.NewCasperServer()
	http.Handle("/submit", service)
	l, e := net.Listen("tcp", ":"+*port)
	if e != nil {
		log.Fatal("listen error:", e)
	}
	http.Serve(l, nil)
}

func BenchmarkCcloudCurl(b *testing.B) {
	for i := 0; i < b.N; i++ {
		cmd := exec.Command("curl", "http://127.0.0.1:8000/submit?tmpl=qqnews")
		runCmd(cmd, b)
	}
}

func BenchmarkCcloudHttpClient(b *testing.B) {
	for i := 0; i < b.N; i++ {
		resp, err := httpClient.Get("http://127.0.0.1:8000/submit?tmpl=qqnews")
		if err != nil {
			b.Error(err)
		}
		defer resp.Body.Close()
		ioutil.ReadAll(resp.Body)

	}
}

func BenchmarkCcloudCurl10(b *testing.B) {
	for i := 0; i < b.N; i++ {
		for j := 0; j < 10; j++ {
			cmd := exec.Command("curl", "http://127.0.0.1:8000/submit?tmpl=qqnews")
			runCmd(cmd, b)
		}
	}
}

func BenchmarkCcloudHttpClient10(b *testing.B) {
	for i := 0; i < b.N; i++ {
		for j := 0; j < 10; j++ {
			resp, err := httpClient.Get("http://127.0.0.1:8000/submit?tmpl=qqnews")
			if err != nil {
				b.Error(err)
			}
			defer resp.Body.Close()
			ioutil.ReadAll(resp.Body)
		}
	}
}
