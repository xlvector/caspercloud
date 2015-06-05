package caspercloud

import (
	"code.google.com/p/mahonia"
	"crypto/rsa"
	"crypto/tls"
	"fmt"
	"github.com/bitly/go-simplejson"
	"github.com/saintfish/chardet"
	"github.com/xlvector/dlog"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
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

type TaobaoShopCmd struct {
	id            string
	tmpl          string
	userName      string
	userId        string
	passWord      string
	path          string
	message       chan *Output
	input         chan map[string]string
	isKill        bool
	isFinish      bool
	args          map[string]string
	taobaoCookies []*http.Cookie
	alipayCookies []*http.Cookie
	privateKey    *rsa.PrivateKey
	analyzer      *Analyzer
	converter     *Utf8Converter
	client        *http.Client
}

type TaobaoShopCmdFactory struct{}

func (s *TaobaoShopCmdFactory) CreateCommand(params url.Values) Command {
	tmpl := params.Get("tmpl")
	userid := params.Get("userid")
	ret := &TaobaoShopCmd{
		id:        fmt.Sprintf("%s_%d", tmpl, time.Now().UnixNano()),
		tmpl:      tmpl,
		userName:  "",
		userId:    userid,
		passWord:  "",
		message:   make(chan *Output, 5),
		input:     make(chan map[string]string, 5),
		args:      make(map[string]string),
		isKill:    false,
		isFinish:  false,
		analyzer:  NewAnalyzer("server_list.json"),
		converter: NewUtf8Converter(),
		client:    newHttpClient(40),
	}
	var err error
	ret.privateKey, err = GenerateRSAKey()
	if err != nil {
		dlog.Fatalln("fail to generate rsa key", err)
	}
	go ret.run()
	return ret
}

func (s *TaobaoShopCmdFactory) CreateCommandWithPrivateKey(params url.Values, pk *rsa.PrivateKey) Command {
	tmpl := params.Get("tmpl")
	userid := params.Get("userid")
	ret := &TaobaoShopCmd{
		id:         fmt.Sprintf("%s_%d", tmpl, time.Now().UnixNano()),
		tmpl:       tmpl,
		userName:   "",
		userId:     userid,
		passWord:   "",
		message:    make(chan *Output, 5),
		input:      make(chan map[string]string, 5),
		args:       make(map[string]string),
		isKill:     false,
		isFinish:   false,
		analyzer:   NewAnalyzer("server_list.json"),
		converter:  NewUtf8Converter(),
		client:     newHttpClient(20),
		privateKey: pk,
	}
	go ret.run()
	return ret
}

func (self *TaobaoShopCmd) GetId() string {
	return self.id
}

func (self *TaobaoShopCmd) SetInputArgs(input map[string]string) {
	for k, v := range input {
		dlog.Info("set args:%s->%s", k, v)
	}

	if self.Finished() {
		dlog.Warn("start another casperjs")
		go self.run()
	}
	self.input <- input
}

func (self *TaobaoShopCmd) GetMessage() *Output {
	return <-self.message
}

func (self *TaobaoShopCmd) readInputArgs(key string) string {
	dlog.Info("read args:%s", key)

	args := <-self.input
	for k, v := range args {
		if k == "username" {
			self.userName = v
		}

		if k == "password" {
			self.passWord = v
		}

		self.args[k] = v
	}
	if val, ok := self.args[key]; ok {
		return val
	}

	message := &Output{
		Id:        self.GetArgsValue("id"),
		NeedParam: key,
		Status:    NEED_PARAM,
	}
	dlog.Warn("need param:%s", key)
	self.message <- message
	return ""
}

func (self *TaobaoShopCmd) GetArgsValue(key string) string {
	if val, ok := self.args[key]; ok {
		dlog.Info("successfully get args value:%s", val)
		return val
	}
	for {
		val := self.readInputArgs(key)
		if len(val) != 0 {
			dlog.Info("successfully get args value:%s", val)
			return val
		}
	}

}

func (self *TaobaoShopCmd) GetParseReq(fetchStatus string) *ParseRequest {
	ret := &ParseRequest{}
	ret.PrivateKey = string(PrivateKeyString(self.privateKey))
	ret.PublicKey = string(PublicKeyString(&self.privateKey.PublicKey))
	ret.Tmpl = self.tmpl
	ret.FetchStatus = fetchStatus
	ret.UserName = self.userName
	ret.Secret = self.passWord
	if len(self.userId) > 0 {
		ret.RowKey = self.tmpl + "|" + self.userId + "|" + self.userName
	} else {
		ret.RowKey = self.tmpl + "|" + self.userName
	}

	ret.ReqType = ParseRequestType_Html

	// harder code(Todo refact)
	switch {
	case self.tmpl == "taobao_shop":
		ret.ReqType = ParseRequestType_TaobaoShop
	case strings.HasPrefix(self.tmpl, "mail.com"):
		ret.ReqType = ParseRequestType_Eml
	}
	return ret
}

func (self *TaobaoShopCmd) Successed() bool {
	return true
}

func (self *TaobaoShopCmd) Finished() bool {
	return self.isKill || self.isFinish
}

func newHttpClient(timeOutSeconds int) *http.Client {
	client := &http.Client{
		Transport: &http.Transport{
			Dial: func(netw, addr string) (net.Conn, error) {
				timeout := time.Duration(timeOutSeconds) * time.Second
				deadline := time.Now().Add(timeout)
				c, err := net.DialTimeout(netw, addr, timeout)
				if err != nil {
					return nil, err
				}
				c.SetDeadline(deadline)
				return c, nil
			},
			DisableKeepAlives:     true,
			ResponseHeaderTimeout: time.Duration(timeOutSeconds) * time.Second,
			TLSClientConfig:       &tls.Config{InsecureSkipVerify: true},
		},
	}
	return client
}

func (self *TaobaoShopCmd) download(req *http.Request) ([]*http.Cookie, string) {
	for i := 0; i < 5; i++ {
		resp, err := self.client.Do(req)
		if err != nil {
			dlog.Warn("new get request get error:%s", err.Error())
			time.Sleep(100 * time.Millisecond)
			continue
		}
		if resp.StatusCode != 200 {
			dlog.Warn("donload fail:%s", req.URL.String())
			time.Sleep(100 * time.Millisecond)
			continue
		}

		if resp.Body != nil {
			defer resp.Body.Close()
		}
		html, _ := ioutil.ReadAll(resp.Body)
		utf8Body := string(self.converter.ToUTF8(html))
		time.Sleep(200 * time.Millisecond)
		return resp.Cookies(), string(utf8Body)
	}
	return nil, ""
}

func (self *TaobaoShopCmd) downloadWORedirect(req *http.Request) (http.Header, []*http.Cookie) {

	transport := http.Transport{}
	resp, err := transport.RoundTrip(req)
	if err != nil {
		dlog.Warn("round trip error:%s", err.Error())
		return nil, nil
	}
	return resp.Header, resp.Cookies()
}

func (self *TaobaoShopCmd) setCookies(newCookies, oldCookies []*http.Cookie) []*http.Cookie {
	mc := make(map[string]*http.Cookie)
	for _, c := range oldCookies {
		mc[c.Name] = c
	}
	for _, c := range newCookies {
		mc[c.Name] = c
	}
	ret := []*http.Cookie{}
	for _, c := range mc {
		ret = append(ret, c)
	}

	return ret
}

func (self *TaobaoShopCmd) dedupCookie(cookies []*http.Cookie) []*http.Cookie {
	mc := make(map[string]*http.Cookie)
	for _, c := range cookies {
		mc[c.Name] = c
	}
	ret := []*http.Cookie{}
	for _, c := range mc {
		ret = append(ret, c)
	}
	return ret
}

func setHeader(req *http.Request) *http.Request {
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded; charset=UTF-8")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_9_5) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/41.0.2272.104 Safari/537.36")
	return req
}

