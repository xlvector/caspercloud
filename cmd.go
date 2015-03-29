package caspercloud

type Command interface {
	GetMessage() *Output
	SetInputArgs(map[string]string)
	Finished() bool
	GetId() string
}

const (
	PARAM_USERNAME    = "username"
	PARAM_PASSWORD    = "password"
	PARAM_PASSWORD2   = "password2"
	PARAM_VERIFY_CODE = "randcode"

	FAIL                  = "fail"
	NEED_PARAM            = "need_param"
	WRONG_PASSWORD        = "wrong_password"
	WRONG_VERIFYCODE      = "wrong_verifycode"
	WRONG_SECOND_PASSWORD = "wrong_second_password"
	LOGIN_SUCCESS         = "login_success"
	BEGIN_FETCH_DATA      = "begin_fetch_data"
	FINISH_FETCH_DATA     = "finish_fetch_data"
	FINISH_ALL            = "finish_all"
	OUTPUT_PUBLICKEY      = "output_publickey"
	OUTPUT_VERIFYCODE     = "output_verifycode"
)

type Output struct {
	Status    string `json:"status"`
	NeedParam string `json:"need_param"`
	Id        string `json:"id"`
	Data      string `json:"data"`
}
