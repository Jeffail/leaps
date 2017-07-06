/*
Copyright (c) 2014 Ashley Jeffs

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, sub to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
*/

/*--------------------------------------------------------------------------------------------------
 */

var la = require('../leapclient').apply,
    leap_str = require('../leapclient').str;

var content = "hello world";

var tests = [
	{ transform : { position : 3, insert : new leap_str("123"), num_delete : 0 }, result : "hel123lo world" },
	{ transform : { position : 3, insert : new leap_str("123"), num_delete : 3 }, result : "hel123world" },
	{ transform : { position : 0, insert : new leap_str(""), num_delete : 5 }, result : " world" }
];

module.exports = function(test) {
	"use strict";

	for ( var i = 0, l = tests.length; i < l; i++ ) {
		var result = la(tests[i].transform, content);
		test.ok(tests[i].result === result, tests[i].result + " != " + result);
	}
	test.done();
};

/*--------------------------------------------------------------------------------------------------
 */
