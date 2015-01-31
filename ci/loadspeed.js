t = Date.now();
var page = require('webpage').create(),
    system = require('system'),
    t, address;

if (system.args.length === 1) {
    console.log('Usage: loadspeed.js <some URL>');
    phantom.exit(1);
} else {
    address = system.args[1];
    page.open(address, function (status) {
	    if (status !== 'success') {
		console.log('FAIL to load the address');
	    } else {
		t = Date.now() - t;
		console.log('Loading time ' + t + ' msec');
	    }
	    phantom.exit();
	});
}