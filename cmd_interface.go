package caspercloud

const (
	kNotGetResutl = "not get needed result"
)

type Cmd interface {
	GetMessage() map[string]string
	SetCmd(cmd string)
	SetInputArgs(map[string]string)
	Run()
}
