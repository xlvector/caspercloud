package caspercloud

import (
	"code.google.com/p/mahonia"
	"github.com/saintfish/chardet"
	"strings"
)

type Utf8Converter struct {
	detector *chardet.Detector
	gbk      mahonia.Decoder
	big5     mahonia.Decoder
}

func NewUtf8Converter() *Utf8Converter {
	ret := Utf8Converter{}
	ret.detector = chardet.NewHtmlDetector()
	ret.gbk = mahonia.NewDecoder("gb18030")
	ret.big5 = mahonia.NewDecoder("big5")
	return &ret
}

func (self *Utf8Converter) DetectCharset(html []byte) string {
	rets, err := self.detector.DetectAll(html)
	if err != nil {
		return ""
	}
	maxret := ""
	w := 0
	for _, ret := range rets {
		cs := strings.ToLower(ret.Charset)
		if strings.HasPrefix(cs, "gb") || strings.HasPrefix(cs, "utf") {
			if w < ret.Confidence {
				w = ret.Confidence
				maxret = cs
			}
		} else {
			continue
		}
	}
	return maxret
}

func (self *Utf8Converter) ToUTF8(html []byte) []byte {
	charset := self.DetectCharset(html)

	if !strings.Contains(charset, "gb") && !strings.Contains(charset, "big") {
		charset = "utf-8"
	}
	if charset == "utf-8" || charset == "utf8" {
		return html
	} else if charset == "gb2312" || charset == "gb-2312" || charset == "gbk" || charset == "gb18030" || charset == "gb-18030" {
		ret, ok := self.gbk.ConvertStringOK(string(html))
		if ok {
			return []byte(ret)
		}
	} else if charset == "big5" {
		ret, ok := self.big5.ConvertStringOK(string(html))
		if ok {
			return []byte(ret)
		}
	}
	return nil
}
