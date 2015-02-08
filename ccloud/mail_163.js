var casper = require('casper').create();
var system = require('system');
casper.on('resource.requested', function(resource) {
});

casper.options.onResourceRequested = function(C, requestData, request) {
    var url = requestData['url'];
    /*
    //console.log(url);
    if ((/http:.+?.(gif|png|jpg|woff)/gi).test(url)
        || url.indexOf('data:image') == 0
        || url.indexOf('/web.yixin.im/') >= 0
        || url.indexOf('/pimg1.126.net/') >= 0
        || url.indexOf('/weather.mail.163.com/') >= 0
        || url.indexOf('/caipiao.163.com/') >= 0) {
        request.abort();
    }
    */
    
};

casper.options.onResourceReceived = function(C, response) {
    //console.log('download ' + JSON.stringify(response))
};

casper.userAgent("Mozilla/5.0 (Macintosh; Intel Mac OS X 10_10_0) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/39.0.2171.71 Safari/537.36");

var interact = "";
var username = "";
var password = "";
var randcode = "";
var exported = true;

casper.start("http://mail.163.com/",function() {
    //this.capture("init.png");
})
.repeat(1000, function(){
    console.log('CMD INFO WAITING FOR SERVICE');
    interact = system.stdin.readLine();
    console.log('CMD INFO STARTED');
    exported = true;

    casper.waitUntilVisible("#lbNormal",function() {
        system.stdout.writeLine("CMD GET ARGS /username/password");
        username = system.stdin.readLine();
        password = system.stdin.readLine();
        this.sendKeys("#idInputLine input", username, {reset: true});
        this.sendKeys("#pwdInputLine input", password, {reset: true});      
        this.click("#loginBtn");
        casper.waitUntilVisible(".error-tt", function(){
            console.log('CMD INFO CONTENT password is wrong!');
            this.click("#pwdInput");
            casper.bypass(1);
        }, function(){},1000);

        casper.waitUntilVisible("#login_link", function(){
                this.click("#login_link");
                //this.debugHTML();
                this.captureSelector("images/" + interact + 'randcode.png', '#randomNoImg');
                console.log("CMD INFO RANDCODE" +interact + 'randcode.png');
                system.stdout.writeLine("CMD GET ARGS /randcode");
                randcode  = system.stdin.readLine();
                this.sendKeys("#usercheckcode", randcode, {reset: true});
                this.click("#next_step");
        },function(){},1000)

        casper.then(function(){
            this.waitUntilVisible("#_mail_component_45_45", function(){
                    this.click("#_mail_component_45_45");
                }, function(){
                    if (this.exists('#errUsercheckcode')) {
                        console.log('CMD INFO CONTENT varyfy code is wrong!');
                        this.open("http://mail.163.com/");
                        this.bypass(1);
                    }
                },1000)
                .then(function() {
                    this.waitUntilVisible("div[id$=_ListDiv]", function(){
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
                        console.log('CMD INFO CONTENT' + mails);
                        this.click("#_mail_component_33_33 a");
                        this.waitUntilVisible(".relogin", function(){
                            this.click(".info a");
                            this.bypass(1);
                        }, function(){},1000);
                    }, function(){}, 1000);
                });

            
            this.waitUntilVisible("#_mail_component_51_51", function(){
                this.click("#_mail_component_51_51");
            }, function(){},1000)
                .waitUntilVisible("div[id$=_ListDiv]", function(){
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
                    console.log('CMD INFO CONTENT' + mails);
                    this.click("#_mail_component_33_33 a");
                    this.waitUntilVisible(".relogin", function(){
                        this.click(".info a");
                    }, function(){},1000);
                }, function(){}, 1000);                 
        });
    }); 
})

casper.run();
