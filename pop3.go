package caspercloud

import (
	"errors"
	"github.com/taknb2nch/go-pop3"
	"log"
	"strings"
)

func getAddressByUsername(username string) string {
	tks := strings.Split(username, "@")
	if len(tks) != 2 {
		return ""
	}
	switch tks[1] {
	case "aliyun.com":
		return "pop3.mail.aliyun.com:110"
	default:
		return ""
	}
	return ""
}

func Pop3ReceiveMail(username, password string) ([]string, error) {
	ret := []string{}
	addr := getAddressByUsername(username)
	if len(addr) == 0 {
		return ret, errors.New("fail to get pop3 addr by username")
	}
	if err := pop3.ReceiveMail(addr, username, password,
		func(number int, uid, data string, err error) (bool, error) {
			log.Println(number, uid)
			ret = append(ret, data)
			return true, nil
		}); err != nil {
		return ret, err
	}
	return ret, nil
}
