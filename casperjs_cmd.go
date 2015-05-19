package caspercloud

import (
	"bufio"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/xlvector/dlog"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"time"
)

const (
	kFetchStarted  = "started"
	kFetchFinished = "finished"
	kFetchFailed   = "failed"
)

type CasperCmd struct {
	proxyServer string
	id          string
	tmpl        string
	userName    string
	userId      string
	passWord    string
	message     chan *Output
	input       chan map[string]string
	isKill      bool
	isFinish    bool
	args        map[string]string
	privateKey  *rsa.PrivateKey
	analyzer    *Analyzer
}

type CasperCmdFactory struct{}

func (s *CasperCmdFactory) CreateCommand(params url.Values) Command {
	tmpl := params.Get("tmpl")
	userid := params.Get("userid")
	ret := &CasperCmd{
		proxyServer: "",
		id:          fmt.Sprintf("%s_%d", tmpl, time.Now().UnixNano()),
		tmpl:        tmpl,
		userName:    "",
		userId:      userid,
		passWord:    "",
		message:     make(chan *Output, 5),
		input:       make(chan map[string]string, 5),
		args:        make(map[string]string),
		isKill:      false,
		isFinish:    false,
		analyzer:    NewAnalyzer("server_list.json"),
	}
	var err error
	ret.privateKey, err = GenerateRSAKey()
	if err != nil {
		dlog.Fatalln("fail to generate rsa key", err)
	}
	go ret.run()
	return ret
}

func (s *CasperCmdFactory) CreateCommandWithPrivateKey(params url.Values, pk *rsa.PrivateKey) Command {
	tmpl := params.Get("tmpl")
	userid := params.Get("userid")
	ret := &CasperCmd{
		proxyServer: "",
		id:          fmt.Sprintf("%s_%d", tmpl, time.Now().UnixNano()),
		tmpl:        tmpl,
		userName:    "",
		userId:      userid,
		passWord:    "",
		message:     make(chan *Output, 5),
		input:       make(chan map[string]string, 5),
		args:        make(map[string]string),
		isKill:      false,
		isFinish:    false,
		analyzer:    NewAnalyzer("server_list.json"),
		privateKey:  pk,
	}
	go ret.run()
	return ret
}

func (self *CasperCmd) GetId() string {
	return self.id
}

func (self *CasperCmd) SetInputArgs(input map[string]string) {
	if self.Finished() {
		dlog.Warn("start another casperjs")
		go self.run()
	}
	self.input <- input
}

func (self *CasperCmd) GetMessage() *Output {
	return <-self.message
}

func (self *CasperCmd) readInputArgs(key string) string {
	args := <-self.input
	for k, v := range args {
		if k == "username" {
			self.userName = v
		}

		if k == "password" {
			self.passWord = v
		}

		self.args[k] = v
	}
	if val, ok := self.args[key]; ok {
		return val
	}

	message := &Output{
		Id:        self.GetArgsValue("id"),
		NeedParam: key,
		Status:    NEED_PARAM,
	}
	dlog.Warn("need param:%s", key)
	self.message <- message
	return ""
}

func (self *CasperCmd) GetArgsValue(key string) string {
	if val, ok := self.args[key]; ok {
		dlog.Info("successfully get args value:%s", val)
		return val
	}
	for {
		val := self.readInputArgs(key)
		if len(val) != 0 {
			dlog.Info("successfully get args value:%s", val)
			return val
		}
	}

}

func (self *CasperCmd) getArgsList(args string) []string {
	segs := strings.Split(args, "/")
	if len(segs) < 2 {
		return nil
	}
	return segs[1:]
}

func (self *CasperCmd) GetParseReq(fetchStatus string) *ParseRequest {
	ret := &ParseRequest{}
	ret.PrivateKey = string(PrivateKeyString(self.privateKey))
	ret.PublicKey = string(PublicKeyString(&self.privateKey.PublicKey))
	ret.Tmpl = self.tmpl
	ret.FetchStatus = fetchStatus
	ret.UserName = self.userName
	ret.Secret = self.passWord
	if len(self.userId) > 0 {
		ret.RowKey = self.tmpl + "|" + self.userId + "|" + self.userName
	} else {
		ret.RowKey = self.tmpl + "|" + self.userName
	}

	ret.ReqType = ParseRequestType_Html

	// harder code(Todo refact)
	switch {
	case self.tmpl == "taobao_shop":
		ret.ReqType = ParseRequestType_TaobaoShop
	case strings.HasPrefix(self.tmpl, "mail.com"):
		ret.ReqType = ParseRequestType_Eml
	}
	return ret
}

func (self *CasperCmd) Successed() bool {
	return true
}

func (self *CasperCmd) Finished() bool {
	return self.isKill || self.isFinish
}

func DecodePassword(p string, privateKey *rsa.PrivateKey) string {
	bp, err := hex.DecodeString(p)
	if err != nil {
		dlog.Warn("decode password hex error:%s", err.Error())
		return ""
	}
	out, err := rsa.DecryptOAEP(sha256.New(), rand.Reader, privateKey,
		bp, []byte(""))
	if err != nil {
		dlog.Warn("decode password error:%s", err.Error())
		return ""
	}
	return string(out)
}

