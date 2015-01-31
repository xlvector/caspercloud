package caspercloud

import (
	"bufio"
	"log"
	"os"
	"os/exec"
	"strings"
	"time"
)

type CasperCmd struct {
	cmd         string
	proxyServer string
	id          string
	tmpl        string
	message     chan map[string]string
	input       chan map[string]string
	kill        chan int
	isKill      bool
	isFinish    bool
	initialArgs map[string]string
}

func NewCasperCmd(id, tmpl, proxyServer string) *CasperCmd {
	return &CasperCmd{
		cmd:         "",
		proxyServer: proxyServer,
		id:          id,
		tmpl:        tmpl,
		message:     make(chan map[string]string, 1),
		input:       make(chan map[string]string, 1),
		kill:        make(chan int, 1),
		isKill:      false,
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

func (self *CasperCmd) Finished() bool {
	return self.isKill || self.isFinish
}

func (self *CasperCmd) Run() {
	path := "./" + self.tmpl + "/" + self.id
	os.RemoveAll(path)
	if err := os.MkdirAll(path, 0755); err != nil {
		log.Fatalln("can not create", path, err)
	}

	cookieFile, err := os.Create(path + "/cookie.txt")
	defer cookieFile.Close()
	var cmd *exec.Cmd
	if len(self.proxyServer) == 0 {
		cmd = exec.Command("casperjs", self.tmpl+".js", "--cookies-file="+path+"/cookie.txt")
	} else {
		cmd = exec.Command("casperjs", self.tmpl+".js", "--cookies-file="+path+"/cookie.txt", "--proxy="+self.proxyServer, "--proxy-type=http")
	}
	go func() {
		timer := time.NewTimer(time.Minute * KEEP_MINUTES)
		<-timer.C
		self.isKill = true
		cmd.Process.Kill()
	}()
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
	self.isFinish = true
	self.message <- message
}