func (self *TaobaoShopCmd) saveStringToFile(src, fileName string) bool {
	saveFile, err := os.Create(self.path + fileName)
	if err != nil {
		return false
	}
	defer saveFile.Close()
	saveFile.WriteString(src)
	return true
}

func (self *TaobaoShopCmd) getRandCode(cookies []*http.Cookie,
	userName string) string {
	// check varycode
	checkVarycodeParams := url.Values{}
	checkVarycodeParams.Set("username", userName)
	checkVarycodeReq, _ := http.NewRequest("POST", "https://login.taobao.com/member/request_nick_check.do?_input_charset=utf-8", strings.NewReader(checkVarycodeParams.Encode()))
	for _, ck := range cookies {
		checkVarycodeReq.AddCookie(ck)
	}
	checkVarycodeReq = setHeader(checkVarycodeReq)
	_, checkVaryHtml := self.download(checkVarycodeReq)

	json, _ := simplejson.NewJson([]byte(checkVaryHtml))
	if needRandcode, _ := json.Get("needcode").Bool(); needRandcode {
		randcodeLink, _ := json.Get("url").String()
		dlog.Info("get rand code url:%s", randcodeLink)
		randcodeReq, _ := http.NewRequest("GET", randcodeLink, nil)
		setHeader(randcodeReq)
		for _, ck := range cookies {
			randcodeReq.AddCookie(ck)
		}
		_, randcodeBody := self.download(randcodeReq)
		self.saveStringToFile(randcodeBody, "randcode.png")
		imgLink := UploadImage(self.path + "randcode.png")
		dlog.Info("success upload captcha image to:%s", imgLink)

		message := &Output{
			Id:        self.GetArgsValue("id"),
			Status:    OUTPUT_VERIFYCODE,
			Data:      imgLink,
			NeedParam: PARAM_VERIFY_CODE,
		}
		self.message <- message

		randcode := self.GetArgsValue("randcode")
		delete(self.args, "randcode")
		return randcode
	}
	return ""
}

