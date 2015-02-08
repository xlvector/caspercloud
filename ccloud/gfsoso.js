var casper = require('casper').create();
var system = require('system');

casper.options.onResourceRequested = function(C, requestData, request) {
};

casper.options.onResourceReceived = function(C, response) {
    //console.log('download ' + JSON.stringify(response));
};

casper.userAgent("Mozilla/5.0 (Macintosh; Intel Mac OS X 10_10_0) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/39.0.2171.71 Safari/537.36");

var interact = "";

casper.start("http://www.gfsoso.com/?q=null", function() {})
.repeat(10000, function(){
    console.log('CMD INFO WAITING FOR SERVICE')
    interact = system.stdin.readLine();
    console.log('CMD INFO STARTED')
    casper.waitUntilVisible('.input-append', function() {
        system.stdout.writeLine("CMD GET ARGS /word");
        var word = system.stdin.readLine();
        this.sendKeys("#lst-ib", word, {reset: true});
        this.click(".lsbb"); 
    }, function(){
        console.log('CMD INFO CONTENT time out');
    }, 5000)
    .waitUntilVisible('#center_col', function(){
        var list = this.evaluate(function(){
            var news = document.querySelectorAll('.g .r a');
            ret = [];
            for(var i = 0; i < news.length; i++){
                ret.push({"title":news[i].innerText});
            }
            return JSON.stringify(ret);
        });
        console.log('CMD INFO CONTENT' + list);
        console.log('CMD INFO FINISHED')

    }, function(){
        console.log('CMD INFO CONTENT time out');
    },1000);
})

casper.run();