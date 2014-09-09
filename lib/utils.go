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

package lib

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"math/rand"
	"time"
)

/*--------------------------------------------------------------------------------------------------
 */

/*
GenerateID - Create a unique ID for using a hash of a string, a timestamp and a pseudo random
integer as a hex encoded string with the timestamp on the end.
*/
func GenerateID(content string) string {
	tstamp := time.Now().Unix()
	hasher := sha1.New()
	hasher.Write([]byte(fmt.Sprintf("%v%v%v", content, rand.Int(), tstamp)))

	return fmt.Sprintf("%v%v", hex.EncodeToString(hasher.Sum(nil)), tstamp)
}

/*--------------------------------------------------------------------------------------------------
 */
