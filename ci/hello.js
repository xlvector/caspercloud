var casper = require('casper').create();
var system = require('system');

var query = "";

casper.start("http://127.0.0.1:8000/hello?query=test", function() {
    system.stdout.writeLine("CMD GET ARGS /query");
    query = system.stdin.readLine();
})
.repeat(100, function(){
    casper.thenOpen("http://127.0.0.1:8000/hello?query=" + query, function(){
        var result = this.evaluate(function(){
            return document.querySelector("#query").innerText;
        })
        system.stdout.writeLine("CMD INFO CONTENT" + result);
        system.stdout.writeLine("CMD GET ARGS /query");
        query = system.stdin.readLine();
    });
})

casper.run();

