package caspercloud

import (
	"encoding/json"
	"github.com/PuerkitoBio/goquery"
	"github.com/xlvector/dlog"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"io/ioutil"
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
		dlog.Warn("fail to load file:%s", err.Error())
		return err
	}
	defer f.Close()

	doc, err := goquery.NewDocumentFromReader(f)
	if err != nil {
		dlog.Warn("fail to get dom:%s", err.Error())
		return err
	}
	dlog.Info("file length:%d", len(doc.Text()))
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
		dlog.Warn("read %s get error:%s", path, err.Error())
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
			dlog.Warn("call get error:%s", err.Error())
			time.Sleep(1 * time.Second)
			p.recoverClient(index)
			continue
		}
		dlog.Println("get server reply:", *reply)
		return true
	}
	return false
}

func (p *MailProcessor) Process(req *ParseRequest, downloads []string) bool {
	req.ReqType = ParseRequestType_Html
	for _, fn := range downloads {
		f, err := os.Open(fn)
		if err != nil {
			dlog.Fatal("open file get error:%s", err.Error())
		}

		fd, err := ioutil.ReadAll(f)
		if err != nil {
			dlog.Fatal("read file get error:%s", err.Error())
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
