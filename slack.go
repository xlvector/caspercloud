package caspercloud

import (
	"io/ioutil"
	"log"
	"net/http"
	"strings"
)

func PostDataToSlack(msg, channel string) bool {
	resp, err := http.Post("https://crediteasebdp.slack.com/services/hooks/slackbot?token=ZX2bGIhBvC06TfxWVtXqEypH&channel=%23"+channel,
		"text/plain;charset=UTF-8", strings.NewReader(msg))
	if err != nil {
		log.Println("fail to post slack", err)
		return false
	}
	defer resp.Body.Close()
	buf, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println("fail to post slack", err)
		return false
	}
	ret := string(buf)
	if ret == "ok" {
		return true
	}
	return false
}
