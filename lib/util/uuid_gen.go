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

/*
Package util - Various miscellaneous utilities used throughout leaps lib.
*/
package util

import (
	"fmt"
	"log"
	"time"

	"github.com/gofrs/uuid"
)

/*--------------------------------------------------------------------------------------------------
 */

/*
GenerateStampedUUID - Generates a UUID and prepends a timestamp to it.
*/
func GenerateStampedUUID() string {
	tstamp := time.Now().Unix()

	return fmt.Sprintf("%v%v", tstamp, GenerateUUID())
}

/*
GenerateUUID - Generates a UUID and returns it as a hex encoded string.
*/
// TODO: handle the error more gracefully
func GenerateUUID() string {
	u, err := uuid.NewV4()
	if err != nil {
		log.Fatal(err)
	}
	return u.String()
}

/*--------------------------------------------------------------------------------------------------
 */