func (self *CasperCmd) run() {
	dlog.Info("begin run cmd:%s", self.tmpl)
	self.isFinish = false
	self.isKill = false

	path := "./" + self.tmpl + "/" + self.id
	os.RemoveAll(path)
	if err := os.MkdirAll(path, 0755); err != nil {
		dlog.Fatalln("can not create", path, err)
	}

	cookieFile, err := os.Create(path + "/cookie.txt")
	defer cookieFile.Close()
	var cmd *exec.Cmd
	if len(self.proxyServer) == 0 {
		cmd = exec.Command("casperjs", self.tmpl+".js",
			"--ignore-ssl-errors=true",
			"--web-security=no",
			"--cookies-file="+path+"/cookie.txt",
			"--context="+path)
	} else {
		cmd = exec.Command("casperjs", self.tmpl+".js",
			"--ignore-ssl-errors=true",
			"--web-security=no",
			"--cookies-file="+path+"/cookie.txt",
			"--proxy="+self.proxyServer, "--proxy-type=http",
			"--context="+path)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		dlog.Panic("can not get stdout pipe:%s", err.Error())
	}
	bufout := bufio.NewReader(stdout)

	stdin, err := cmd.StdinPipe()
	if err != nil {
		dlog.Panic("can not get stdin pipe:%s", err.Error())
	}
	bufin := bufio.NewWriter(stdin)

	if err := cmd.Start(); err != nil {
		dlog.Panic("can not start cmd:%s", err.Error())
	}

	go func() {
		timer := time.NewTimer(5 * time.Minute)
		<-timer.C
		cmd.Process.Kill()
		self.isKill = true
	}()

	dlog.Info("begin read line from capser")
	for {
		line, err := bufout.ReadString('\n')
		line = strings.Trim(line, "\n")
		if err != nil {
			dlog.Error("read stdin get error:%s", err.Error())
			cmd.Process.Wait()
			cmd.Process.Kill()
			break
		}

		dlog.Info("scripts debug info:%s", line)

		if strings.HasPrefix(line, "CMD INFO STARTED") {
			message := &Output{
				Id:     self.GetArgsValue("id"),
				Status: OUTPUT_PUBLICKEY,
				Data:   string(PublicKeyString(&self.privateKey.PublicKey)),
			}
			self.message <- message
			continue
		}

		if strings.HasPrefix(line, "CMD GET ARGS") {
			for _, v := range self.getArgsList(line) {
				key := strings.TrimRight(v, "\n")
				val := self.GetArgsValue(key)
				if key == "password" {
					val = DecodePassword(val, self.privateKey)
				}
				if key == "username" && self.analyzer != nil {
					req := self.GetParseReq(kFetchStarted)
					self.analyzer.sendReq(req)
					dlog.Info("report status started:%s", req.RowKey)
				}
				bufin.WriteString(val)
				delete(self.args, key)
				bufin.WriteRune('\n')
				bufin.Flush()
			}
			continue
		}

		if strings.HasPrefix(line, "CMD INFO LOGIN SUCCESS") {
			message := &Output{
				Id:     self.GetArgsValue("id"),
				Status: LOGIN_SUCCESS,
			}
			self.message <- message
			continue
		}

		if strings.HasPrefix(line, "CMD NEED") {
			result := strings.TrimPrefix(line, "CMD NEED")
			result = strings.TrimSpace(result)
			message := &Output{
				Id:        self.GetArgsValue("id"),
				Status:    NEED_PARAM,
				NeedParam: result,
			}
			self.message <- message
			continue
		}

		if strings.HasPrefix(line, "CMD INFO RANDCODE") {
			result := strings.TrimPrefix(line, "CMD INFO RANDCODE")
			result = strings.Trim(result, " \n")
			result = UploadImage("./site/" + result)
			dlog.Info("success upload captcha image to:%s", result)
			message := &Output{
				Id:        self.GetArgsValue("id"),
				Status:    OUTPUT_VERIFYCODE,
				Data:      result,
				NeedParam: PARAM_VERIFY_CODE,
			}
			self.message <- message
			continue
		}

		if strings.HasPrefix(line, "CMD INFO CONTENT") {
			message := &Output{
				Status: strings.TrimSpace(strings.TrimPrefix(line,
					"CMD INFO CONTENT")),
				Id: self.GetArgsValue("id"),
			}
			self.message <- message
			continue
		}

		if strings.HasPrefix(line, "CMD INFO FETCHED MAIL") {
			result := strings.TrimSpace(strings.TrimPrefix(line,
				"CMD INFO FETCHED MAIL"))
			var out CasperOutput
			json.Unmarshal([]byte(result), &out)
			if self.analyzer != nil {
				req := self.GetParseReq(kFetchFinished)
				dlog.Info("fetch finished:%s", req.RowKey)
				go self.analyzer.Process(req, out.Downloads)
			}
			message := &Output{
				Status: FINISH_FETCH_DATA,
				Id:     self.GetArgsValue("id"),
			}
			self.message <- message
			continue
		}

		if strings.HasPrefix(line, "CMD FAIL") {
			if self.analyzer != nil {
				req := self.GetParseReq(kFetchFailed)
				dlog.Info("fetch failed:%s", req.RowKey)
				go self.analyzer.sendReq(req)
			}
			message := &Output{
				Status: FAIL,
				Id:     self.GetArgsValue("id"),
			}
			self.message <- message
			cmd.Process.Wait()
			cmd.Process.Kill()
			break
		}
	}
	self.isFinish = true
}
