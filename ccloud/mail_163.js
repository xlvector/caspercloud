var totalTime = (new Date()).getTime();

var casper = require('casper').create();
var system = require('system');
casper.on('resource.requested', function(resource) {
    //this.echo(resource.url);
});

casper.options.onResourceRequested = function(C, requestData, request) {
    var url = requestData['url'];
    if ((/http:.+?.(gif|png|jpg|woff)/gi).test(url)
        || url.indexOf('data:image') == 0
        || url.indexOf('/web.yixin.im/') >= 0
        || url.indexOf('/pimg1.126.net/') >= 0
        || url.indexOf('/weather.mail.163.com/') >= 0
        || url.indexOf('/caipiao.163.com/') >= 0) {
        request.abort();
    }
};

casper.options.onResourceReceived = function(C, response) {
    //console.log('download ' + JSON.stringify(response));
};

casper.userAgent("Mozilla/5.0 (Macintosh; Intel Mac OS X 10_10_0) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/39.0.2171.71 Safari/537.36");

var username = "";
var password = "";

casper.start("http://mail.163.com/", function(){
    this.fill("#login163", {"username": "", "password": ""}, false);
    system.stdout.writeLine("CMD Info List: username and password");
    system.stdout.writeLine("CMD GET ARGS /username/password");
    username = system.stdin.readLine();
    password = system.stdin.readLine();
    this.sendKeys("#idInputLine input", username);
    this.sendKeys("#pwdInputLine input", password);
    this.capture("./mail_163/" + username + "/login.png");
    this.click("#loginBtn");
})

casper.waitUntilVisible("#_mail_component_51_51", function(){
    this.capture("./mail_163/" + username + "/email1.png");
    this.click("#_mail_component_51_51")
}, function(){
    this.capture("./mail_163/" + username + "/email1_timeout.png");
}, 30000);

casper.waitUntilVisible("div[id$=_ListDiv]", function(){
    this.capture("./mail_163/" + username + "/email2.png");
    var mails = this.evaluate(function(){
        var list = document.querySelectorAll(".nl0");
        var ret = [];
        for(var i = 0; i < list.length; i++){
            ret.push({
                title: list[i].querySelector(".dP0").innerText, 
                from: list[i].querySelector(".il0").innerText
            });
        }
        return JSON.stringify(ret);
    });
    console.log(mails);
    console.log("used time: " + ((new Date()).getTime() - totalTime) + "ms");
}, function(){
    this.capture("./mail_163/" + username + "/email2_timeout.png");
}, 30000);


casper.run();