func (self *TaobaoShopCmd) login(userName, passWd string) ([]*http.Cookie, string) {
	var ret []*http.Cookie
	// first step
	startUrl := "https://login.taobao.com/member/login.jhtml"
	startReq, _ := http.NewRequest("GET", startUrl, nil)
	startReq = setHeader(startReq)
	cookies, _ := self.download(startReq)
	randcodeCookies := self.setCookies(cookies, cookies)
	taobaoCookies := self.setCookies(cookies, cookies)
	postHtml := ""
	index := 0
	for ; index < 5; index++ {
		// post data to server
		postParams := url.Values{}
		// check varycode
		randcode := self.getRandCode(randcodeCookies, userName)
		if len(randcode) != 0 {
			postParams.Set("TPL_checkcode", randcode)
		}
		postParams.Set("TPL_username", userName)
		postParams.Set("TPL_password", passWd)
		postParams.Set("newlogin", "0")
		postParams.Set("loginsite", "0")
		postParams.Set("osVer", "macos|10.95")
		postParams.Set("from", "tbTop")
		postParams.Set("umto", "NaN")
		postParams.Set("fc", "default")
		postParams.Set("keyLogin", "false")
		postParams.Set("qrLogin", "true")
		postParams.Set("newMini", "false")
		postParams.Set("support", "000001")
		postParams.Set("CtrlVersion", "1,0,0,7")
		postParams.Set("gvfdcname", "10")
		postParams.Set("loginType", "3")
		postParams.Set("naviVer", "chrome|41.02272101")

		postReq, _ := http.NewRequest("POST", "https://unit.login.taobao.com/member/login.jhtml", strings.NewReader(postParams.Encode()))
		postReq = setHeader(postReq)
		postReq.Header.Set("Referer", "https://login.taobao.com/member/login.jhtml")
		for _, ck := range taobaoCookies {
			postReq.AddCookie(ck)
		}
		cookies, postHtml = self.download(postReq)
		taobaoCookies = self.setCookies(cookies, taobaoCookies)
		//dlog.Info("get post body:%s, username:%s", postHtml, userName)
		if strings.Contains(postHtml, "验证码错误") {
			dlog.Warn("randcode error:%s", userName)
			postHtml = ""
			continue
		}
		if strings.Contains(postHtml, "密码和账户名不匹配") {
			dlog.Warn("username and passwd not match:%s", userName)
			return ret, "账号和密码不匹配"
		}
		break
	}
	if index == 5 {
		return ret, "验证码错误次数大于5次"
	}

	dlog.Info("get post result:%s", postHtml)
	firstSplit := strings.Split(postHtml, "gotoURL:\"")
	if len(firstSplit) < 2 {
		return ret, "二次验证过程失败，请联系管理员"
	}
	secondSplit := strings.Split(firstSplit[1], "\",")
	gotoLink := secondSplit[0]

	gotoReq, _ := http.NewRequest("GET", gotoLink, nil)
	gotoReq = setHeader(gotoReq)
	gotoReq.Header.Set("Refer", "https://login.taobao.com/member/login.jhtml")

	for _, ck := range taobaoCookies {
		gotoReq.AddCookie(ck)
	}

	cookies, gotoBody := self.download(gotoReq)
	taobaoCookies = self.setCookies(cookies, taobaoCookies)

	if strings.Contains(gotoLink, "http://i.taobao.com/my_taobao.htm") {
		return taobaoCookies, ""
	}

	//dlog.Info("get gotoBody:%s, user:%s", gotoBody, self.userName)

	firstSplit = strings.Split(gotoBody, "var durexPop = AQPop({")
	if len(firstSplit) <= 1 {
		return ret, "二次验证过程失败，请联系管理员"
	}
	secondSplit = strings.Split(firstSplit[1], "',")
	phoneLink := strings.TrimPrefix(strings.TrimSpace(strings.Replace(secondSplit[0], "\n", "", -1)), "url:'")
	dlog.Info("get phone link:%s", phoneLink)

	firstSplit = strings.Split(gotoBody, "window.location.href = \"")
	if len(firstSplit) <= 1 {
		return ret, "二次验证过程失败，请联系管理员"
	}
	secondSplit = strings.Split(firstSplit[1], "\";")
	jumpLink := secondSplit[0]
	dlog.Info("get jump link:%s", jumpLink)

	phoneReq, _ := http.NewRequest("GET", phoneLink, nil)
	phoneReq = setHeader(phoneReq)
	phoneReq.Header.Set("Refer", gotoLink)
	for _, ck := range taobaoCookies {
		phoneReq.AddCookie(ck)
	}

	phoneCookies, phoneHtml := self.download(phoneReq)
	taobaoCookies = self.setCookies(phoneCookies, taobaoCookies)

	fSplit := strings.Split(phoneHtml, "id=\"J_DurexData\"")
	sSplit := strings.Split(fSplit[0], "input type=\"hidden\"")
	value := sSplit[len(sSplit)-1]
	value = strings.TrimPrefix(strings.TrimSpace(value), "value='")
	value = strings.TrimSuffix(value, "'")
	dlog.Info("get value:%s", value)

	phoneJosn, _ := simplejson.NewJson([]byte(value))
	if phoneJosn == nil {
		finalFistReq, _ := http.NewRequest("GET", jumpLink, nil)
		finalFistReq = setHeader(finalFistReq)
		for _, ck := range taobaoCookies {
			finalFistReq.AddCookie(ck)
		}

		header, cookies := self.downloadWORedirect(finalFistReq)
		taobaoCookies = self.setCookies(cookies, taobaoCookies)
		finalLink := header.Get("Location")
		dlog.Info("get final second link:%s", finalLink)

		finalSecondReq, _ := http.NewRequest("GET", finalLink, nil)
		finalSecondReq = setHeader(finalSecondReq)
		for _, ck := range taobaoCookies {
			finalSecondReq.AddCookie(ck)
		}
		finalSecondRespCookies, _ := self.download(finalSecondReq)
		taobaoCookies = self.setCookies(finalSecondRespCookies, taobaoCookies)
		return taobaoCookies, ""
	}

	param := phoneJosn.GetPath("param")
	if param == nil {
		dlog.Warn("get nil param:%s", userName)
		return ret, "二次验证过程失败，请联系管理员"
	}
	paramValue, _ := param.String()

	options := phoneJosn.GetPath("options")
	if options == nil {
		dlog.Info("not get options:%s", userName)
		return ret, "二次验证过程失败，请联系管理员"
	}

	option := options.GetIndex(0)
	if option == nil {
		dlog.Warn("get index nil option:%s", userName)
		return ret, "二次验证过程失败，请联系管理员"
	}
	optionText := option.GetPath("optionText")
	if optionText == nil {
		dlog.Warn("get nil optionText:%s", userName)
		return ret, "二次验证过程失败，请联系管理员"
	}
	optionText = optionText.GetIndex(0)
	if optionText == nil {
		dlog.Warn("get nil index of optionText:%s", userName)
		return ret, "二次验证过程失败，请联系管理员"
	}
	phone, _ := optionText.Get("name").String()
	code, _ := optionText.Get("code").String()

	if len(paramValue) == 0 || len(phone) == 0 || len(code) == 0 {
		dlog.Warn("not get enough args:%s", userName)
		return ret, "二次验证过程失败，请联系管理员"
	}

	dlog.Info("user:%s get parame value:%s, phone:%s, code:%s", userName, paramValue, phone, code)
	tmpCookies := taobaoCookies

	for i := 0; i < 5; i++ {
		taobaoCookies = tmpCookies
		phoneParams := url.Values{}
		phoneParams.Set("checkType", "phone")
		phoneParams.Set("target", code)
		phoneParams.Set("safePhoneNum", "")
		phoneParams.Set("checkCode", "")

		phoneCodeLink := fmt.Sprintf("https://aq.taobao.com/durex/sendcode?param=%s&checkType=phone", paramValue)
		dlog.Info("get phone code link:%s", phoneCodeLink)
		sendVarycodeReq, _ := http.NewRequest("POST", phoneCodeLink, strings.NewReader(phoneParams.Encode()))
		sendVarycodeReq = setHeader(sendVarycodeReq)
		sendVarycodeReq.Header.Set("Origin", "https://aq.taobao.com")
		sendVarycodeReq.Header.Set("Referer", phoneLink)

		for _, ck := range taobaoCookies {
			sendVarycodeReq.AddCookie(ck)
		}
		cookies, phoneBody := self.download(sendVarycodeReq)
		taobaoCookies = self.setCookies(cookies, taobaoCookies)

		dlog.Info("get phonebody:%s, username:%s", phoneBody, userName)

		sendJson, _ := simplejson.NewJson([]byte(phoneBody))
		if sendJson == nil {
			dlog.Warn("get nil json:%s", self.userName)
			continue
		}
		succeed, _ := sendJson.Get("isSuccess").Bool()
		if !succeed {
			continue
		}
		message := &Output{
			Id:        self.GetArgsValue("id"),
			NeedParam: "password2",
			Status:    NEED_PARAM,
			Data:      phone,
		}
		self.message <- message

		phoneCode := self.GetArgsValue("password2")
		delete(self.args, "password2")
		phoneParams.Set("checkCode", phoneCode)
		checkPhoneCodeLink := fmt.Sprintf("https://aq.taobao.com/durex/checkcode?param=%s", paramValue)
		varycodeReq, _ := http.NewRequest("POST", checkPhoneCodeLink, strings.NewReader(phoneParams.Encode()))
		varycodeReq = setHeader(varycodeReq)
		varycodeReq.Header.Set("Origin", "https://aq.taobao.com")
		varycodeReq.Header.Set("Referer", phoneLink)

		for _, ck := range taobaoCookies {
			varycodeReq.AddCookie(ck)
		}

		cookies, body := self.download(varycodeReq)
		taobaoCookies = self.setCookies(cookies, taobaoCookies)
		dlog.Info("get phone code check body:%s, username:%s", body, userName)
		if !strings.Contains(body, "isSuccess\":true,") {
			dlog.Warn("get wrong phone check body:%s", body)
			continue
		}

		finalFistReq, _ := http.NewRequest("GET", jumpLink, nil)
		finalFistReq = setHeader(finalFistReq)
		for _, ck := range taobaoCookies {
			finalFistReq.AddCookie(ck)
		}

		header, cookies := self.downloadWORedirect(finalFistReq)
		taobaoCookies = self.setCookies(cookies, taobaoCookies)
		finalLink := header.Get("Location")
		if len(finalLink) != 0 {
			dlog.Info("get final second link:%s", finalLink)
			finalSecondReq, _ := http.NewRequest("GET", finalLink, nil)
			finalSecondReq = setHeader(finalSecondReq)
			for _, ck := range taobaoCookies {
				finalSecondReq.AddCookie(ck)
			}
			finalSecondRespCookies, _ := self.download(finalSecondReq)
			taobaoCookies = self.setCookies(finalSecondRespCookies, taobaoCookies)
		}

		return taobaoCookies, ""

	}
	return ret, "手机验证5次失败，请重新开始登陆"
}

