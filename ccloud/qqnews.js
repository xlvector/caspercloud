var casper = require('casper').create();
var system = require('system');

casper.options.onResourceRequested = function(C, requestData, request) {
};

casper.options.onResourceReceived = function(C, response) {
    //console.log('download ' + JSON.stringify(response));
};

casper.userAgent("Mozilla/5.0 (Macintosh; Intel Mac OS X 10_10_0) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/39.0.2171.71 Safari/537.36");

var interact = "";

casper.start("http://roll.news.qq.com/", function() {})
.repeat(3, function(){
    console.log('CMD INFO WAITING FOR SERVICE')
    interact = system.stdin.readLine();
    console.log('CMD INFO STARTED')
    casper.waitUntilVisible('.RefreshBtn', function() {
        this.click(".RefreshBtn"); 
    }, function(){}, 10000)
    .waitFor(function check() {
        return this.evaluate(function() {
            return document.querySelectorAll('#artContainer li a').length > 2;
        });
    }, function then() {
        var list = this.evaluate(function(){
            var news = document.querySelectorAll('#artContainer li a');
            ret = [];
            for(var i = 0; i < news.length; i++){
                ret.push(news[i].innerText);
            }
            return JSON.stringify(ret);
        });
        console.log('CMD INFO CONTENT' + list);
        this.click(".qm_footer_userInfo a");
        console.log('CMD INFO FINISHED');
    });
})

casper.run();




