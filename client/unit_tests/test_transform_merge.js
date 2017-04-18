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
var model = new (require('../leapclient')._model)(0);

var tests = [
	{
		first : { position : 5, insert : new leap_str("hello"), num_delete : 0 },
		second : { position : 10, insert : new leap_str(" world"), num_delete : 0 },
		result : { position : 5, insert : new leap_str("hello world"), num_delete : 0 }
	},
	{
		first : { position : 5, insert : new leap_str("hello"), num_delete : 4 },
		second : { position : 10, insert : new leap_str(" world"), num_delete : 3 },
		result : { position : 5, insert : new leap_str("hello world"), num_delete : 7 }
	},
	{
		first : { position : 5, insert : new leap_str("hello"), num_delete : 2 },
		second : { position : 5, insert : new leap_str("j"), num_delete : 1 },
		result : { position : 5, insert : new leap_str("jello"), num_delete : 2 }
	},
	{
		first : { position : 5, insert : new leap_str("hello"), num_delete : 0 },
		second : { position : 7, insert : new leap_str("y world"), num_delete : 4 },
		result : { position : 5, insert : new leap_str("hey world"), num_delete : 1 }
	},
	{
		first : { position : 5, insert : new leap_str("0"), num_delete : 1 },
		second : { position : 6, insert : new leap_str("1"), num_delete : 1 },
		result : { position : 5, insert : new leap_str("01"), num_delete : 2 }
	}
];

module.exports = function(test) {
	"use strict";

	for ( var i = 0, l = tests.length; i < l; i++ ) {
		test.ok(model._merge_transforms(tests[i].first, tests[i].second), "merge " + (i+1) + " failed");
		var result = JSON.stringify(tests[i].first);
		var expected = JSON.stringify(tests[i].result);

		test.ok(result === expected, result + " != " + expected);
	}
	test.done();
};

/*--------------------------------------------------------------------------------------------------
 */
