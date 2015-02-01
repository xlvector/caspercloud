var totalTime = (new Date()).getTime();

var casper = require('casper').create();
var system = require('system');

casper.options.onResourceRequested = function(C, requestData, request) {
    var url = requestData['url'];
    if ((/http:.+?.(gif|png|jpg|css)/gi).test(url)) {
        request.abort();
    }
};

casper.userAgent("Mozilla/5.0 (Macintosh; Intel Mac OS X 10_10_0) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/39.0.2171.71 Safari/537.36");

var query = "";

casper.start("http://www.baidu.com/", function(){
    system.stdout.writeLine("CMD Info List: query");
    system.stdout.writeLine("CMD GET ARGS /query");
    query = system.stdin.readLine();
    this.evaluate(function(){
        document.getElementById("kw").value = "";
    });
    console.log(query);
    this.sendKeys("#kw", query);
    this.click("#su");
}).repeat(100, function(){
    casper.waitForSelectorTextChange("h3.t", function() {
        var results = this.evaluate(function(){
            return document.querySelector(".nums").innerText;
        });
        system.stdout.writeLine(results);
        system.stdout.writeLine("CMD Info List: query");
        system.stdout.writeLine("CMD GET ARGS /query");
        query = system.stdin.readLine();
        this.evaluate(function(){
            document.getElementById("kw").value = "";
        });
        this.sendKeys("#kw", query);
        this.click("#su");
    });
});


casper.run();




