var interact = "";
var casper = require('casper').create();
//casper.options.waitTimeout = 2000000;
var system = require('system');
casper.options.onResourceRequested = function(C, requestData, request) {
    /*
    if ((/http:.+?.gif/gi).test(requestData['url']) 
        || (/http:.+?.png/gi).test(requestData['url']) ) {
        request.abort();
    }
    */
    //console.log(requestData['url']);
};

casper.userAgent("Mozilla/5.0 (Macintosh; Intel Mac OS X 10_9_5) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/40.0.2214.93 Safari/537.36");

casper.start("https://w.mail.qq.com/cgi-bin/loginpage?t=loginpage", function() {
    //this.capture("init.png");
})
.repeat(1000, function(){
    casper.waitUntilVisible("#loginform", function(){
        console.log('CMD INFO WAITING FOR SERVICE');
        interact = system.stdin.readLine();
        system.stdout.writeLine("CMD GET ARGS /username/password");
        var name = system.stdin.readLine();
        var password = system.stdin.readLine();
        //this.capture("login.png");
        this.sendKeys("#uin", name, {reset: true});
        this.sendKeys("#pwd", password, {reset: true});
        this.click("#submitBtn");
        /*
        casper.waitUntilVisible(".qm_tips_ErrorInfo", function(){
            console.log('CMD INFO CONTENT password is wrong!');
            //this.click("#uin");
            casper.bypass(1);
        });*/

        casper.then(function(){
            this.waitUntilVisible(".folderlist_content", function(){
                    this.click("a.qm_list_item_content");
                },function(){
                if (this.exits('.qm_verifyImg_img')) {
                    this.captureSelector("images/" + interact + 'randcode.png', '.qm_verifyImg_img');
                    console.log("CMD INFO RANDCODE" +interact + 'randcode.png');
                    system.stdout.writeLine("CMD GET ARGS /randcode");
                    var randcode  = system.stdin.readLine();
                    this.sendKeys(".qm_formText", randcode, {reset: true});
                    this.click("#submitBtn");
                } else {
                    var msg = new String(this.fetchText('#validcodeMsg')) ;
                    if (msg.length > 0) {
                        console.log('CMD INFO CONTENT password is wrong!');
                        this.click("#uin");
                        this.bypass(1);
                    }
                }
                //console.log('CMD INFO CONTENT password is wrong!'); 
                //this.debugHTML();
            },1000);
       
            this.waitUntilVisible(".maillist_listItem", function(){
                //this.capture("email.png");
                //this.captureSelector('weather.png', '.readmail_list');
                var content = this.evaluate(function() {
                    //console.log("get content: " + document.querySelector(".readmail_list").src);
                    //return document.querySelector(".readmail_list").innerText;
                    var list = document.querySelectorAll(".readmail_list .maillist_listItem ");
                    //return list.length; 
                    var ret = [];
                    for(var i = 0; i < list.length; i++){
                        ret.push({
                            title: list[i].querySelector("div.maillist_listItemLineSecond").innerText, 
                            from: list[i].querySelector("div.maillist_listItem_title").innerText,
                            content: list[i].querySelector("div.maillist_listItem_abstract").innerText
                        });
                    }
                    return JSON.stringify(ret);
                });
                console.log('CMD INFO CONTENT' + content);
                this.click(".qm_footer_userInfo a");
            });

        });

    });
})
casper.run();