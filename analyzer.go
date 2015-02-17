package caspercloud

import (
	"github.com/PuerkitoBio/goquery"
	"log"
	"os"
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
