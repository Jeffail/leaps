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

var leap_str = require('../leapclient').str;

function test_content(test) {
	var good_cases = [
		"hello world",
		"",
		'',
		'a',
		['a','b','c'],
		new String("test"),
		new String(),
		new leap_str('')
	];

	for ( var index = 0; index < good_cases.length; ++index ) {
		try {
			var tmp = new leap_str(good_cases[index]);
		} catch (e) {
			test.ok(false, e.what());
		}
	}
};

function test_strings(test) {
	var cases = [
		{ input: "hello world", u: ['h','e','l','l','o',' ','w','o','r','l','d'], s: "hello world" }
	];

	for ( var index = 0; index < cases.length; ++index ) {
		try {
			var tmp = new leap_str(cases[index].input);
			test.ok(cases[index].s === tmp.str(), "Non-matching regular strings: " + cases[index].s + " != " + tmp.str());
			test.ok(JSON.stringify(cases[index].u) === JSON.stringify(tmp.u_str()),
					"Non-matching codepoint arrays: " + cases[index].u + " != " + tmp.u_str());
		} catch (e) {
			test.ok(false, e.what());
		}
	}
};

module.exports = function(test) {
	"use strict";

	test_content(test);
	test_strings(test);

	test.done();
};

/*--------------------------------------------------------------------------------------------------
 */
