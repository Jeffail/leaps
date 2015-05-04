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

package net

import (
	"fmt"
	"io/ioutil"
	"time"
)
import "net/http"
import "testing"

/*--------------------------------------------------------------------------------------------------
 */

type FakeAdmin struct{}

func (f FakeAdmin) KickUser(doc, user string, timeout time.Duration) error {
	return nil
}

func TestEndpointsEndpoint(t *testing.T) {
	log, stats := loggerAndStats()

	config := NewInternalServerConfig()
	config.Address = "localhost:8767"
	config.Path = "/internal"

	admin := FakeAdmin{}

	internalServer, err := NewInternalServer(admin, config, log, stats)
	if err != nil {
		t.Errorf("Error creating server: %v\n", err)
		return
	}

	internalServer.Register("/first", "The first endpoint", func(http.ResponseWriter, *http.Request) {})
	internalServer.Register("/second", "The second endpoint", func(http.ResponseWriter, *http.Request) {})
	internalServer.Register("/third", "The third endpoint", func(http.ResponseWriter, *http.Request) {})

	go internalServer.Listen()

	res, err := http.Get("http://localhost:8767/internal/endpoints")
	if err != nil {
		t.Errorf("Error getting endpoints from server: %v\n", err)
		return
	}

	bytes, err := ioutil.ReadAll(res.Body)
	if err != nil {
		t.Errorf("Error reading response from server: %v\n", err)
		return
	}

	expectedEndpoints := "/internal/endpoints: <GET> the available endpoints of this leaps API\n" +
		`/internal/kick_user: <POST> Kick a user from a document {"user_id":"<id>", "doc_id":"<id>"}` +
		"\n/internal/first: The first endpoint\n" +
		"/internal/second: The second endpoint\n" +
		"/internal/third: The third endpoint\n"

	if string(bytes) != expectedEndpoints {
		t.Errorf("Endpoints endpoint failed:\n%v != \n%v", expectedEndpoints, string(bytes))
	}
}

func TestRegisterEndpoint(t *testing.T) {
	log, stats := loggerAndStats()

	config := NewInternalServerConfig()
	config.Address = "localhost:8768"
	config.Path = "/internal"

	admin := FakeAdmin{}

	internalServer, err := NewInternalServer(admin, config, log, stats)
	if err != nil {
		t.Errorf("Error creating server: %v\n", err)
		return
	}

	endpointTests := []string{
		"first",
		"second",
		"third",
	}

	for _, e := range endpointTests {
		internalServer.Register("/"+e, "no", func(epnt string) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				fmt.Fprintf(w, epnt)
			}
		}(e))
	}

	go internalServer.Listen()

	for _, e := range endpointTests {
		res, err := http.Get("http://localhost:8768/internal/" + e)
		if err != nil {
			t.Errorf("Error getting endpoint from server: %v\n", err)
			return
		}

		bytes, err := ioutil.ReadAll(res.Body)
		if err != nil {
			t.Errorf("Error reading response from server: %v\n", err)
			return
		}

		if string(bytes) != e {
			t.Errorf("Endpoint register failed:\n%v != \n%v", e, string(bytes))
		}
	}
}

/*--------------------------------------------------------------------------------------------------
 */
