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
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path"
	"runtime"
	"syscall"

	"github.com/kardianos/osext"

	"github.com/jeffail/leaps/lib"
	"github.com/jeffail/leaps/net"
	"github.com/jeffail/util"
	"github.com/jeffail/util/log"
)

/*--------------------------------------------------------------------------------------------------
 */

/*
LeapsConfig - The all encompassing leaps configuration. Contains configurations for individual leaps
components, which determine the role of this leaps instance. Currently a stand alone leaps server is
the only supported role.
*/
type LeapsConfig struct {
	NumProcesses        int                          `json:"num_processes" yaml:"num_processes"`
	LoggerConfig        log.LoggerConfig             `json:"logger" yaml:"logger"`
	StatsConfig         log.StatsConfig              `json:"stats" yaml:"stats"`
	StoreConfig         lib.DocumentStoreConfig      `json:"storage" yaml:"storage"`
	AuthenticatorConfig lib.TokenAuthenticatorConfig `json:"authenticator" yaml:"authenticator"`
	CuratorConfig       lib.CuratorConfig            `json:"curator" yaml:"curator"`
	HTTPServerConfig    net.HTTPServerConfig         `json:"http_server" yaml:"http_server"`
	StatsServerConfig   log.StatsServerConfig        `json:"stats_server" yaml:"stats_server"`
}

/*--------------------------------------------------------------------------------------------------
 */

var (
	sharePathOverride *string
)

func init() {
	sharePathOverride = flag.String("share", "", "Override the path for file system sharing configs")
}

/*--------------------------------------------------------------------------------------------------
 */

func main() {
	var (
		curator   net.LeapLocator
		err       error
		closeChan = make(chan bool)
	)

	leapsConfig := LeapsConfig{
		NumProcesses:        runtime.NumCPU(),
		LoggerConfig:        log.DefaultLoggerConfig(),
		StatsConfig:         log.DefaultStatsConfig(),
		StoreConfig:         lib.DefaultDocumentStoreConfig(),
		AuthenticatorConfig: lib.DefaultTokenAuthenticatorConfig(),
		CuratorConfig:       lib.DefaultCuratorConfig(),
		HTTPServerConfig:    net.DefaultHTTPServerConfig(),
		StatsServerConfig:   log.DefaultStatsServerConfig(),
	}

	// A list of default config paths to check for if not explicitly defined
	defaultPaths := []string{}

	if executablePath, err := osext.ExecutableFolder(); err == nil {
		defaultPaths = append(defaultPaths, path.Join(executablePath, "config.yaml"))
		defaultPaths = append(defaultPaths, path.Join(executablePath, "config", "leaps.yaml"))
		defaultPaths = append(defaultPaths, path.Join(executablePath, "config.json"))
		defaultPaths = append(defaultPaths, path.Join(executablePath, "config", "leaps.json"))
	}

	defaultPaths = append(defaultPaths, []string{
		path.Join(".", "leaps.yaml"),
		path.Join(".", "leaps.json"),
		"/etc/leaps.yaml",
		"/etc/leaps.json",
		"/etc/leaps/config.yaml",
		"/etc/leaps/config.json",
	}...)

	// Load configuration etc
	if !util.Bootstrap(&leapsConfig, defaultPaths...) {
		return
	}

	if len(*sharePathOverride) > 0 {
		leapsConfig.AuthenticatorConfig.FileConfig.SharePath = *sharePathOverride
		leapsConfig.StoreConfig.StoreDirectory = *sharePathOverride
	}

	runtime.GOMAXPROCS(leapsConfig.NumProcesses)

	logger := log.NewLogger(os.Stdout, leapsConfig.LoggerConfig)
	stats := log.NewStats(leapsConfig.StatsConfig)

	fmt.Printf("Launching a leaps instance, use CTRL+C to close.\n\n")

	// Document storage engine
	documentStore, err := lib.DocumentStoreFactory(leapsConfig.StoreConfig)
	if err != nil {
		fmt.Fprintln(os.Stderr, fmt.Sprintf("Document store error: %v\n", err))
		return
	}

	// Authenticator
	authenticator, err := lib.TokenAuthenticatorFactory(leapsConfig.AuthenticatorConfig, logger)
	if err != nil {
		fmt.Fprintln(os.Stderr, fmt.Sprintf("Authenticator error: %v\n", err))
		return
	}

	// Curator of documents
	curator, err = lib.NewCurator(leapsConfig.CuratorConfig, logger, stats, authenticator, documentStore)
	if err != nil {
		fmt.Fprintln(os.Stderr, fmt.Sprintf("Curator error: %v\n", err))
		return
	}

	// HTTP API
	leapHTTP, err := net.CreateHTTPServer(curator, leapsConfig.HTTPServerConfig, logger, stats)
	if err != nil {
		fmt.Fprintln(os.Stderr, fmt.Sprintf("HTTP error: %v\n", err))
		return
	}

	go func() {
		if httperr := leapHTTP.Listen(); httperr != nil {
			fmt.Fprintln(os.Stderr, fmt.Sprintf("Http listen error: %v\n", httperr))
		}
		closeChan <- true
	}()

	// Internal Statistics HTTP API
	statsServer, err := log.NewStatsServer(leapsConfig.StatsServerConfig, logger, stats)
	if err != nil {
		fmt.Fprintln(os.Stderr, fmt.Sprintf("Stats error: %v\n", err))
		return
	}

	go func() {
		if statserr := statsServer.Listen(); statserr != nil {
			fmt.Fprintln(os.Stderr, fmt.Sprintf("Stats server listen error: %v\n", statserr))
		}
	}()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Wait for termination signal
	select {
	case <-sigChan:
	case <-closeChan:
	}

	leapHTTP.Stop()
	curator.Close()
}

/*--------------------------------------------------------------------------------------------------
 */
