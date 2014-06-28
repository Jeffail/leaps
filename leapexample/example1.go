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

/*
ExampleConfig - Example of a full server configuration, containing configurations for each component.
*/
type ExampleConfig struct {
	CuratorConfig     leaplib.CuratorConfig
	HTTPServerConfig  leapnet.HTTPServerConfig
	StatsServerConfig leapnet.StatsServerConfig
}

func main() {
	exampleConfig := ExampleConfig{
		CuratorConfig:     leaplib.DefaultCuratorConfig(),
		HTTPServerConfig:  leapnet.DefaultHTTPServerConfig(),
		StatsServerConfig: leapnet.DefaultStatsServerConfig(),
	}

	exampleConfig.CuratorConfig.StoreConfig.Type = "mock"
	exampleConfig.CuratorConfig.StoreConfig.Name = "test_document"

	exampleConfig.StatsServerConfig.StaticFilePath = "./stats_files"

	fmt.Printf("Launching a leaps example server, use CTRL+C to close.\n\n")

	curator, err := leaplib.CreateNewCurator(exampleConfig.CuratorConfig)
	if err != nil {
		fmt.Printf("Curator error: %v\n", err)
		return
	}

	leapHTTP, err := leapnet.CreateHTTPServer(curator, exampleConfig.HTTPServerConfig, nil)
	if err != nil {
		fmt.Printf("Http create error: %v\n", err)
		return
	}

	http.Handle("/", http.StripPrefix("/", http.FileServer(http.Dir("./files"))))

	closeChan := make(chan bool)

	go func() {
		if err := leapHTTP.Listen(); err != nil {
			fmt.Printf("Http listen error: %v\n", err)
		}
		closeChan <- true
	}()

	statsServer, err := leapnet.CreateStatsServer(curator.GetLogger(), exampleConfig.StatsServerConfig)
	if err != nil {
		fmt.Printf("Stats server create error: %v\n", err)
		return
	}

	go func() {
		if err := statsServer.Listen(); err != nil {
			fmt.Printf("Stats server listen error: %v\n", err)
		}
	}()

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	select {
	case <-c:
	case <-closeChan:
	}

	curator.Close()
}
