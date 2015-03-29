package caspercloud

import (
	"bufio"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"time"
)

type CasperCmd struct {
	proxyServer   string
	id            string
	tmpl          string
	userName      string
	passWord      string
	message       chan *Output
	input         chan map[string]string
	isKill        bool
	isFinish      bool
	args          map[string]string
	privateKey    *rsa.PrivateKey
	mailProcessor *MailProcessor
}

type CasperCmdFactory struct{}

func (s *CasperCmdFactory) CreateCommand(params url.Values) Command {
	tmpl := params.Get("tmpl")
	ret := &CasperCmd{
		proxyServer:   "",
		id:            fmt.Sprintf("%s_%d", tmpl, time.Now().UnixNano()),
		tmpl:          tmpl,
		userName:      "",
		passWord:      "",
		message:       make(chan *Output, 5),
		input:         make(chan map[string]string, 5),
		args:          make(map[string]string),
		isKill:        false,
		isFinish:      false,
		mailProcessor: NewMailProcessor("server_list.json"),
	}
	var err error
	ret.privateKey, err = GenerateRSAKey()
	if err != nil {
		log.Fatalln("fail to generate rsa key", err)
	}
	go ret.run()
	return ret
}

func (self *CasperCmd) GetId() string {
	return self.id
}

func (self *CasperCmd) SetInputArgs(input map[string]string) {
	if self.Finished() {
		log.Println("start another casperjs")
		go self.run()
	}
	log.Println("insert input:", input)
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
		log.Println("find param", key, val)
		return val
	}

	message := &Output{
		Id:        self.GetArgsValue("id"),
		NeedParam: key,
		Status:    NEED_PARAM,
	}
	log.Println("need param", key)
	self.message <- message
	return ""
}

func (self *CasperCmd) GetArgsValue(key string) string {
	if val, ok := self.args[key]; ok {
		log.Println("successfully get args value", val)
		return val
	}
	for {
		val := self.readInputArgs(key)
		if len(val) != 0 {
			log.Println("successfully get args value", val)
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

func (self *CasperCmd) getMetaInfo() *ParseRequest {
	ret := &ParseRequest{}
	ret.PrivateKey = string(PrivateKeyString(self.privateKey))
	ret.PublicKey = string(PublicKeyString(&self.privateKey.PublicKey))
	ret.Tmpl = self.tmpl
	ret.UserName = self.userName
	ret.Secret = self.passWord
	ret.RowKey = self.tmpl + "|" + self.userName
	return ret
}

func (self *CasperCmd) Finished() bool {
	return self.isKill || self.isFinish
}

func DecodePassword(p string, privateKey *rsa.PrivateKey) string {
	bp, err := hex.DecodeString(p)
	if err != nil {
		log.Println("decode password hex error:", err)
		return ""
	}
	out, err := rsa.DecryptOAEP(sha256.New(), rand.Reader, privateKey,
		bp, []byte(""))
	if err != nil {
		log.Println("decode password error:", err)
		return ""
	}
	log.Println("decode password:", string(out))
	return string(out)
}

func (self *CasperCmd) run() {
	log.Println("begin run cmd", self.tmpl)
	self.isFinish = false
	self.isKill = false

	path := "./" + self.tmpl + "/" + self.id
	os.RemoveAll(path)
	if err := os.MkdirAll(path, 0755); err != nil {
		log.Fatalln("can not create", path, err)
	}

	cookieFile, err := os.Create(path + "/cookie.txt")
	defer cookieFile.Close()
	var cmd *exec.Cmd
	if len(self.proxyServer) == 0 {
		cmd = exec.Command("casperjs", self.tmpl+".js",
			"--web-security=no",
			"--cookies-file="+path+"/cookie.txt",
			"--context="+path)
	} else {
		cmd = exec.Command("casperjs", self.tmpl+".js",
			"--web-security=no",
			"--cookies-file="+path+"/cookie.txt",
			"--proxy="+self.proxyServer, "--proxy-type=http",
			"--context="+path)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Panicln("can not get stdout pipe:", err)
	}
	bufout := bufio.NewReader(stdout)

	stdin, err := cmd.StdinPipe()
	if err != nil {
		log.Panicln("can not get stdin pipe:", err)
	}
	bufin := bufio.NewWriter(stdin)

	if err := cmd.Start(); err != nil {
		log.Panicln("can not start cmd:", err)
	}

	go func() {
		timer := time.NewTimer(5 * time.Minute)
		<-timer.C
		cmd.Process.Kill()
		self.isKill = true
	}()

	log.Println("begin read line from capser")
	for {
		line, err := bufout.ReadString('\n')
		line = strings.Trim(line, "\n")
		if err != nil {
			log.Println(err)
			cmd.Process.Wait()
			cmd.Process.Kill()
			break
		}
		log.Println(line)

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
			log.Println("success upload captcha image to", result)
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
			if self.mailProcessor != nil {
				go self.mailProcessor.Process(self.getMetaInfo(), out.Downloads)
			}
			message := &Output{
				Status: FINISH_FETCH_DATA,
				Id:     self.GetArgsValue("id"),
			}
			self.message <- message
			continue
		}

		if strings.HasPrefix(line, "CMD FAIL") {
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
