package caspercloud

import (
	"log"
	"math/rand"
	"strconv"
	"strings"
	"sync"
	"time"
)

type ServerData struct {
	data   map[string][]Command
	index  map[string]Command
	lock   *sync.RWMutex
	random *rand.Rand
}

func NewServerData() *ServerData {
	return &ServerData{
		data:   make(map[string][]Command),
		index:  make(map[string]Command),
		lock:   &sync.RWMutex{},
		random: rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

func (self *ServerData) searchIdleCommand(cmds []Command) Command {
	for i := 0; i < len(cmds) && i < 3; i++ {
		k := self.random.Intn(len(cmds))
		if cmds[k].GetStatus() == kCommandStatusIdle {
			return cmds[k]
		}
	}
	return nil
}

func (self *ServerData) GetNewCommand(tmpl, proxyServer string) Command {
	self.lock.RLock()
	val, ok := self.data[tmpl]
	self.lock.RUnlock()
	if ok {
		c := self.searchIdleCommand(val)
		if c != nil {
			return c
		}
	}
	if !ok {
		self.lock.Lock()
		var cmds []Command
		c := NewCasperCmd(tmpl+"#"+strconv.FormatInt(time.Now().UnixNano(), 10), tmpl, proxyServer)
		cmds = append(cmds, c)
		self.data[tmpl] = cmds
		self.index[c.GetId()] = c
		log.Println("add cmd for template:", tmpl)
		self.lock.Unlock()
		return c
	}

	c := NewCasperCmd(tmpl+"#"+strconv.FormatInt(time.Now().UnixNano(), 10), tmpl, proxyServer)
	val = append(val, c)
	return c
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

	val, ok := self.index[id]
	if !ok {
		return nil
	}
	return val
}
