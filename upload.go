package caspercloud

import (
	"bytes"
	"encoding/json"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
)

func UploadImage(path string) string {
	params := map[string]string{
		"file": path,
	}
	b, err := upload("http://static.yixin.com/upload",
		params, "file", path)
	if err != nil {
		return ""
	}
	var out map[string]string
	err = json.Unmarshal(b, &out)
	if err != nil {
		return ""
	}
	ret, ok := out["file_url"]
	if !ok {
		return ""
	}
	return ret
}

func upload(link string, params map[string]string,
	paramName, path string) ([]byte, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile(paramName, filepath.Base(path))
	if err != nil {
		return nil, err
	}
	_, err = io.Copy(part, file)
	if err != nil {
		return nil, err
	}

	for key, val := range params {
		writer.WriteField(key, val)
	}
	err = writer.Close()
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", link, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	c := &http.Client{}
	resp, err := c.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return b, nil
}
