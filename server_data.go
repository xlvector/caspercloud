package caspercloud

import (
	"log"
	"math/rand"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	kMaxServerNum = 1
)

type ServerData struct {
	data   map[string][]Command
	lock   *sync.RWMutex
	random *rand.Rand
}

func NewServerData() *ServerData {
	return &ServerData{
		data:   make(map[string][]Command),
		lock:   &sync.RWMutex{},
		random: rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

func (self *ServerData) getNextIndex(index int) int {
	index += 1
	if index >= kMaxServerNum {
		return 0
	}
	return index
}

func (self *ServerData) GetNewCommand(tmpl, proxyServer string) (id string, c Command) {
	self.lock.RLock()
	val, ok := self.data[tmpl]
	self.lock.RUnlock()
	if !ok {
		self.lock.Lock()
		var cmds []Command
		for i := 0; i < kMaxServerNum; i++ {
			c := NewCasperCmd(strconv.FormatInt(int64(i), 10), tmpl, proxyServer)
			cmds = append(cmds, c)
		}
		self.data[tmpl] = cmds
		log.Println("add cmd for template:", tmpl)
		self.lock.Unlock()
		val, _ = self.data[tmpl]
	}

	index := self.random.Intn(kMaxServerNum)
	initialIndex := index
	for {
		if val[index].GetStatus() == kCommandStatusIdle {
			id = tmpl + "#" + strconv.FormatInt(int64(index), 10)
			c = val[index]
			return id, c
		}

		index = self.getNextIndex(index)
		if index == initialIndex {
			return "", nil
		}
	}
	return "", nil

}

func (self *ServerData) parseId(id string) (tmpl string, index int) {
	segs := strings.Split(id, "#")
	if len(segs) < 1 {
		return "", -1
	}
	tmpl = segs[0]
	index, _ = strconv.Atoi(segs[1])
	return tmpl, index
}

func (self *ServerData) GetCommand(id string) Command {
	self.lock.RLock()
	defer self.lock.RUnlock()

	tmpl, index := self.parseId(id)
	if index < 0 || index >= kMaxServerNum {
		return nil
	}

	if val, ok := self.data[tmpl]; ok {
		return val[index]
	}
	return nil
}
