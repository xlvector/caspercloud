var casper = require('casper').create();
var system = require('system');

casper.start("http://127.0.0.1:8000/form/init", function() {
    system.stdout.writeLine("CMD INFO CONTENT phone");
    system.stdout.writeLine("CMD GET ARGS /phone");
    var phone = system.stdin.readLine();
    this.sendKeys("#phone", phone);
    this.click("#submit");
})
.waitUntilVisible("#verify_code", function(){
    system.stdout.writeLine("CMD INFO CONTENT verify_code");
    system.stdout.writeLine("CMD GET ARGS /verify_code");
    var verify_code = system.stdin.readLine();
    if(verify_code != "123456"){
        system.stdout.writeLine("CMD INFO CONTENT wrong verify code")
    } else {
        this.sendKeys("#verify_code", verify_code);
        this.click("#submit");
    }
})
.waitUntilVisible("h1", function(){
    var result = this.evaluate(function(){
        return document.querySelector("h1").innerText;
    });
    system.stdout.writeLine("CMD INFO CONTENT " + result);
})

casper.run();

