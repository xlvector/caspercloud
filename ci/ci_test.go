package ci

import (
	"bufio"
	"encoding/json"
	"fmt"
	"github.com/xlvector/caspercloud"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os/exec"
	"testing"
	"time"
)

var client *http.Client

func init() {
	go runMockSite()
	client = &http.Client{}
}

func runMockSite() {
	http.HandleFunc("/hello", func(w http.ResponseWriter, req *http.Request) {
		html := "<html><head><title>Hello World</title></head><body><h1>Hello World</h1></body></html>\n"
		time.Sleep(100 * time.Millisecond)
		fmt.Fprint(w, html)
	})
	service := caspercloud.NewCasperServer()
	http.Handle("/submit", service)
	l, e := net.Listen("tcp", ":20893")
	if e != nil {
		log.Fatal("listen error:", e)
	}
	http.Serve(l, nil)
}

func getJson(link string) map[string]string {
	resp, err := client.Get(link)
	if err != nil {
		log.Println("fail to get resp")
		return nil
	}
	defer resp.Body.Close()
	out, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println("fail to read output")
		return nil
	}
	ret := make(map[string]string)
	json.Unmarshal(out, &ret)
	return ret
}

func runCmd(cmd *exec.Cmd, b *testing.B, info bool) {
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Panicln("can not get stdout pipe:", err)
	}
	bufout := bufio.NewReader(stdout)
	err = cmd.Start()
	if err != nil {
		b.Error("fail to run curl")
	}
	for {
		line, err := bufout.ReadString('\n')
		if err != nil {
			break
		}
		if info {
			log.Println(line)
		}
	}
	cmd.Wait()
}

func BenchmarkCurl(b *testing.B) {
	for i := 0; i < b.N; i++ {
		cmd := exec.Command("curl", "http://127.0.0.1:20893/hello")
		runCmd(cmd, b, false)
	}
}

func BenchmarkHttpClient(b *testing.B) {
	for i := 0; i < b.N; i++ {
		resp, err := client.Get("http://127.0.0.1:20893/hello")
		if err != nil {
			b.Error(err)
		}
		defer resp.Body.Close()
		ioutil.ReadAll(resp.Body)
	}
}

func BenchmarkCasperJs(b *testing.B) {
	for i := 0; i < b.N; i++ {
		cmd := exec.Command("casperjs", "mock.js")
		runCmd(cmd, b, false)
	}
}

func BenchmarkCurl100(b *testing.B) {
	for i := 0; i < b.N; i++ {
		cmd := exec.Command("curl", "http://127.0.0.1:20893/hello?[1-100]")
		runCmd(cmd, b, false)
	}
}

func BenchmarkHttpClient100(b *testing.B) {
	for i := 0; i < b.N; i++ {
		for j := 0; j < 100; j++ {
			resp, err := client.Get("http://127.0.0.1:20893/hello")
			if err != nil {
				b.Error(err)
			}
			defer resp.Body.Close()
			ioutil.ReadAll(resp.Body)
		}
	}
}

func BenchmarkCasperJs100(b *testing.B) {
	for i := 0; i < b.N; i++ {
		cmd := exec.Command("casperjs", "mock_100.js")
		runCmd(cmd, b, false)
	}
}

func BenchmarkPhantomJs(b *testing.B) {
	for i := 0; i < b.N; i++ {
		cmd := exec.Command("phantomjs", "loadspeed.js", "http://127.0.0.1:20893/hello", "1")
		runCmd(cmd, b, true)
	}
}

func TestBaidu(t *testing.T) {
	ret := getJson("http://127.0.0.1:20893/submit?tmpl=baidu")
	id, ok := ret["id"]
	if !ok {
		t.Error("can not find id")
	}
	ret = getJson("http://127.0.0.1:20893/submit?tmpl=baidu&id=" + id + "&_query=google")
	log.Println(ret)
	ret = getJson("http://127.0.0.1:20893/submit?tmpl=baidu&id=" + id + "&_query=sina")
	log.Println(ret)
	ret = getJson("http://127.0.0.1:20893/submit?tmpl=baidu&id=" + id + "&_query=golang")
	log.Println(ret)
	ret = getJson("http://127.0.0.1:20893/submit?tmpl=baidu&id=" + id + "&_query=weibo")
	log.Println(ret)
	ret = getJson("http://127.0.0.1:20893/submit?tmpl=baidu&id=" + id + "&_query=xlvector")
	log.Println(ret)
}
