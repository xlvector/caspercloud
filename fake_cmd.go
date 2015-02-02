package caspercloud

import (
	"log"
)

type FakeCmd struct {
	cmd      string
	id       string
	message  chan map[string]string
	input    chan map[string]string
	isFinish bool
}

func NewFakeCmd(id string) *FakeCmd {
	return &FakeCmd{
		cmd:      "",
		id:       id,
		message:  make(chan map[string]string, 1),
		input:    make(chan map[string]string, 1),
		isFinish: false,
	}
}

func (fakeCmd *FakeCmd) GetId() string {
	return fakeCmd.id
}

func (fakeCmd *FakeCmd) SetInputArgs(input map[string]string) {
	fakeCmd.input <- input
}

func (fakeCmd *FakeCmd) SetCmd(cmd string) {
	fakeCmd.cmd = cmd
}

func (fakeCmd *FakeCmd) GetMessage() map[string]string {
	return <-fakeCmd.message
}

func (fakeCmd *FakeCmd) Finished() bool {
	return fakeCmd.isFinish
}

func (fakeCmd *FakeCmd) Run() {
	// the first message
	message1 := make(map[string]string)
	message1["id"] = fakeCmd.id
	message1["randcode"] = "http://localhost:8080/abc"
	message1["user_name"] = ""
	message1["passwd"] = ""
	message1[kJobStatus] = kJobOndoing

	fakeCmd.message <- message1

	inputMessage := <-fakeCmd.input
	for k, v := range inputMessage {
		log.Println("get key:", k, " get value:", v)
	}

	message2 := make(map[string]string)
	message2["id"] = fakeCmd.id
	message2["result"] = "get needed value"
	message2[kJobStatus] = kJobFinished
	fakeCmd.isFinish = true
	fakeCmd.message <- message2
}