func (self *TaobaoShopCmd) crawl(link, fileName string,
	cookies []*http.Cookie) {
	req, _ := http.NewRequest("GET", link, nil)
	req = setHeader(req)
	if cookies != nil {
		for _, ck := range cookies {
			req.AddCookie(ck)
		}
	}

	_, html := self.download(req)
	self.saveStringToFile(html, fileName)
}

func (self *TaobaoShopCmd) postCrawl(link, fileName string, params url.Values, cookies []*http.Cookie) {
	req, _ := http.NewRequest("POST", link, strings.NewReader(params.Encode()))
	req = setHeader(req)
	for _, ck := range cookies {
		req.AddCookie(ck)
	}
	_, html := self.download(req)
	self.saveStringToFile(html, fileName)
}

func (self *TaobaoShopCmd) downloadAjax(cookies []*http.Cookie) []string {
	var ret []string
	self.crawl("http://beta.sycm.taobao.com/rank/getShopRank.json",
		"SYCM-IDX_shoprank.ajax", cookies)
	ret = append(ret, self.path+"SYCM-IDX_shoprank.ajax")

	self.crawl("http://live.sycm.taobao.com/live/rank/getHotOfferRank.json", "SYCM-IDX_hotofferrank.ajax", cookies)
	ret = append(ret, self.path+"SYCM-IDX_hotofferrank.ajax")
	nowTime := time.Now()
	duration, _ := time.ParseDuration("-24h")
	arg := fmt.Sprintf("%d-%d-%d", nowTime.Add(duration).Year(), nowTime.Add(duration).Month(), nowTime.Add(duration).Day())
	self.crawl("http://bda.sycm.taobao.com/homepage/widget/shopservice/getData.json?dateRange="+arg+"|"+arg+"&dateType=day", "SYCM-IDX_shopservice.ajax", cookies)
	ret = append(ret, self.path+"SYCM-IDX_shopservice.ajax")

	self.crawl("http://bda.sycm.taobao.com/widget/shoptrad/getData.json?dateRange="+arg+"|"+arg+"&dateType=day", "SYCM-IDX_shoprad.ajax", cookies)
	ret = append(ret, self.path+"SYCM-IDX_shoprad.ajax")

	self.crawl("http://bda.sycm.taobao.com/homepage/item/total.json?dateRange="+arg+"|"+arg+"&dateType=day", "SYCM-IDX_item.ajax", cookies)
	ret = append(ret, self.path+"SYCM-IDX_item.ajax")

	self.crawl("http://bda.sycm.taobao.com/summary/getShopSummary.json?dateRange="+arg+"|"+arg+"&dateType=day", "SYCM-IDX_shopsummary.ajax", cookies)
	ret = append(ret, self.path+"SYCM-IDX_shopsummary.ajax")

	self.crawl("http://bda.sycm.taobao.com/tradinganaly/overview/get_summary.json?dateRange="+arg+"|"+arg+"&dateType=day&device=0", "SYCM-TRADE_summary.ajax", cookies)
	ret = append(ret, self.path+"SYCM-TRADE_summary.ajax")

	self.crawl("http://bda.sycm.taobao.com/tradinganaly/overview/getTradeExplanation.json?dateRange="+arg+"|"+arg+"&dateType=day", "SYCM-TRADE_explanation.ajax", cookies)
	ret = append(ret, self.path+"SYCM-TRADE_explanation.ajax")

	self.crawl("http://bda.sycm.taobao.com/tradinganaly/overview/getConvertExplanation.json?dateRange="+arg+"|"+arg+"&dateType=day&device=0", "SYCM-TRADE_convertexplanation.ajax", cookies)
	ret = append(ret, self.path+"SYCM-TRADE_convertexplanation.ajax")

	self.crawl("http://bda.sycm.taobao.com/tradinganaly/overview/getCateTop.json?dateRange="+arg+"|"+arg+"&dateType=day&device=0", "SYCM-TRADE_catetop.ajax", cookies)
	ret = append(ret, self.path+"SYCM-TRADE_catetop.ajax")

	self.crawl("http://bda.sycm.taobao.com/tradinganaly/overview/getTendency.json?dateRange="+arg+"|"+arg+"&dateType=day&device=0", "SYCM-TRADE_tendency.ajax", cookies)
	ret = append(ret, self.path+"SYCM-TRADE_tendency.ajax")

	self.crawl("http://diy.sycm.taobao.com/execute/preview.json?date=3&dateId=1006960&dateType=dynamic&desc=%E5%BA%97%E9%93%BA%E5%88%86%E6%9E%90%E6%9C%88%E6%8A%A5%EF%BC%8C%E6%AF%8F%E6%9C%88%E4%B8%80%E6%AC%A1%E7%9A%84%E6%A0%B8%E5%BF%83%E6%8C%87%E6%A0%87%E7%9A%84%E6%B1%87%E6%80%BB%E6%95%B0%E6%8D%AE%EF%BC%8C%E6%8F%90%E4%BE%9B%E6%9C%80%E8%BF%913%E4%B8%AA%E8%87%AA%E7%84%B6%E6%9C%88%E7%9A%84%E5%BA%97%E9%93%BA%E6%B5%81%E9%87%8F%E4%B8%8E%E9%94%80%E5%94%AE%E7%9B%B8%E5%85%B3%E7%9A%84%E6%95%B0%E6%8D%AE%E6%9F%A5%E8%AF%A2%E3%80%82&filter=[6,9]&id=null&itemId=null&name=%E5%BA%97%E9%93%BA%E5%88%86%E6%9E%90%E6%9C%88%E6%8A%A5&owner=user&show=[{%22id%22:1007205},{%22id%22:1007214},{%22id%22:1016039},{%22id%22:1007223},{%22id%22:1016041},{%22id%22:1007208},{%22id%22:1007210},{%22id%22:1007206},{%22id%22:1016049},{%22id%22:1016056},{%22id%22:1007049},{%22id%22:1007056},{%22id%22:1007064},{%22id%22:1007050},{%22id%22:1011719},{%22id%22:1007057},{%22id%22:1007065},{%22id%22:1007052}]", "SYCM-DTL.ajax", cookies)
	ret = append(ret, self.path+"SYCM-DTL.ajax")

	self.crawl("http://sapp.taobao.com/report/chartLineData.do", "SELL_PRT_chartline.ajax", cookies)
	ret = append(ret, self.path+"SELL_PRT_chartline.ajax")

	self.crawl("http://sapp.taobao.com/report/targetData.do", "SELL_PRT_targetdata.ajax", cookies)
	ret = append(ret, self.path+"SELL_PRT_targetdata.ajax")

	beginDate := fmt.Sprintf("%d-%d-%d", nowTime.Year()-1, nowTime.Month(), nowTime.Day())
	endDate := fmt.Sprintf("%d-%d-%d", nowTime.Year(), nowTime.Month(), nowTime.Day())
	self.crawl("http://notice.taobao.com/json/getPunishHistory.do?begin="+beginDate+"&end="+endDate+"&rangeBegin="+beginDate+"&rangeEnd="+endDate+"&majorType=-1&page=1&pageSize=100", "PUNISH.ajax", cookies)
	ret = append(ret, self.path+"PUNISH.ajax")

	self.crawl("http://notice.taobao.com/json/getPunishHistory.do?begin="+beginDate+"&end="+endDate+"&rangeBegin="+beginDate+"&rangeEnd="+endDate+"&majorType=2&page=1&pageSize=100", "PUNISH_2.ajax", cookies)
	ret = append(ret, self.path+"PUNISH_2.ajax")

	self.crawl("http://notice.taobao.com/json/getPunishHistory.do?begin="+beginDate+"&end="+endDate+"&rangeBegin="+beginDate+"&rangeEnd="+endDate+"&majorType=5&page=1&pageSize=100", "PUNISH_5.ajax", cookies)
	ret = append(ret, self.path+"PUNISH_5.ajax")

	wlEnd := fmt.Sprintf("%d-%d-%d", nowTime.Add(duration).Year(), nowTime.Add(duration).Month(), nowTime.Add(duration).Day())
	wlp2 := fmt.Sprintf("%d-%d-%d", nowTime.Add(duration*7).Year(), nowTime.Add(duration*7).Month(), nowTime.Add(duration*7).Day())
	wlp3 := fmt.Sprintf("%d-%d-%d", nowTime.Add(duration*30).Year(), nowTime.Add(duration*30).Month(), nowTime.Add(duration*30).Day())

	self.crawl("http://bda.sycm.taobao.com/flow/flowmap/flowSource.json?cateId=0&dateRange="+wlEnd+"|"+wlEnd+"&dateType=recent1&device=1&deviceLogicType=1&id=null&index=uv,orderBuyerCnt,orderRate&isActive=false&sourceDataType=0", "SYCM-M-PC_p1.ajax", cookies)
	ret = append(ret, self.path+"SYCM-M-PC_p1.ajax")

	self.crawl("http://bda.sycm.taobao.com/flow/flowmap/flowSource.json?cateId=0&dateRange="+wlp2+"|"+wlEnd+"&dateType=recent7&device=1&deviceLogicType=1&id=null&index=uv,orderBuyerCnt,orderRate&isActive=false&sourceDataType=0", "SYCM-M-PC_p2.ajax", cookies)
	ret = append(ret, self.path+"SYCM-M-PC_p2.ajax")

	self.crawl("http://bda.sycm.taobao.com/flow/flowmap/flowSource.json?cateId=0&dateRange="+wlp3+"|"+wlEnd+"&dateType=recent30&device=1&deviceLogicType=1&id=null&index=uv,orderBuyerCnt,orderRate&isActive=false&sourceDataType=0", "SYCM-M-PC_p3.ajax", cookies)
	ret = append(ret, self.path+"SYCM-M-PC_p3.ajax")

	self.crawl("http://bda.sycm.taobao.com/flow/flowmap/flowSource.json?cateId=0&dateRange="+wlEnd+"|"+wlEnd+"&dateType=recent1&device=4&deviceLogicType=2&id=null&index=uv,orderBuyerCnt,orderRate&isActive=false&sourceDataType=0", "SYCM-M-WL_p1.ajax", cookies)
	ret = append(ret, self.path+"SYCM-M-WL_p1.ajax")

	self.crawl("http://bda.sycm.taobao.com/flow/flowmap/flowSource.json?cateId=0&dateRange="+wlp2+"|"+wlEnd+"&dateType=recent7&device=4&deviceLogicType=2&id=null&index=uv,orderBuyerCnt,orderRate&isActive=false&sourceDataType=0", "SYCM-M-WL_p2.ajax", cookies)
	ret = append(ret, self.path+"SYCM-M-WL_p2.ajax")

	self.crawl("http://bda.sycm.taobao.com/flow/flowmap/flowSource.json?cateId=0&dateRange="+wlp3+"|"+wlEnd+"&dateType=recent30&device=4&deviceLogicType=2&id=null&index=uv,orderBuyerCnt,orderRate&isActive=false&sourceDataType=0", "SYCM-M-WL_p3.ajax", cookies)
	ret = append(ret, self.path+"SYCM-M-WL_p3.ajax")
	return ret
}

