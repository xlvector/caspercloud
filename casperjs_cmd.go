package caspercloud

import (
	"bufio"
	"log"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"
)

type CasperCmd struct {
	proxyServer string
	id          string
	tmpl        string
	message     chan map[string]string
	input       chan map[string]string
	isKill      bool
	isFinish    bool
	args        map[string]string
	status      int
	lock        *sync.RWMutex
}

func NewCasperCmd(id, tmpl, proxyServer string) *CasperCmd {
	ret := &CasperCmd{
		proxyServer: proxyServer,
		id:          id,
		tmpl:        tmpl,
		message:     make(chan map[string]string, 1),
		input:       make(chan map[string]string, 1),
		args:        make(map[string]string),
		status:      kCommandStatusIdle,
		lock:        &sync.RWMutex{},
	}
	go ret.run()

	return ret
}

func (self *CasperCmd) GetStatus() int {
	self.lock.RLock()
	defer self.lock.RUnlock()
	return self.status
}

func (self *CasperCmd) SetInputArgs(input map[string]string) {
	if self.Finished() {
		go self.run()
	}

	self.input <- input
}

func (self *CasperCmd) GetMessage() map[string]string {
	return <-self.message
}

func (self *CasperCmd) readInputArgs(key string) string {
	args := <-self.input
	for k, v := range args {
		self.args[k] = v
	}
	if val, ok := self.args[key]; ok {
		return val
	}

	message := make(map[string]string)
	message["id"] = self.GetArgsValue("id")
	message["need_args"] = key
	message[kJobStatus] = kJobOndoing
	self.message <- message

	return ""
}

func (self *CasperCmd) GetArgsValue(key string) string {
	log.Println("start to get args:", key)
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
	self.lock.Lock()
	defer self.lock.Unlock()
	return self.isKill || self.isFinish
}

func (self *CasperCmd) run() {
	self.lock.Lock()
	self.isFinish = false
	self.isKill = false
	self.lock.Unlock()

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
		timer := time.NewTimer(time.Minute * kKeepMinutes)
		<-timer.C
		self.lock.Lock()
		self.isKill = true
		self.lock.Unlock()
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

	for {
		line, err := bufout.ReadString('\n')
		if err != nil {
			break
		}

		if strings.HasPrefix(line, "CMD INFO WAITING FOR SERVICE") {
			if self.GetArgsValue("start") == "yes" {
				delete(self.args, "start")
				self.lock.Lock()
				self.status = kCommandStatusBusy
				self.lock.Unlock()
				bufin.WriteString("start")
				bufin.WriteRune('\n')
				bufin.Flush()
			}
		}

		if strings.HasPrefix(line, "CMD Info List") {
			message := make(map[string]string)
			message["id"] = self.GetArgsValue("id")
			message["info"] = strings.TrimPrefix(line, "CMD Info List")
			message[kJobStatus] = kJobOndoing
			self.message <- message
		}

		if strings.HasPrefix(line, "CMD GET ARGS") {
			for _, v := range self.getArgsList(line) {
				key := strings.TrimRight(v, "\n")
				bufin.WriteString(self.GetArgsValue(key))
				delete(self.args, key)
				bufin.WriteRune('\n')
				bufin.Flush()
			}
			continue
		}

		if strings.HasPrefix(line, "CMD INFO CONTENT") {
			message := make(map[string]string)
			message["id"] = self.GetArgsValue("id")
			message["result"] = strings.TrimPrefix(line, "CMD INFO CONTENT")
			message[kJobStatus] = kJobFinished
			self.message <- message
			self.lock.Lock()
			self.status = kCommandStatusIdle
			self.lock.Unlock()
		}
	}
	/*
		message := make(map[string]string)
		message["id"] = self.id
		//message["result"] = result
		message[kJobStatus] = kJobFinished
	*/
	self.lock.Lock()
	self.isFinish = true
	self.lock.Unlock()
	//self.message <- message
}
