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
	"sync/atomic"
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

type Analyzer struct {
	ServerList   []string `json:"server_list"`
	conns        []*grpc.ClientConn
	parseClients []ParserClient
	random       *rand.Rand
	statusSync   []int32
}

func NewAnalyzer(path string) *Analyzer {
	text, err := ioutil.ReadFile(path)
	if err != nil {
		dlog.Warn("read %s get error:%s", path, err.Error())
		return nil
	}
	ret := Analyzer{}
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
		ret.statusSync = append(ret.statusSync, 0)
	}

	ret.random = rand.New(rand.NewSource(time.Now().UnixNano()))
	return &ret
}

func (p *Analyzer) Close() {
	for _, c := range p.conns {
		if c != nil {
			c.Close()
		}
	}
}

func (p *Analyzer) recoverClient(index int) {
	if index > len(p.ServerList) {
		return
	}
	if p.conns[index] != nil {
		p.conns[index].Close()
	}

	p.conns[index], _ = grpc.Dial(p.ServerList[index])
	p.parseClients[index] = NewParserClient(p.conns[index])
}

func (p *Analyzer) sendReq(req *ParseRequest) bool {
	for i := 0; i < kMaxTryCount; i++ {
		index := p.random.Intn(len(p.parseClients))
		reply, err := p.parseClients[index].ProcessParseRequest(context.Background(), req)
		if err != nil {
			dlog.Warn("call get error:%s", err.Error())
			if atomic.CompareAndSwapInt32(&p.statusSync[index], 0, 1) {
				p.recoverClient(index)
				atomic.CompareAndSwapInt32(&p.statusSync[index], 1, 0)
			}
			time.Sleep(1 * time.Second)
			continue
		}
		dlog.Println("get server reply:", *reply)
		return true
	}
	return false
}

func (p *Analyzer) getPathLastPart(path string) string {
	segs := strings.Split(path, "/")
	if len(segs) >= 1 {
		return segs[len(segs)-1]
	}
	return path
}

func (p *Analyzer) Process(req *ParseRequest, downloads []string) bool {
	for _, fn := range downloads {

		f, err := os.Open(fn)
		if err != nil {
			dlog.Fatal("open file get error:%s", err.Error())
		}

		fd, err := ioutil.ReadAll(f)
		if err != nil {
			dlog.Fatal("read file get error:%s", err.Error())
		}

		req.Data = append(req.Data, string(fd))
		req.DataMetaInfo = append(req.DataMetaInfo, p.getPathLastPart(fn))

		f.Close()
	}
	return p.sendReq(req)
}
