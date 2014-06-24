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

package main

import (
	"fmt"
	"github.com/jeffail/leaps/leaplib"
	"github.com/jeffail/leaps/leapnet"
	"net/http"
	"os"
	"os/signal"
)

func main() {
	curatorConfig := leaplib.DefaultCuratorConfig()
	curatorConfig.LogVerbose = true

	curatorConfig.BinderConfig.LogVerbose = true
	curatorConfig.StoreConfig.Type = "mock"
	curatorConfig.StoreConfig.Name = "test_document"

	httpServerConfig := leapnet.DefaultHTTPServerConfig()
	httpServerConfig.LogVerbose = true

	fmt.Printf("Launching a leaps example server, use CTRL+C to close.\n\n")

	curator, err := leaplib.CreateNewCurator(curatorConfig)
	if err != nil {
		fmt.Printf("Curator error: %v\n", err)
		return
	}

	leapHttp, err := leapnet.CreateHTTPServer(curator, httpServerConfig)
	if err != nil {
		fmt.Printf("Http create error: %v\n", err)
		return
	}

	http.Handle("/", http.StripPrefix("/", http.FileServer(http.Dir("./files"))))

	closeChan := make(chan bool)

	go func() {
		if err := leapHttp.Listen(); err != nil {
			fmt.Printf("Http listen error: %v\n", err)
		}
		closeChan <- true
	}()

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	select {
	case <-c:
	case <-closeChan:
	}

	curator.Close()
}
