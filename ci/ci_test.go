package ci

import (
	"bufio"
	"encoding/json"
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

const (
	ENDPOINT = "http://127.0.0.1:8000"
)

func init() {
	go runMockSite()
	client = &http.Client{}
}

func runMockSite() {
	service := caspercloud.NewCasperServer()
	http.Handle("/submit", service)
	l, e := net.Listen("tcp", ":8000")
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
		cmd := exec.Command("curl", ENDPOINT+"/hello")
		runCmd(cmd, b, false)
	}
}

func BenchmarkHttpClient(b *testing.B) {
	for i := 0; i < b.N; i++ {
		resp, err := client.Get(ENDPOINT + "/hello")
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
		cmd := exec.Command("curl", ENDPOINT+"/hello?query=[1-100]")
		runCmd(cmd, b, false)
	}
}

func BenchmarkHttpClient100(b *testing.B) {
	for i := 0; i < b.N; i++ {
		for j := 0; j < 100; j++ {
			resp, err := client.Get(ENDPOINT + "/hello")
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
		cmd := exec.Command("phantomjs", "loadspeed.js", ENDPOINT+"/hello", "1")
		runCmd(cmd, b, true)
	}
}

func TestHello(t *testing.T) {
	for i := 0; i < 5; i++ {
		go func() {
			queries := []string{"sina", "twitter", "xlvector", "bigtong"}
			for _, q := range queries {
				time.Sleep(time.Millisecond * 50)
				ret := getJson(ENDPOINT + "/submit?tmpl=hello&_query=" + q)
				if v, _ := ret["result"]; v != q {
					t.Error(v)
				}
			}
		}()
	}
	time.Sleep(time.Second * 5)
}

func TestForm(t *testing.T) {
	for i := 0; i < 5; i++ {
		go func() {
			ret := getJson(ENDPOINT + "/submit?tmpl=form")
			id, ok := ret["id"]
			if !ok {
				t.Error("can not find id in result")
				return
			}
			ret = getJson(ENDPOINT + "/submit?tmpl=form&_phone=18612345678&id=" + id)
			log.Println(ret)
			ret = getJson(ENDPOINT + "/submit?tmpl=form&_verify_code=123456&id=" + id)
			log.Println(ret)
			if ret["result"] != "Thanks" {
				t.Error("result not right:" + ret["result"])
			}
		}()
	}
	time.Sleep(time.Second * 5)
}
