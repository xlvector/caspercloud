package caspercloud

import (
	"log"
	"testing"
)

func TestUpload(t *testing.T) {
	params := map[string]string{
		"file": "upload.go",
	}
	b, err := upload("https://static.yixin.com/upload",
		params, "file", "upload.go")
	if err != nil {
		t.Error(err)
	}
	log.Println(string(b))
}
