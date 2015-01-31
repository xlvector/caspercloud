Casper Cloud
=================

![Travis Status](https://travis-ci.org/xlvector/caspercloud.svg)

基准测试
------------

我们Mock了一个服务他的平均响应时间非常短，他产生的HTML非常简单
    
    <html><head><title>Hello World</title></head><body><h1>Hello World</h1></body></html>

然后我们测试三种下载方式的下载时间

    1. curl
    2. golang http.Client
    3. casper js

结果如下

    BenchmarkCurl-4         2000        7960292 ns/op
    BenchmarkHttpClient-4   100000      179471 ns/op
    BenchmarkCasperJs-4     5           3091975091 ns/op

可以看到，Curl启动进程的时间花费了7ms，http.Client启动进程的时间为0， 而CasperJs启动进程的时间最长，大概需要3s。

但是，如果节省CasperJs打开进程的时间，那么实际下载时间是多少？

我们写了两段js

mock.js

    var casper = require("casper").create();
    casper.start("http://127.0.0.1:20893/hello", function(){
        console.log(this.getHTML());
    });
    casper.run();

mock_100.js

    var casper = require("casper").create();
    casper.start("http://127.0.0.1:20893/hello", function(){
        console.log(this.getHTML());
    }).repeat(99, function(){
        casper.thenOpen("http://127.0.0.1:20893/hello", function(){
            console.log(this.getHTML());
        });
    });
    casper.run();

在mock.js中，我们打开一次casperjs只下载一个链接，而在mock_100.js中，我们打开一次下载100个链接。然后我们在服务端让程序10ms返回。再做一次测试，结果如下：

    BenchmarkCasperJs-4        20   2824344729 ns/op
    BenchmarkCasperJs100-4         5    9634887107 ns/op

可以看到，在批量下载的程序里，一次下载的时间是 (9635 - 2824) / 99 = 68ms

同样的，我们也对比一下curl的单次下载和批量下载，结果如下

    BenchmarkCurl-4     2000      22667633 ns/op
    BenchmarkCurl100-4        30    1202024886 ns/op

可以看到，curl的一次下载时间是 (1202 - 22) / 99 = 11ms

从而可以看到，在单次下载中，casperjs的耗时是curl的6倍左右。
