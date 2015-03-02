package main

import (
	"io/ioutil"
	"log"
	"net/http"
	"runtime"
	"strconv"
	"time"
)

const (
	kUrlWithProxy = "http://127.0.0.1:8000/submit?proxy=127.0.0.1:7182&tmpl=gfsoso&_word="
	kUrlNoProxy   = "http://127.0.0.1:8000/submit?tmpl=gfsoso&_word="
)

func TestSpeed(withProxy bool) {
	allTime := int64(0)
	client := &http.Client{}
	log.Println("start to test")
	for i := int64(15201007026); i < 15201007026+1000; i++ {
		url := ""
		if withProxy {
			url = kUrlWithProxy + strconv.FormatInt(i, 10)
		} else {
			url = kUrlNoProxy + strconv.FormatInt(i, 10)
		}
		startTime := time.Now().UnixNano()
		resp, _ := client.Get(url)
		if resp == nil {
			log.Println("get http response error")
			continue
		}
		if resp.Body != nil {
			defer resp.Body.Close()
		}
		content, _ := ioutil.ReadAll(resp.Body)
		log.Println("get content:", string(content))

		endTime := time.Now().UnixNano()
		allTime += (endTime - startTime)
		time.Sleep(1 * time.Second)
	}
	log.Println("end with proxy,  spent time:", allTime)
}

func main() {
	runtime.GOMAXPROCS(2)
	TestSpeed(false)
	TestSpeed(true)
}