func (self *TaobaoShopCmd) downloadNormal(cookies []*http.Cookie) []string {
	var ret []string
	self.crawl("http://member1.taobao.com/member/fresh/deliver_address.htm", "MY_ADDRESS.htm", cookies)
	ret = append(ret, self.path+"MY_ADDRESS.htm")

	self.crawl("http://rate.taobao.com/myRate.htm", "MY_RATE.htm", cookies)
	ret = append(ret, self.path+"MY_RATE.htm")

	self.crawl("http://gold.mai.taobao.com/widget/jinpai.htm", "JINPAI.htm", cookies)
	ret = append(ret, self.path+"JINPAI.htm")

	self.crawl("http://shuju.taobao.com/dataAnalysisShow.htm", "DATA_CTR.htm", cookies)
	ret = append(ret, self.path+"DATA_CTR.htm")

	self.crawl("http://ishop.taobao.com/setup/shop_basic.htm", "SETTING.htm", cookies)
	ret = append(ret, self.path+"SETTING.htm")

	nowTime := time.Now()
	duration, _ := time.ParseDuration("-24h")

	tradeParams := url.Values{}
	tradeParams.Set("billCycleBegin", "201002")
	tradeParams.Set("billCycleBEnd", fmt.Sprintf("%d-%d", nowTime.Add(duration*30).Year(), nowTime.Add(duration*30).Month()))
	tradeParams.Set("countsType", "0")
	tradeParams.Set("checkType", "monthly")
	tradeParams.Set("isSearch", "true")
	tradeParams.Set("action", "payments/ExportFileAction")
	tradeParams.Set("event_submit_do_search", "anything")
	tradeParams.Set("quickSelectTime", "preMonth")
	self.postCrawl("http://pay.taobao.com/payments/tradeDetailFrame.htm", "PAY_TRADE.htm", tradeParams, cookies)
	ret = append(ret, self.path+"PAY_TRADE.htm")

	self.crawl("http://zhaoshang.mall.taobao.com/qualification/qualificationList.htm", "CARD_BRAND.htm", cookies)
	ret = append(ret, self.path+"CARD_BRAND.htm")

	s := fmt.Sprintf("%d-%d-%d", nowTime.Add(duration*15).Year(), nowTime.Add(duration*15).Month(), nowTime.Add(duration*15).Day())
	e := fmt.Sprintf("%d-%d-%d", nowTime.Year(), nowTime.Month(), nowTime.Day())
	ztcS := fmt.Sprintf("%d-%d", nowTime.Add(duration*90).Year(), nowTime.Add(90*duration).Month())
	ztcE := fmt.Sprintf("%d-%d", nowTime.Year(), nowTime.Month())
	self.crawl("http://wuliu.taobao.com/merchant/merchant_dsr.htm?type=4&begin="+s+"&end="+e+"&city=&orderType=100&orderCompare=HANG_YE_AVG", "LOGISTICS.htm", cookies)
	ret = append(ret, self.path+"LOGISTICS.htm")

	self.crawl("http://subway.simba.taobao.com/?tx#!/report/bpreport/index?tcId=-1&chartCompareSelected=&chartCompareDate=&ctab=1&chartField=14&chartCompareField=&start="+ztcS+"&end="+ztcE, "ZTC.htm", cookies)
	ret = append(ret, self.path+"ZTC.htm")

	self.crawl("http://zhaoshang.mall.taobao.com/udform/udfPageFromPC.htm?tpTypeId=102&bizTypeId=13&formName=viewSellerInfo", "CARD_COMPANY.htm", cookies)
	ret = append(ret, self.path+"CARD_COMPANY.htm")

	payExpendParams := url.Values{}
	payExpendParams.Set("accountItemId", "-1")
	payExpendParams.Set("checkType", "monthly")
	payExpendParams.Set("action", "payments/MonthlyUserBillAction")
	payExpendParams.Set("event_submit_do_search", "anything")
	payExpendParams.Set("quickSelectTime", "preMonth")
	payExpendParams.Set("billCycleBegin", "201002")
	payExpendParams.Set("billCycleBEnd", fmt.Sprintf("%d-%d", nowTime.Add(duration*30).Year(), nowTime.Add(duration*30).Month()))
	self.postCrawl("http://pay.taobao.com/payments/expendDetailFrame.htm", "PAY_EXPEND.htm", payExpendParams, cookies)
	ret = append(ret, self.path+"PAY_EXPEND.htm")

	accountParams := url.Values{}
	accountParams.Set("isExport", "false")
	accountParams.Set("checkType", "monthly")
	accountParams.Set("billCycleEnd", fmt.Sprintf("%d-%d", nowTime.Add(duration*30).Year(), nowTime.Add(duration*30).Month()))
	accountParams.Set("acctBookItemId", "-1")
	self.postCrawl("http://pay.taobao.com/accountManage/accountInfoFrame.htm", "PAY_ACCOUNT.htm", accountParams, cookies)
	ret = append(ret, self.path+"PAY_ACCOUNT.htm")

	self.crawl("http://shop.crm.taobao.com/member/memberList.htm", "MEMBER_RL.htm", cookies)
	ret = append(ret, self.path+"MEMBER_RL.htm")

	memberListParams := url.Values{}
	memberListParams.Set("sendType", "1")
	memberListParams.Set("status", "0")
	memberListParams.Set("grade", "-1")
	memberListParams.Set("minTradeAmount", "1000.00")
	memberListParams.Set("minTradeCount", "1")
	memberListParams.Set("province", "0")
	memberListParams.Set("sex", "0")
	memberListParams.Set("mPageSize", "80")
	self.postCrawl("http://shop.crm.taobao.com/member/memberList.htm", "TD_AMT_1000.htm", memberListParams, cookies)
	ret = append(ret, self.path+"TD_AMT_1000.htm")

	memberListParams.Set("minTradeAmount", "0.01")
	memberListParams.Set("minTradeCount", "10")
	self.postCrawl("http://shop.crm.taobao.com/member/memberList.htm", "TD_CNT_10.htm", memberListParams, cookies)
	ret = append(ret, self.path+"TD_CNT_10.htm")

	//TODO(add CARD_STORE.htm && CARD_PEOPLE.htm)
	rateLinkReq, _ := http.NewRequest("GET", "http://sapp.taobao.com/apps/widgets2.htm?id=10010&url=transSellerInfo.vm&callback=jsonp217", nil)
	for _, ck := range cookies {
		rateLinkReq.AddCookie(ck)
	}
	_, rateLinkBody := self.download(rateLinkReq)

	fistSplit := strings.Split(rateLinkBody, "rate.taobao.com")
	if len(fistSplit) > 2 {
		dlog.Info("seg's length:%d seg 0:%s, seg 1:", len(fistSplit), fistSplit[1], fistSplit[2])
		secondSplit := strings.Split(fistSplit[2], " target=")
		rateLink := strings.TrimPrefix(secondSplit[0], "\\/")
		rateLink = strings.TrimSuffix(rateLink, "\\\"")
		rateLink = "http://rate.taobao.com/" + rateLink
		dlog.Info("get rate link:%s", rateLink)
		self.crawl(rateLink, "RATE.htm", nil)
	}
	//post
	return ret
}

