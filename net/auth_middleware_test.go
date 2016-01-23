/*
Copyright (c) 2014 Ashley Jeffs

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

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
	"net/http"
	"path/filepath"
	"testing"
)

func TestReadGoodFile(t *testing.T) {
	logger, stats := loggerAndStats()

	absPath, err := filepath.Abs("./htpasswd_test")
	if err != nil {
		t.Errorf("Failed to make absolute path: %v", err)
		return
	}

	config := NewAuthMiddlewareConfig()
	config.Enabled = true
	config.PasswdFilePath = absPath
	authMiddleware, err := NewAuthMiddleware(config, logger, stats)
	if err != nil {
		t.Errorf("Failed to read good htpasswd file: %v", err)
		return
	}

	expectedNHashes := 4
	if expectedNHashes != len(authMiddleware.accounts) {
		t.Errorf("Read incorrect # of accounts from htpasswd: %v != %v", len(authMiddleware.accounts), expectedNHashes)
		return
	}
}

func TestReadBadFile(t *testing.T) {
	logger, stats := loggerAndStats()

	absPath, err := filepath.Abs("./htpasswd_bad_test")
	if err != nil {
		t.Errorf("Failed to make absolute path: %v", err)
		return
	}

	config := NewAuthMiddlewareConfig()
	config.Enabled = true
	config.PasswdFilePath = absPath
	if _, err := NewAuthMiddleware(config, logger, stats); err == nil {
		t.Error("Error not returned from bad htpasswd file")
	}
}

type TestResponseWriter struct {
	status int
	data   []byte
	header http.Header
}

func NewTestResponseWriter() *TestResponseWriter {
	return &TestResponseWriter{
		status: 200,
		data:   []byte{},
		header: http.Header{},
	}
}

func (t *TestResponseWriter) Header() http.Header {
	return t.header
}

func (t *TestResponseWriter) Write(data []byte) (int, error) {
	t.data = append(t.data, data...)
	return len(data), nil
}

func (t *TestResponseWriter) WriteHeader(header int) {
	t.status = header
}

var TestHandler = func(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(200)
	w.Write([]byte("authenticated"))
}

type userTest struct {
	username     string
	password     string
	expectedPass bool
}

type emptyReader struct{}

func (e emptyReader) Read(p []byte) (n int, err error) { return 0, nil }

func TestBasicAccess(t *testing.T) {
	logger, stats := loggerAndStats()

	absPath, err := filepath.Abs("./htpasswd_test")
	if err != nil {
		t.Errorf("Failed to make absolute path: %v", err)
		return
	}

	config := NewAuthMiddlewareConfig()
	config.Enabled = true
	config.PasswdFilePath = absPath
	authMiddleware, err := NewAuthMiddleware(config, logger, stats)
	if err != nil {
		t.Errorf("Failed to read good htpasswd file: %v", err)
		return
	}
	handler := authMiddleware.WrapHandlerFunc(TestHandler)

	userTests := []userTest{
		{"hello", "world", true},
		{"noone", "ponies", false},
		{"test", "account", true},
		{"nope", "chess", false},
		{"secure", "password123", true},
		{"non-user", "doesntmatter", false},
		{"bcrypt_guy1", "iamlegend", true},
	}
	for _, test := range userTests {
		testBytes := emptyReader{}
		request, err := http.NewRequest("GET", "localhost", testBytes)
		if err != nil {
			t.Errorf("Failed to create request: %v", err)
			continue
		}
		request.SetBasicAuth(test.username, test.password)

		response := NewTestResponseWriter()
		handler(response, request)

		if test.expectedPass {
			if response.status != 200 {
				t.Errorf("Correct credentials rejected: %v != %v", response.status, 200)
			}
			if string(response.data) != "authenticated" {
				t.Error("Failed to pass request to wrapped handler")
			}
		} else {
			if response.status != 401 {
				t.Errorf("Incorrect credentials non 401 response: %v != %v", response.status, 401)
			}
			if string(response.data) == "authenticated" {
				t.Error("Unauthed user passed to protected wrapped handler")
			}
		}
	}
}
