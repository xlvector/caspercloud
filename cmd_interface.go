package caspercloud

type Command interface {
	GetMessage() map[string]string
	SetCmd(cmd string)
	SetInputArgs(map[string]string)
	Run()
}
