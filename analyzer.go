package caspercloud

import (
	"encoding/json"
	"github.com/PuerkitoBio/goquery"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"io/ioutil"
	"log"
	"math/rand"
	"os"
	"strings"
	"time"
)

const (
	kMaxTryCount = 5
)

type Mail struct {
	From  string `json:"from"`
	Title string `json:"title"`
}

type CasperOutput struct {
	Downloads []string `json:"downloads"`
	Mails     []Mail   `json:"mails"`
	Status    string   `json:"status"`
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
	ServerList   []string `json:"server_list"`
	conns        []*grpc.ClientConn
	parseClients []ParserClient
	random       *rand.Rand
}

func NewMailProcessor(path string) *MailProcessor {
	text, err := ioutil.ReadFile(path)
	if err != nil {
		log.Println(err)
		return nil
	}
	ret := MailProcessor{}
	err = json.Unmarshal(text, &ret)
	if err != nil {
		panic(err)
	}

	if len(ret.ServerList) == 0 {
		return nil
	}

	for _, addr := range ret.ServerList {
		conn, _ := grpc.Dial(addr)
		ret.conns = append(ret.conns, conn)
		ret.parseClients = append(ret.parseClients, NewParserClient(conn))
	}

	ret.random = rand.New(rand.NewSource(time.Now().UnixNano()))
	return &ret
}

func (p *MailProcessor) Close() {
	for _, c := range p.conns {
		if c != nil {
			c.Close()
		}
	}
}

func (p *MailProcessor) recoverClient(index int) {
	if index > len(p.ServerList) {
		return
	}
	if p.conns[index] != nil {
		p.conns[index].Close()
	}

	p.conns[index], _ = grpc.Dial(p.ServerList[index])
	p.parseClients[index] = NewParserClient(p.conns[index])
}

func (p *MailProcessor) sendReq(req *ParseRequest) bool {
	for i := 0; i < kMaxTryCount; i++ {
		index := p.random.Intn(len(p.parseClients))
		reply, err := p.parseClients[index].ProcessParseRequest(context.Background(), req)
		if err != nil {
			log.Println("call get error:", err.Error())
			time.Sleep(1 * time.Second)
			p.recoverClient(index)
			continue
		}
		log.Println("get server reply:", *reply)
		return true
	}
	return false
}

func (p *MailProcessor) Process(req *ParseRequest, downloads []string) bool {
	req.ReqType = ParseRequestType_Html
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
			req.ReqType = ParseRequestType_Eml
			req.IsZip = true
		}

		if strings.HasSuffix(fn, ".eml") {
			req.ReqType = ParseRequestType_Eml
		}

		req.Data = append(req.Data, string(fd))
		f.Close()
	}
	return p.sendReq(req)
}
