package caspercloud

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"encoding/json"
	"github.com/PuerkitoBio/goquery"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
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
	buf := bytes.NewBuffer(nil)
	w := gzip.NewWriter(buf)
	defer w.Close()

	if _, err := w.Write([]byte(data)); err != nil {
		log.Println("gzip compress err:", err)
	}
	w.Flush()

	params := url.Values{}
	params.Set("data", string(buf.Bytes()))
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
	isZip := false
	for _, fn := range downloads {
		f, err := os.Open(fn)
		if err != nil {
			log.Fatal("open file get error:", err.Error())
		}

		fd, err := ioutil.ReadAll(f)
		if err != nil {
			log.Fatal("read file get error:", err.Error())
		}
		if strings.HasSuffix(fn, ".zip") {
			isZip = true
			mails = append(mails, base64.StdEncoding.EncodeToString(fd))
		} else {
			mails = append(mails, string(fd))
		}

		f.Close()
	}
	htmls, err := json.Marshal(mails)
	if err != nil {
		log.Fatal("marshal mails get err:", err.Error())
	}
	metaInfo["raw_html"] = string(htmls)

	if isZip {
		metaInfo["is_zip"] = "true"
	} else {
		metaInfo["is_zip"] = "false"
	}

	data, err := json.Marshal(metaInfo)
	if err != nil {
		log.Fatal("marshal metainfo get error:", err.Error())
	}
	p.postData(string(data))
	return true
}
