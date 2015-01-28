
var casper = require('casper').create();
var system = require('system');


casper.options.onResourceRequested = function(C, requestData, request) {
    var url = requestData['url'];
    console.log('url:' + url);
    if (url.indexOf('vcode.aspx?f=12') >= 0) {
        var newUrl = url +'&randcode=true';
        request.changeUrl(newUrl);
    } else {
       // var newUrl = url +'&randcode=false';
       // request.changeUrl(newUrl);
    }
};

casper.options.onResourceReceived = function(C, response) {   
    if (response.url.indexOf('vcode.aspx?f=12') >= 0) {
        console.log('CMD Info List:' + JSON.stringify(response.headers));
    }
};

casper.userAgent("Mozilla/5.0 (Macintosh; Intel Mac OS X 10_10_0) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/39.0.2171.71 Safari/537.36");

var name = "";
var randcode = "";

casper.start("http://www.haiguan.info/onlinesearch/gateway/CompanyInfo.aspx", function(){
    if (this.exists('#ctl00_MainContent_lbCheckCode')) {
        this.fill("#aspnetForm", {"ctl00$MainContent$txtCode": "","ctl00$MainContent$code_op":""}, false);
        this.capture()
        system.stdout.writeLine("CMD GET ARGS /name/randcode");
        name = system.stdin.readLine();
        randcode = system.stdin.readLine();
        this.sendKeys("#ctl00_MainContent_txtCode", name);
        this.sendKeys("#ctl00_MainContent_code_op", randcode);
        console.log("click sigin");
        this.click("#ctl00_MainContent_ImgBtn"); 
    }
});

casper.waitUntilVisible("table[id$=ctl00_MainContent_gvCmpany]", function(){
    console.log("find table id");
    var content = this.evaluate(function(){
        var list = document.querySelectorAll("#ctl00_MainContent_gvCmpany tbody tr td");
        var ret = [];
        for(var i = 0; i < list.length; i++){
            ret.push({
                text: list[i].innerText
            });
        }
        //console.log(JSON.stringify(ret));
        return JSON.stringify(ret);
    });
    console.log("get content:" + content);
    //console.log("used time: " + ((new Date()).getTime() - totalTime) + "ms");
}, function(){
    //this.capture("./12306/" + username + "/email2_timeout.png");
}, 10000);


casper.run();
