package caspercloud

import (
	"bytes"
	"crypto/tls"
	"github.com/PuerkitoBio/goquery"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"time"
)

const (
	kParserServer = "http://parser.crawler.bdp.cc/submit"
)

type Mail struct {
	From  string `json:"from"`
	Title string `json:"title"`
}

type CasperOutput struct {
	Downloads []string `json:"downloads"`
	Mails     []Mail   `json:"mails"`
}

func LoadDownloads(fs []string) {
	for _, fn := range fs {
		ParseFile(fn)
	}
}

func ParseFile(fn string) error {
	f, err := os.Open(fn)
	if err != nil {
		log.Println("fail to load file:", err)
		return err
	}
	defer f.Close()

	doc, err := goquery.NewDocumentFromReader(f)
	if err != nil {
		log.Println("fail to get dom:", err)
		return err
	}
	log.Println("file length", len(doc.Text()))
	return nil
}

func newHttpClient(timeOutSeconds int) *http.Client {
	client := &http.Client{
		Transport: &http.Transport{
			Dial: func(netw, addr string) (net.Conn, error) {
				timeout := time.Duration(timeOutSeconds) * time.Second
				deadline := time.Now().Add(timeout)
				c, err := net.DialTimeout(netw, addr, timeout)
				if err != nil {
					return nil, err
				}
				c.SetDeadline(deadline)
				return c, nil
			},
			DisableKeepAlives:     true,
			ResponseHeaderTimeout: time.Duration(timeOutSeconds) * time.Second,
			TLSClientConfig:       &tls.Config{InsecureSkipVerify: true},
		},
	}
	return client
}

type MailProcessor struct {
	client *http.Client
}

func NewMailProcessor() *MailProcessor {
	c := newHttpClient(20)
	if c == nil {
		return nil
	}

	return &MailProcessor{
		client: c,
	}
}

func (mailProcessor *MailProcessor) deal(info map[string]string, path string) bool {

	f, err := os.Open(path)
	if err != nil {
		log.Println("open file failed:", err.Error())
	}
	defer f.Close()

	fd, err := ioutil.ReadAll(f)
	if err != nil {
		log.Println("read file get error:", err.Error())
	}

	/*
		msg, err := mail.ReadMessage(f)
		if err != nil {
			log.Println("open file get error:", err.Error())
			return false
		}

		// TODO add header info
		// msg.Header

		body, err := enmime.ParseMIMEBody(msg)
		if err != nil {
			return false
		}
	*/

	info["row_html"] = string(fd)

	params := url.Values{}
	for key, value := range info {
		params.Set(key, value)
	}
	reqest, err := http.NewRequest("POST", kParserServer, bytes.NewReader([]byte(params.Encode())))
	if err != nil {
		log.Println("new request get error:", err.Error())
		return false
	}

	response, err := mailProcessor.client.Do(reqest)
	if err != nil || response == nil {
		log.Println("do request get error:", err.Error(), " response:", response)
		return false
	}

	log.Println("|path|", path, "|post result|", *response)
	return true

}

func (p *MailProcessor) Process(metaInfo map[string]string, downloads []string) bool {
	for _, fn := range downloads {
		p.deal(metaInfo, fn)
	}
	return true
}
