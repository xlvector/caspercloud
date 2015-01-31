package caspercloud

type Command interface {
	GetMessage() map[string]string
	GetStatus() int
	SetInputArgs(map[string]string)
	Finished() bool
}
