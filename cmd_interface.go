package caspercloud

type Command interface {
	GetMessage() map[string]interface{}
	GetStatus() int
	SetInputArgs(map[string]string)
	Finished() bool
	GetId() string
}