func (self *TaobaoShopCmd) getAliloanCookies(cookies []*http.Cookie) []*http.Cookie {
	ret := []*http.Cookie{}
	taobaoCookies := cookies
	firstReq, _ := http.NewRequest("GET", "http://login.taobao.com/member/aliloanAsoGateway.do?target=https://taobao.aliloan.com/tbloan/query/loan_list.htm", nil)
	firstReq = setHeader(firstReq)
	firstReq.Header.Set("Refer", "http://i.taobao.com/my_taobao.htm")

	for _, ck := range cookies {
		firstReq.AddCookie(ck)
	}

	firstRespHeader, firstRespCookies := self.downloadWORedirect(firstReq)
	taobaoCookies = self.setCookies(firstRespCookies, taobaoCookies)
	taobaoCookies = self.dedupCookie(taobaoCookies)
	secondLink := firstRespHeader.Get("Location")
	secondLink = strings.TrimSuffix(secondLink, ")")
	secondLink = strings.TrimPrefix(secondLink, "%!(EXTRA string=")
	dlog.Info("get aliloan first jump link:%s", secondLink)

	if len(secondLink) == 0 {
		return ret
	}
	sencondReq, _ := http.NewRequest("GET", secondLink, nil)
	sencondReq = setHeader(sencondReq)
	secondRespHeader, _ := self.downloadWORedirect(sencondReq)

	thirdLink := secondRespHeader.Get("Location")
	dlog.Info("get aliloan second jump link:%s", thirdLink)
	if len(thirdLink) == 0 {
		return ret
	}

	thirdReq, _ := http.NewRequest("GET", thirdLink, nil)
	thirdReq = setHeader(thirdReq)
	for _, ck := range taobaoCookies {
		thirdReq.AddCookie(ck)
	}
	thirdRespHeader, _ := self.downloadWORedirect(thirdReq)

	fourthLink := thirdRespHeader.Get("Location")
	dlog.Info("get fourth link:%s", fourthLink)
	if len(fourthLink) == 0 {
		return ret
	}

	fourthReq, _ := http.NewRequest("GET", fourthLink, nil)
	fourthReq = setHeader(fourthReq)
	_, aliloanCookies := self.downloadWORedirect(fourthReq)
	return self.dedupCookie(aliloanCookies)
}

