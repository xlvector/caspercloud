var casper = require('casper').create();

var endpoint = casper.cli.get('endpoint');
var id = parseInt(casper.cli.raw.get('transport_id'));
var total = parseInt(casper.cli.raw.get('total'));
var part = parseInt(casper.cli.raw.get('part'));
var proc = parseInt(casper.cli.raw.get('proc'));

var n = 0;
var n1 = 0;
var e1 = 0;
var n2 = 0;
var e2 = 0;
var n3 = 0;
var e3 = 0;
//var timeout = 0;
var start = 0;
var time_consume = 0;
var wait_time_ms = 3000;

var total_fields = 0;
var zero_fields = 0;

function parseDate(str){
    if(str == undefined || str == null) return new Date();
    tks = str.split(" ");
    if(tks.length != 2) return new Date();
    ymd = tks[0].split("-");
    hms = tks[1].split(":");
    var ret = new Date();

    ret.setFullYear(parseInt(ymd[0], 10));
    ret.setMonth(parseInt(ymd[1], 10) - 1);
    ret.setDate(parseInt(ymd[2], 10));
    ret.setHours(parseInt(hms[0], 10));
    ret.setMinutes(parseInt(hms[1], 10));
    ret.setSeconds(parseInt(hms[2], 10));
    return ret;
}

casper.start(endpoint + '#?use_cache=false&debug=true', function() {
    this.echo("\n" + part + " : " + id);
    this.sendKeys('input[ng-model="username"]', 'stdyisou');
    this.sendKeys('input[ng-model="password"]', 'stdMojiti');
    this.click('.login-btn');
    id = id - proc - total;
})
.repeat(proc, function(){
    casper.waitUntilVisible('.query-input', function() {
        id += total;
        n += 1;

        this.echo("\n---------------\n" + part + " : " + id + " , " + n);
        this.sendKeys('.query-input', id.toString(), {reset: true});

        start = (new Date()).getTime();
        this.click('.query-btn');
        }, function(){}, 10000)
        .wait(wait_time_ms, function(){
            time_consume += (new Date()).getTime() - start;
            this.echo('avg consumed time ' + (1.0 * time_consume / n - wait_time_ms).toString() + " ms");

            create_date =
                this.evaluate(
                              function(){
                                  return document.getElementById('transport_date').innerText;
                              });
            this.echo('create_date: ' + create_date);
            this.echo('create_date: ' + parseDate(create_date));
            this.echo('now_date: ' + new Date());
            cd = (new Date()).getTime();
            if(create_date != null && create_date.length != null && create_date.length > 0 && parseDate(create_date) != null){
                cd = parseDate(create_date).getTime();

                now = (new Date()).getTime();
                this.echo(now - cd);

                is_finish = 1;

                //未完成自动网查
                if(this.exists('.non-check-offline-text')){
                    is_finish = 0;
                    this.echo('does not finish, non-check-offline');
                }

                if((now - cd) < 600000){
                    n1 += 1;
                    e1 += 1 - is_finish;
                }else{
                    if((now - cd) < 1800000){
                        n2 += 1;
                        e2 += 1 - is_finish;
                    }
                    else{
                        n3 += 1;
                        e3 += 1 - is_finish;
                    }
                }
                if((now - cd) < 600000) return;
                var ret = [];
                nmobile = this.evaluate(function(){
                        return document.querySelector("#kwebmobile .notice").innerText;
                    });
                ret.push(nmobile);
                nhomeaddr = this.evaluate(function(){
                        return document.querySelector("#kwebhomeaddr .notice").innerText;
                    });
                ret.push(nhomeaddr);
                nnamecity = this.evaluate(function(){
                        return document.querySelector("#kwebnamecity .notice").innerText;
                    });
                ret.push(nnamecity);
                norg = this.evaluate(function(){
                        return document.querySelector("#kweborg .notice").innerText;
                    });
                ret.push(norg);
                norgaddr = this.evaluate(function(){
                        return document.querySelector("#kweborgaddr .notice").innerText;
                    });
                ret.push(norgaddr);
                norgtel = this.evaluate(function(){
                        return document.querySelector("#kweborgtel .notice").innerText;
                    });
                ret.push(norgtel);
                nc1 = this.evaluate(function(){
                        return document.querySelector("#kwebc1 .notice").innerText;
                    });
                ret.push(nc1);
                nc2 = this.evaluate(function(){
                        return document.querySelector("#kwebc2 .notice").innerText;
                    });
                ret.push(nc2);
                nc3 = this.evaluate(function(){
                        return document.querySelector("#kwebc3 .notice").innerText;
                    });
                ret.push(nc3);
                for(var i = 0; i < ret.length; i++){
                    if(ret[i] == null || ret[i] == "") continue;
                    this.echo(ret[i]);
                    if(parseInt(ret[i]) == 0) zero_fields += 1;
                    total_fields += 1;
                }
            }
        })
})
    .then(function(){
            //this.echo('\ntimeout ratio : ' + (timeout * 1.0 / n).toString());
            this.echo('\n\nempty ratio < 10 min : ' + e1.toString() + "/" + n1.toString() + '=' + (e1 * 1.0 / n1).toString());
            this.echo('empty ratio in (10 - 30) min : ' + e2.toString() + "/" + n2.toString() + '=' + (e2 * 1.0 / n2).toString());
            this.echo('empty ratio > 30 min : ' + e3.toString() + "/" + n3.toString() + '=' + (e3 * 1.0 / n3).toString());
            this.echo('total empty ratio: ' + ((e1+e2+e3)*1.0 / (n1+n2+n3)*1.0).toString() )
            this.echo('avg consumed time : ' + (time_consume * 1.0 / n - wait_time_ms).toString());
            this.echo('field empty ratio : ' + (zero_fields * 1.0 / total_fields).toString());
            this.echo('zero field: ' + zero_fields.toString());
            this.echo('total field: ' + total_fields.toString() + '\n');

            var now_hour = (new Date()).getHours();
            this.echo('hour now: ' + now_hour)

            //8:00 ~ 19:59
            if ( now_hour > 7 && now_hour < 20 ){
                //< 10 min 抓取失败率超过15%
                if( n1 > 0 && (e1 * 1.0 / n1) >= 0.15){
                    this.exit(1);
                }
                //(10 - 30) min 抓取失败率超过4%
                if(n2 > 0 &&  (e2 * 1.0 / n2) >= 0.04){
                    this.exit(1);
                }
                // >30 min 抓取失败率超过1%
                if(n3 > 0 && (e2 * 1.0 / n2) >= 0.01){
                    this.exit(1);
                }
            }else{
                if( n1 > 0 && (e1 * 1.0 / n1) >= 0.3){
                    this.exit(1);
                }
                if(n2 > 0 &&  (e2 * 1.0 / n2) >= 0.09){
                    this.exit(1);
                }
                if(n3 > 0 && (e2 * 1.0 / n2) >= 0.07){
                    this.exit(1);
                }
            }

            //平均耗时大于80ms
            if((time_consume * 1.0 / n - wait_time_ms) >= 80){
                this.exit(1);
            }

            /*
            if(zero_fields * 1.0 / total_fields > 0.05){
                this.exit(1);
            }
            */
        })
.run();