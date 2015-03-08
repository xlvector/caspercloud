package caspercloud

import (
	"github.com/taknb2nch/go-pop3"
	"log"
	"testing"
)

func TestPop3(t *testing.T) {
	if err := pop3.ReceiveMail("pop3.mail.aliyun.com:110",
		"caspercloud@aliyun.com", "Change2Day",
		func(number int, uid, data string, err error) (bool, error) {
			log.Printf("%d, %s\n", number, uid)
			log.Println(data)
			// implement your own logic here

			return false, nil
		}); err != nil {
		log.Fatalf("%v\n", err)
	}
}
