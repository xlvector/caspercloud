package caspercloud

import (
	"log"
	"strconv"
	"sync"
	"time"
)

type ServerData struct {
	data map[string]Command
	lock *sync.RWMutex
}

func NewServerData() *ServerData {
	return &ServerData{
		data: make(map[string]Command),
		lock: &sync.RWMutex{},
	}
}

func (self *ServerData) CreateCommand(tmpl, proxyServer string) Command {
	c := NewCasperCmd(tmpl+"_"+strconv.FormatInt(time.Now().UnixNano(), 10),
		tmpl, proxyServer)
	log.Println("success create cmd for", tmpl)
	self.lock.Lock()
	defer self.lock.Unlock()
	self.data[c.GetId()] = c
	return c
}

func (self *ServerData) Delete(id string) {
	self.lock.Lock()
	defer self.lock.Unlock()
	delete(self.data, id)
}

func (self *ServerData) GetCommand(id string) Command {
	self.lock.RLock()
	defer self.lock.RUnlock()

	val, ok := self.data[id]
	if !ok {
		return nil
	}
	return val
}
