package caspercloud

import (
	"bufio"
	"log"
	"os"
	"os/exec"
	"strings"
)

const (
	kProxyServer = "127.0.0.1:7182"
)

type CasperCmd struct {
	cmd         string
	id          string
	timpl       string
	message     chan map[string]string
	input       chan map[string]string
	initialArgs map[string]string
}

func NewCasperCmd(id, timpl string) *CasperCmd {
	return &CasperCmd{
		cmd:         "",
		id:          id,
		timpl:       timpl,
		message:     make(chan map[string]string, 1),
		input:       make(chan map[string]string, 1),
		initialArgs: make(map[string]string),
	}
}

func (self *CasperCmd) SetInputArgs(input map[string]string) {
	self.input <- input
}

func (self *CasperCmd) SetCmd(cmd string) {
	self.cmd = cmd
}

func (self *CasperCmd) SetInitialArgs(args map[string]string) {
	for k, v := range args {
		self.initialArgs[k] = v
		log.Println("set k:", k, " v:", v)
	}
}

func (self *CasperCmd) GetMessage() map[string]string {
	return <-self.message
}

func (self *CasperCmd) readInputArgs(key string) string {
	args := <-self.input
	for k, v := range args {
		self.initialArgs[k] = v
	}
	if val, ok := self.initialArgs[key]; ok {
		return val
	}

	message := make(map[string]string)
	message["id"] = self.id
	message["need_args"] = key
	message[kJobStatus] = kJobOndoing
	self.message <- message

	return ""
}

func (self *CasperCmd) GetArgsValue(key string) string {
	key = strings.TrimRight(key, "\n")
	log.Println("start to get args:", key)
	if val, ok := self.initialArgs[key]; ok {
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
	return ""
}

func (self *CasperCmd) getArgsList(args string) []string {
	segs := strings.Split(args, "/")
	if len(segs) < 2 {
		return nil
	}
	return segs[1:]
}

func (self *CasperCmd) Run() {
	path := "./" + self.timpl + "/" + self.id
	os.RemoveAll(path)
	if err := os.MkdirAll(path, 0755); err != nil {
		log.Fatalln("can not create", path, err)
	}

	cookieFile, err := os.Create(path + "/cookie.txt")
	defer cookieFile.Close()

	//cmd := exec.Command("casperjs", self.timpl+".js", "--cookies-file="+path+"/cookie.txt", "--proxy="+kProxyServer, "--proxy-type=http")
	cmd := exec.Command("casperjs", self.timpl+".js", "--cookies-file="+path+"/cookie.txt")
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Fatalln("can not get stdout pipe:", err)
	}
	bufout := bufio.NewReader(stdout)

	stdin, err := cmd.StdinPipe()
	if err != nil {
		log.Fatalln("can not get stdin pipe:", err)
	}

	bufin := bufio.NewWriter(stdin)
	if err := cmd.Start(); err != nil {
		log.Fatalln("can not start cmd:", err)
	}

	result := ""
	for {
		line, err := bufout.ReadString('\n')
		if err != nil {
			break
		}

		if strings.Contains(line, "CMD Info List") {
			message := make(map[string]string)
			message["id"] = self.id
			message["info"] = line
			message[kJobStatus] = kJobOndoing
			self.message <- message
		}

		if strings.Contains(line, "CMD GET ARGS") {
			log.Println("strings.Contains(line)")
			for _, v := range self.getArgsList(line) {
				bufin.WriteString(self.GetArgsValue(v))
				bufin.WriteRune('\n')
				bufin.Flush()
			}
			continue
		}

		result += line + "<br/>"
	}
	message := make(map[string]string)
	message["id"] = self.id
	message["result"] = result
	message[kJobStatus] = kJobFinished
	self.message <- message
}