func (self *TaobaoShopCmd) getZhifubaoCookies(cookies []*http.Cookie) []*http.Cookie {
	ret := []*http.Cookie{}
	firsReq, _ := http.NewRequest("GET", "https://login.taobao.com/member/login.jhtml?tpl_redirect_url=https%3A%2F%2Fauthzth.alipay.com%3A443%2Flogin%2FtrustLoginResultDispatch.htm%3FredirectType%3D%26sign_from%3D3000%26goto%3Dhttps%253A%252F%252Flab.alipay.com%252Fuser%252Fi.htm%253Fsrc%253Dyy_content_jygl&from_alipay=1", nil)

	firsReq = setHeader(firsReq)
	firsReq.Header.Set("Refer", "http://i.taobao.com/my_taobao.htm")

	for _, ck := range cookies {
		firsReq.AddCookie(ck)
	}

	firstRespHeader, _ := self.downloadWORedirect(firsReq)
	secondLink := firstRespHeader.Get("Location")
	if len(secondLink) == 0 {
		return ret
	}

	sencondReq, _ := http.NewRequest("GET", secondLink, nil)
	sencondReq = setHeader(sencondReq)
	_, alipayCookies := self.downloadWORedirect(sencondReq)
	alipayCookies = self.dedupCookie(alipayCookies)

	postArgs := url.Values{}
	postArgs.Set("goto", "https://lab.alipay.com/user/i.htm?src=yy_content_jygl")
	postArgs.Set("redirectType", "")
	postArgs.Set("tti", "1000")
	postArgs.Set("isIframe", "false")
	postArgs.Set("_seaside_gogo_", "")
	postArgs.Set("_seaside_gogo_p", "")
	postArgs.Set("_seaside_gogo_pcid", "")
	postArgs.Set("is_sign", "N")
	postArgs.Set("real_sn", "")
	postArgs.Set("certCmdOutput", "")
	postArgs.Set("certCmdInput", "")
	postArgs.Set("certfg", "")
	postArgs.Set("security_chrome_extension_aliedit_installed", "false")
	postArgs.Set("security_chrome_extension_alicert_installed", "false")
	postArgs.Set("certCmdInput", "")
	postArgs.Set("security_activeX_enabled", "false")
	postReq, _ := http.NewRequest("POST", "https://authzth.alipay.com:443/login/certCheck.htm", strings.NewReader(postArgs.Encode()))
	postReq = setHeader(postReq)

	for _, ck := range alipayCookies {
		postReq.AddCookie(ck)
	}
	postRespCookies, _ := self.download(postReq)
	alipayCookies = self.setCookies(postRespCookies, alipayCookies)
	alipayCookies = self.dedupCookie(alipayCookies)

	fourthLink := "https://lab.alipay.com:443/user/navigate.htm?goto=https%3A%2F%2Flab.alipay.com%2Fuser%2Fi.htm%3Fsrc%3Dyy_content_jygl"
	fourthReq, _ := http.NewRequest("GET", fourthLink, nil)
	fourthReq = setHeader(fourthReq)

	for _, ck := range alipayCookies {
		fourthReq.AddCookie(ck)
	}

	_, fourthRespCookies := self.downloadWORedirect(fourthReq)
	alipayCookies = self.setCookies(fourthRespCookies, alipayCookies)
	return self.dedupCookie(alipayCookies)
}

func (self *TaobaoShopCmd) downloadAliloan(cookies []*http.Cookie) []string {
	// get zhifubao cookies
	var ret []string
	cookies = self.getAliloanCookies(cookies)
	if len(cookies) == 0 {
		dlog.Warn("not get aliloan cookies:%s", self.userName)
		return ret
	}
	//nowTime := time.Now()
	//duration, _ := time.ParseDuration("-24h")

	loanPostParams := url.Values{}
	loanPostParams.Set("_csrf_token", "7qu0pyRHRKF0luSVv71rc6")
	loanPostParams.Set("_fm.loa._0.pa", "1")
	loanPostParams.Set("_fm.loa._0.p", "10")
	loanPostParams.Set("_fm.loa._0.m", "0")
	loanPostParams.Set("_fm.loa._0.mi", "0")
	loanPostParams.Set("_fm.loa._0.u", "true")
	loanPostParams.Set("_fm.lo._0.pr", "primary")
	loanPostParams.Set("_fm.lo._0.d", "3months")
	loanPostParams.Set("scrollFormTop", "100")
	loanPostParams.Set("_fm.lo._0.s", "")
	self.postCrawl("https://taobao.aliloan.com/tbloan/query/loan_list.htm", "LOAN_INFO.htm", loanPostParams, cookies)
	ret = append(ret, self.path+"LOAN_INFO.htm")
	self.crawl("https://taobao.aliloan.com/tbloan/query/loan_list.htm", "LOANS.htm", cookies)
	ret = append(ret, self.path+"LOANS.htm")

	self.crawl("https://taobao.aliloan.com/tbloan/user/score/item_search.htm?hidden=false", "SCORE.htm", cookies)
	ret = append(ret, self.path+"SCORE.htm")
	self.crawl("https://taobao.aliloan.com/tbloan/index.htm?urlAutoSelect=xy6", "MY_CREDIT.htm", cookies)
	ret = append(ret, self.path+"MY_CREDIT.htm")

	self.crawl("https://taobao.aliloan.com/sellercenter/index.htm", "RCV_AHEAD.htm", cookies)
	ret = append(ret, self.path+"RCV_AHEAD.htm")
	return ret
}

