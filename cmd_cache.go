package caspercloud

import (
	"sync"
)

type CommandCache struct {
	data map[string]Command
	lock *sync.RWMutex
}

func NewCommandCache() *CommandCache {
	return &CommandCache{
		data: make(map[string]Command),
		lock: &sync.RWMutex{},
	}
}

func (self *CommandCache) SetCommand(c Command) {
	self.lock.Lock()
	defer self.lock.Unlock()
	self.data[c.GetId()] = c
}

func (self *CommandCache) Delete(id string) {
	self.lock.Lock()
	defer self.lock.Unlock()
	delete(self.data, id)
}

func (self *CommandCache) GetCommand(id string) Command {
	self.lock.RLock()
	defer self.lock.RUnlock()

	val, ok := self.data[id]
	if !ok {
		return nil
	}
	return val
}
