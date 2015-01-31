var casper = require("casper").create();

casper.start("http://127.0.0.1:20893/hello", function(){
    console.log(this.getHTML());
});

casper.run();