func (self *TaobaoShopCmd) downloadZhifubao(cookies []*http.Cookie) []string {
	// get zhifubao cookies
	var ret []string
	cookies = self.getZhifubaoCookies(cookies)
	if len(cookies) == 0 {
		return ret
	}
	ctoken := ""
	for _, ck := range cookies {
		if ck.Name == "ctoken" {
			ctoken = ck.Value
		}
	}

	//nowTime := time.Now()
	//duration, _ := time.ParseDuration("-24h")

	self.crawl("https://personalportal.alipay.com/portal/account/index.htm", "MY_ALIPAY.htm", cookies)
	ret = append(ret, self.path+"MY_ALIPAY.htm")

	self.crawl("https://memberprod.alipay.com/user/contacts/index.htm", "CONTACTS.htm", cookies)
	ret = append(ret, self.path+"CONTACTS.htm")

	self.crawl("https://zd.alipay.com/ebill/copenhagen.json?ctoken="+ctoken, "MONTH_BILL_copenhagen.ajax", cookies)
	ret = append(ret, self.path+"MONTH_BILL_copenhagen.ajax")

	self.crawl("https://zd.alipay.com/ebill/frontPlaze.json?ctoken="+ctoken, "MONTH_BILL_frontPlaze.ajax", cookies)
	ret = append(ret, self.path+"MONTH_BILL_frontPlaze.ajax")

	self.crawl("https://zd.alipay.com/ebill/bankInfo.json?ctoken="+ctoken, "MONTH_BILL_bankInfo.ajax", cookies)
	ret = append(ret, self.path+"MONTH_BILL_bankInfo.ajax")

	self.crawl("https://zd.alipay.com/ebill/tradeInfo.json?ctoken="+ctoken, "MONTH_BILL_tradeInfo.ajax", cookies)
	ret = append(ret, self.path+"MONTH_BILL_tradeInfo.ajax")

	self.crawl("https://zd.alipay.com/ebill/consumeTrend.json?ctoken="+ctoken, "MONTH_BILL_consumeTrend.ajax", cookies)
	ret = append(ret, self.path+"MONTH_BILL_consumeTrend.ajax")

	self.crawl("https://zd.alipay.com/ebill/userInfo.json?ctoken="+ctoken, "MONTH_BILL_userInfo.ajax", cookies)
	ret = append(ret, self.path+"MONTH_BILL_userInfo.ajax")

	self.crawl("https://zd.alipay.com/ebill/ebillCycleList.json?ctoken="+ctoken, "MONTH_BILL_ebillCycleList.ajax", cookies)
	ret = append(ret, self.path+"MONTH_BILL_ebillCycleList.ajax")

	return ret
}

func (self *TaobaoShopCmd) checkLoginSuccess(cookies []*http.Cookie) bool {
	checkReq, _ := http.NewRequest("GET", "http://beta.sycm.taobao.com/rank/getShopRank.json", nil)
	checkReq = setHeader(checkReq)
	for _, ck := range cookies {
		checkReq.AddCookie(ck)
	}
	_, checkBody := self.download(checkReq)
	dlog.Info("get check login success body:%s", checkBody)
	if strings.Contains(checkBody, "login system") {
		return false
	}
	return true
}

func (self *TaobaoShopCmd) run() {
	dlog.Info("begin run cmd:%s", self.tmpl)
	self.isFinish = false
	self.isKill = false

	self.path = "./" + self.tmpl + "/" + self.id + "/"
	os.RemoveAll(self.path)
	if err := os.MkdirAll(self.path, 0755); err != nil {
		dlog.Fatalln("can not create", self.path, err)
	}

	go func() {
		timer := time.NewTimer(20 * time.Minute)
		<-timer.C
		self.isKill = true
	}()

	// output public key
	message := &Output{
		Id:     self.GetArgsValue("id"),
		Status: OUTPUT_PUBLICKEY,
		Data:   string(PublicKeyString(&self.privateKey.PublicKey)),
	}
	self.message <- message

	// get username and passwd
	userName := self.GetArgsValue("username")
	delete(self.args, "username")
	if self.analyzer != nil {
		req := self.GetParseReq(kFetchStarted)
		go self.analyzer.sendReq(req)
		dlog.Info("report status started:%s", req.RowKey)
	}

	passWd := self.GetArgsValue("password")
	passWd = DecodePassword(passWd, self.privateKey)
	delete(self.args, "password")
	cookies, msg := self.login(userName, passWd)
	dlog.Info("get msg:%s", msg)
	success := self.checkLoginSuccess(cookies)
	if !success {
		if self.analyzer != nil {
			req := self.GetParseReq(kFetchFailed)
			dlog.Info("fetch failed:%s", req.RowKey)
			go self.analyzer.sendReq(req)
		}
		message = &Output{
			Status: FAIL,
			Id:     self.GetArgsValue("id"),
			Data:   msg,
		}
		self.message <- message
		dlog.Info("get msg:%s", msg)
		return
	}
	// check login

	// login success
	message = &Output{
		Id:     self.GetArgsValue("id"),
		Status: LOGIN_SUCCESS,
	}
	self.message <- message

	var ret []string
	ajax := self.downloadAjax(cookies)
	normal := self.downloadNormal(cookies)
	zhifubao := self.downloadZhifubao(cookies)
	aliloan := self.downloadAliloan(cookies)
	ret = append(ret, ajax...)
	ret = append(ret, normal...)
	ret = append(ret, zhifubao...)
	ret = append(ret, aliloan...)

	dlog.Info("finish all:%s, fetch doc num:%d", self.userName, len(ret))
	if self.analyzer != nil {
		req := self.GetParseReq(kFetchFinished)
		dlog.Info("fetch finished:%s", req.RowKey)
		go self.analyzer.Process(req, ret)
	}

	message = &Output{
		Status: FINISH_FETCH_DATA,
		Id:     self.GetArgsValue("id"),
	}
	self.message <- message

	self.isFinish = true
}
