package caspercloud

import (
	"bufio"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"log"
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
	message       chan map[string]interface{}
	input         chan map[string]string
	isKill        bool
	isFinish      bool
	args          map[string]string
	status        int
	privateKey    *rsa.PrivateKey
	mailProcessor *MailProcessor
}

func NewCasperCmd(id, tmpl, proxyServer string) *CasperCmd {
	ret := &CasperCmd{
		proxyServer:   proxyServer,
		id:            id,
		tmpl:          tmpl,
		userName:      "",
		passWord:      "",
		message:       make(chan map[string]interface{}, 1),
		input:         make(chan map[string]string, 1),
		args:          make(map[string]string),
		status:        kCommandStatusIdle,
		isKill:        false,
		isFinish:      false,
		mailProcessor: NewMailProcessor(),
	}
	var err error
	ret.privateKey, err = generateRSAKey()
	if err != nil {
		log.Fatalln("fail to generate rsa key", err)
	}
	go ret.run()
	return ret
}

func (self *CasperCmd) GetId() string {
	return self.id
}

func (self *CasperCmd) GetStatus() int {
	return self.status
}

func (self *CasperCmd) SetInputArgs(input map[string]string) {
	if self.Finished() {
		log.Println("start another casperjs")
		go self.run()
	}
	log.Println("insert input:", input)
	self.input <- input
}

func (self *CasperCmd) GetMessage() map[string]interface{} {
	return <-self.message
}

func (self *CasperCmd) readInputArgs(key string) string {
	args := <-self.input
	log.Println("read args:", args)
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

	message := make(map[string]interface{})
	message["id"] = self.GetArgsValue("id")
	message["need_args"] = key
	message[kJobStatus] = kJobOndoing
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

func (self *CasperCmd) getMetaInfo() map[string]string {
	var metaInfo = make(map[string]string)
	metaInfo["private_key"] = string(privateKeyString(self.privateKey))
	metaInfo["tmpl"] = self.tmpl
	metaInfo["public_key"] = string(publicKeyString(&self.privateKey.PublicKey))
	metaInfo["username"] = self.userName
	metaInfo["password"] = self.passWord
	metaInfo["row_key"] = self.tmpl + "|" + self.userName
	return metaInfo
}

func (self *CasperCmd) Finished() bool {
	return self.isKill || self.isFinish
}

func (self *CasperCmd) DecodePassword(p string) string {
	bp, err := hex.DecodeString(p)
	if err != nil {
		log.Println("decode password hex error:", err)
		return ""
	}
	out, err := rsa.DecryptOAEP(sha256.New(), rand.Reader, self.privateKey,
		bp, []byte(""))
	if err != nil {
		log.Println("decode password error:", err)
		return ""
	}
	log.Println("decode password:", string(out))
	return string(out)
}

func (self *CasperCmd) getHostByTmpl() string {
	switch self.tmpl {
	case "mail_163":
		return "163.com"
	case "mail_qq":
		return "qq.com"
	case "mail_126":
		return "126.com"
	case "mail_139":
		return "139.com"
	case "mail_aliyun":
		return "aliyun.com"
	default:
		return ""
	}
	return ""
}

func (self *CasperCmd) tryPop3(line string) ([]string, error) {
	if strings.HasPrefix(line, "CMD GET ARGS") {
		req := strings.TrimPrefix(line, "CMD GET ARGS")
		req = strings.Trim(req, " \n\t")
		if req == "/username/password" {
			log.Println("try pop3")
			username := self.GetArgsValue("username")
			PostDataToSlack(username+" begin to try", "captcha")
			password := self.GetArgsValue("password")
			password = self.DecodePassword(password)
			log.Println("try pop3 use", username, password)
			return Pop3ReceiveMail(username+"@"+self.getHostByTmpl(), password)
		}
	}
	return nil, errors.New("invalid command")
}

func (self *CasperCmd) run() {
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
	start := false
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
			start = true
			message := map[string]interface{}{
				"public_key": string(publicKeyString(&self.privateKey.PublicKey)),
				"id":         self.GetArgsValue("id"),
			}
			self.message <- message
			continue
		}

		if strings.HasPrefix(line, "CMD GET ARGS") {
			_, err = self.tryPop3(line)
			if err == nil {
				message := map[string]interface{}{
					"id":     self.GetArgsValue("id"),
					"result": "pop3_success",
				}
				self.message <- message
				break
			} else {
				log.Println("try pop3 error: ", err)
			}
			for _, v := range self.getArgsList(line) {
				key := strings.TrimRight(v, "\n")
				val := self.GetArgsValue(key)
				if key == "password" {
					val = self.DecodePassword(val)
				}
				bufin.WriteString(val)
				delete(self.args, key)
				bufin.WriteRune('\n')
				bufin.Flush()
			}
			continue
		}

		if strings.HasPrefix(line, "CMD INFO LOGIN SUCCESS") {
			var out CasperOutput
			go self.mailProcessor.Process(self.getMetaInfo(), out.Downloads)
			continue
		}

		if strings.HasPrefix(line, "CMD INFO RANDCODE") {
			message := make(map[string]interface{})
			message["public_key"] = string(publicKeyString(&self.privateKey.PublicKey))
			message["id"] = self.GetArgsValue("id")
			result := strings.TrimPrefix(line, "CMD INFO RANDCODE")
			result = strings.Trim(result, " \n")
			result = UploadImage("./site/" + result)
			log.Println("success upload captcha image to", result)
			if PostDataToSlack(result, "captcha") {
				log.Println("success post captcha to slack")
			} else {
				log.Println("fail to post captcha to slack")
			}
			message["result"] = result
			message[kJobStatus] = kJobOndoing
			log.Println("send result:", message)
			self.message <- message
			continue
		}

		if strings.HasPrefix(line, "CMD INFO CONTENT") {
			message := make(map[string]interface{})
			message["public_key"] = string(publicKeyString(&self.privateKey.PublicKey))
			message["id"] = self.GetArgsValue("id")
			result := strings.TrimPrefix(line, "CMD INFO CONTENT")
			result = strings.Trim(result, " \n")
			message["result"] = result
			var out CasperOutput
			err := json.Unmarshal([]byte(result), &out)
			if err == nil {
				message["json"] = out
			}
			message[kJobStatus] = kJobFinished
			log.Println("send result:", message)
			go self.mailProcessor.Process(self.getMetaInfo(), out.Downloads)
			self.message <- message
			self.status = kCommandStatusIdle
			start = false
			continue
		}

		if strings.HasPrefix(line, "CMD EXIT") {
			message := make(map[string]interface{})
			message["id"] = self.GetArgsValue("id")
			message["public_key"] = string(publicKeyString(&self.privateKey.PublicKey))
			message[kJobStatus] = kJobFailed
			log.Println("send result:", message)
			if start {
				self.message <- message
			}
			self.status = kCommandStatusIdle
			cmd.Process.Wait()
			cmd.Process.Kill()
			break
		}
	}
	self.status = kCommandStatusIdle
	self.isFinish = true
}
