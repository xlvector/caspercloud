package caspercloud

import (
	"github.com/jhillyerd/go.enmime"
	"net/mail"
	"os"
	"testing"
)

func TestEml(t *testing.T) {
	f, _ := os.Open("test.eml")
	msg, err := mail.ReadMessage(f)
	if err != nil {
		t.Error(err)
	}
	t.Log(msg.Header)
	body, err := enmime.ParseMIMEBody(msg)
	if err != nil {
		t.Error(err)
	}
	t.Log(body.Html)
}
