package caspercloud

import (
	"archive/zip"
	"encoding/json"
	"github.com/PuerkitoBio/goquery"
	"github.com/xlvector/dlog"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"io/ioutil"
	"math/rand"
	"os"
	"strings"
	"bytes"
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
	ServerList []string `json:"server_list"`
	random     *rand.Rand
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
	ret.random = rand.New(rand.NewSource(time.Now().UnixNano()))
	return &ret
}

func (p *Analyzer) SendReq(req *ParseRequest) bool {
	for i := 0; i < kMaxTryCount; i++ {
		index := p.random.Intn(len(p.ServerList))
		conn, err := grpc.Dial(p.ServerList[index])
		if err != nil {
			dlog.Warn("dial server get error:%s", err.Error())
			time.Sleep(1 * time.Second)
			continue
		}
		if conn != nil {
			defer conn.Close()
		}

		client := NewParserClient(conn)

		reply, err := client.ProcessParseRequest(context.Background(), req)
		if err != nil {
			dlog.Warn("call get error:%s", err.Error())
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
			dlog.Warn("open file get error:%s", err.Error())
			continue
		}

		fd, err := ioutil.ReadAll(f)
		if err != nil {
			dlog.Warn("read file get error:%s", err.Error())
			continue
		}
		fdstr := string(fd)
		fdstr = strings.Trim(fdstr, " \n\r\t")

		if strings.HasSuffix(fn, ".zip") {
			req.IsZip = true
		}

		req.Data = append(req.Data, fdstr)
		req.DataMetaInfo = append(req.DataMetaInfo, p.getPathLastPart(fn))

		f.Close()
	}
	return p.SendReq(req)
}


func (p *Analyzer) ProcessZip(req *ParseRequest, body []byte) bool {
	zipReader, err := zip.NewReader(bytes.NewReader(body),int64(len(body)))
	if err != nil {
		dlog.Warn("unzip file get error: %v", err)
		return false
	}
	for _, f := range zipReader.File {
		rc, err := f.Open()
		fn := f.Name
		if err != nil {
			dlog.Warn("open file get error: %v", err)
			continue
		}
		defer rc.Close()

		data, err := ioutil.ReadAll(rc)
		if err != nil {
			dlog.Warn("read zip file get error: %v", err)
			continue
		}

		fdstr := string(data)
		fdstr = strings.Trim(fdstr, " \n\r\t")

		if strings.HasSuffix(fn, ".zip") {
			req.IsZip = true
		}
		req.Data = append(req.Data, fdstr)
		req.DataMetaInfo = append(req.DataMetaInfo, p.getPathLastPart(fn))
	}
	return p.SendReq(req)
}
