package caspercloud

import (
	"encoding/json"
	"github.com/PuerkitoBio/goquery"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
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

type MailProcessor struct {
}

func NewMailProcessor() *MailProcessor {
	return &MailProcessor{}
}

func (p *MailProcessor) postData(data string) bool {
	params := url.Values{}
	params.Set("data", data)
	response, err := http.PostForm(kParserServer, params)
	if err != nil || response == nil {
		log.Println("do request get error:", err.Error(), " response:", response)
		return false
	}
	defer response.Body.Close()

	body, _ := ioutil.ReadAll(response.Body)

	log.Println("|post result|", string(body))
	return true
}

func (p *MailProcessor) Process(metaInfo map[string]string, downloads []string) bool {
	var mails []string
	for _, fn := range downloads {
		f, err := os.Open(fn)
		if err != nil {
			log.Fatal("open file get error:", err.Error())
		}

		fd, err := ioutil.ReadAll(f)
		if err != nil {
			log.Fatal("read file get error:", err.Error())
		}
		mails = append(mails, string(fd))
		f.Close()
	}
	htmls, err := json.Marshal(mails)
	if err != nil {
		log.Fatal("marshal mails get err:", err.Error())
	}
	metaInfo["row_html"] = string(htmls)
	metaInfo["row_html_len"] = strconv.FormatInt(int64(len(htmls)), 10)

	data, err := json.Marshal(metaInfo)
	if err != nil {
		log.Fatal("marshal metainfo get error:", err.Error())
	}
	p.postData(string(data))
	return true
}
