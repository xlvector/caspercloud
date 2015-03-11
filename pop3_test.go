package caspercloud

import (
	"github.com/taknb2nch/go-pop3"
	"log"
	"testing"
)

func TestPop3(t *testing.T) {
	if err := pop3.ReceiveMail("pop.qq.com:995",
		"xlvector@qq.com", "Pi31415926",
		func(number int, uid, data string, err error) (bool, error) {
			log.Printf("%d, %s\n", number, uid)
			log.Println(data)
			return false, pop3.EOF
		}); err != nil {
		log.Fatalf("%v\n", err)
	}
}
