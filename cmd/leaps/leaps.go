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
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	gopath "path"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/Jeffail/leaps/lib/acl"
	"github.com/Jeffail/leaps/lib/api"
	apiio "github.com/Jeffail/leaps/lib/api/io"
	"github.com/Jeffail/leaps/lib/audit"
	"github.com/Jeffail/leaps/lib/curator"
	"github.com/Jeffail/leaps/lib/store"
	"github.com/Jeffail/leaps/lib/util"
	"github.com/Jeffail/leaps/lib/util/service/log"
	"github.com/Jeffail/leaps/lib/util/service/metrics"
	"github.com/gorilla/websocket"
)

//------------------------------------------------------------------------------

// Flags

type cmdList []string

func (c *cmdList) String() string {
	return strings.Join(*c, ", ")
}

func (c *cmdList) Set(value string) error {
	*c = append(*c, value)
	return nil
}

var (
	httpAddress string
	safeMode    bool
	applyLcot   bool
	discardLcot bool
	showHidden  bool
	debugWWWDir string
	logLevel    string
	subdirPath  string
	showVersion bool
	cmds        cmdList
)

// Build Information
var (
	version   = "0.0.0"
	dateBuilt = "-"
)

func init() {
	flag.BoolVar(&showVersion, "version", false, "Show version information")
	flag.BoolVar(&safeMode, "safe", false, `Do not write changes directly to local files. Instead, store them in a temporary file that can be
	committed afterwards with the --commit flag`)
	flag.BoolVar(&applyLcot, "commit", false, "Commit changes made from leaps in safe mode to your local files and then exit (look at --safe)")
	flag.BoolVar(&discardLcot, "discard", false, "Discard changes made from leaps in safe mode and then exit (look at --safe)")
	flag.StringVar(&httpAddress, "address", ":8080", "The HTTP address to bind to")
	flag.BoolVar(&showHidden, "all", false, "Display all files, including hidden")
	flag.StringVar(&debugWWWDir, "use_www", "", "Serve alternative web files from this dir")
	flag.StringVar(&logLevel, "log_level", "INFO", "Log level (NONE, ERROR, WARM, INFO, DEBUG, TRACE)")
	flag.StringVar(&subdirPath, "path", "/", "Subdirectory (when running leaps in a webserver subdirectory as example.com/myleaps)")
	flag.Var(&cmds, "cmd", "Set commands that can be executed from the web UI, e.g. (-cmd 'make build' -cmd 'make test')")
}

//------------------------------------------------------------------------------

var endpoints = []interface{}{}

func handle(path, description string, handler http.HandlerFunc) {
	path = gopath.Join("/", subdirPath, path)
	http.HandleFunc(path, handler)
	endpoints = append(endpoints, struct {
		Path string `json:"path"`
		Desc string `json:"description"`
	}{
		Path: path,
		Desc: description,
	})
}

func writeAudit(path string, auditor *audit.ToJSON) error {
	data, err := auditor.Serialise()
	if err != nil {
		return err
	}
	return ioutil.WriteFile(path, data, 0644)
}

func readAudit(path string, auditor *audit.ToJSON, docStore store.Type) error {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}
	if err = auditor.Deserialise(data); err != nil {
		return err
	}
	return auditor.Reapply(docStore)
}

//------------------------------------------------------------------------------

type shellRunner struct{}

func (s shellRunner) CMDRun(cmdStr string) (stdout, stderr []byte, err error) {
	var errRead, outRead io.ReadCloser

	cmd := exec.Command("sh", "-c", cmdStr)

	if outRead, err = cmd.StdoutPipe(); err != nil {
		return
	}
	if errRead, err = cmd.StderrPipe(); err != nil {
		return
	}
	if err = cmd.Start(); err != nil {
		return
	}
	if stdout, err = ioutil.ReadAll(outRead); err != nil {
		return
	}
	if stderr, err = ioutil.ReadAll(errRead); err != nil {
		return
	}
	if err = cmd.Wait(); err != nil {
		return
	}
	return
}

//------------------------------------------------------------------------------

func main() {
	var (
		err       error
		closeChan = make(chan bool)
	)

	flag.Usage = func() {
		fmt.Println(`Usage: leaps [flags...] [path/to/share]

If a path is not specified the current directory is shared instead.
`)
		flag.PrintDefaults()
	}

	flag.Parse()

	if showVersion {
		fmt.Printf("Version: %v\nDate: %v\n", version, dateBuilt)
		os.Exit(0)
	}

	targetPath := "."
	if flag.NArg() == 1 {
		targetPath = flag.Arg(0)
	}

	leapsCOTPath := filepath.Join(targetPath, ".leaps_cot.json")

	// Logging and metrics aggregation
	logConf := log.NewLoggerConfig()
	logConf.Prefix = "leaps"
	logConf.LogLevel = logLevel

	logger := log.NewLogger(os.Stdout, logConf)

	statConf := metrics.NewConfig()
	statConf.Type = "http"
	statConf.HTTP.Prefix = "leaps"

	stats, err := metrics.NewHTTP(statConf)
	if err != nil {
		fmt.Fprintln(os.Stderr, fmt.Sprintf("Metrics init error: %v\n", err))
		return
	}
	defer stats.Close()

	// Document storage engine
	docStore, err := store.NewFile(targetPath, !safeMode || applyLcot)
	if err != nil {
		fmt.Fprintln(os.Stderr, fmt.Sprintf("Document store error: %v\n", err))
		os.Exit(1)
	}

	// Authenticator
	storeConf := acl.NewFileExistsConfig()
	storeConf.Path = targetPath
	storeConf.ShowHidden = showHidden
	storeConf.ReservedIgnores = append(storeConf.ReservedIgnores, leapsCOTPath)

	authenticator := acl.NewFileExists(storeConf, logger)

	// Auditors
	auditors := audit.NewToJSON()

	// This flag means the user wants uncommitted changes to be written to disk
	// and then we exit.
	if applyLcot {
		if err := readAudit(leapsCOTPath, auditors, docStore); err != nil && !os.IsNotExist(err) {
			logger.Errorf("Failed to read previously uncommitted changes: %v\n", err)
			os.Exit(1)
		}
		if err := os.Remove(leapsCOTPath); err != nil && !os.IsNotExist(err) {
			logger.Errorf("Changes were successfully committed, but the old audit was not removed: %v\n", err)
			logger.Errorf("You should remove %v manually before running leaps again\n", leapsCOTPath)
			os.Exit(1)
		}
		logger.Infoln("Successfully committed changes to disk.")
		os.Exit(0)
	}

	// This flag means the user wants uncommitted changes to be deleted, and
	// then we exit.
	if discardLcot {
		if err := os.Remove(leapsCOTPath); err != nil && !os.IsNotExist(err) {
			logger.Errorf("Failed to remove %v: %v\n", leapsCOTPath, err)
			os.Exit(1)
		}
		logger.Infoln("Successfully discarded changes.")
		os.Exit(0)
	}

	// This flag means we are not allowed to write changes directly, so instead
	// we write to a Compressed-OT file.
	if safeMode {
		if err := readAudit(leapsCOTPath, auditors, docStore); err != nil && !os.IsNotExist(err) {
			logger.Errorf("Failed to read previously uncommitted changes: %v\n", err)
			os.Exit(1)
		}
		go func() {
			for {
				select {
				case <-time.After(time.Second * 10):
					if err := writeAudit(leapsCOTPath, auditors); err != nil {
						logger.Errorf("Failed to write changes to %v: %v\n", leapsCOTPath, err)
					}
				case <-closeChan:
					return
				}
			}
		}()
		// Use defer to commit final audit before exiting.
		defer func() {
			if err := writeAudit(leapsCOTPath, auditors); err != nil {
				logger.Errorf("Failed to write changes to %v: %v\n", leapsCOTPath, err)
			}
		}()
		logger.Warnf("Changes are being written to %v.\n", leapsCOTPath)
		logger.Warnln("In order to apply these changes you can commit them with `leaps --commit`")
	} else {
		logger.Infoln("Writing changes directly to the filesystem")
	}

	// Curator of documents
	curatorConf := curator.NewConfig()
	curator, err := curator.New(curatorConf, logger, stats, authenticator, docStore, auditors)
	if err != nil {
		fmt.Fprintln(os.Stderr, fmt.Sprintf("Curator error: %v\n", err))
		os.Exit(1)
	}
	defer curator.Close()

	handle("/endpoints", "Lists all available endpoints (including this one).",
		func(w http.ResponseWriter, r *http.Request) {
			data, reqErr := json.Marshal(endpoints)
			if reqErr != nil {
				logger.Errorf("Failed to serve endpoints: %v\n", reqErr)
				http.Error(w, reqErr.Error(), http.StatusInternalServerError)
				return
			}
			w.Header().Add("Content-Type", "application/json")
			w.Write(data)
		})

	handle("/files", "Returns a list of available files and a map of users per document.",
		func(w http.ResponseWriter, r *http.Request) {
			data, err := json.Marshal(struct {
				Paths []string `json:"paths"`
			}{
				Paths: authenticator.GetPaths(),
			})
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				logger.Errorf("Failed to serve users: %v\n", err)
				return
			}
			w.Header().Add("Content-Type", "application/json")
			w.Write(data)
		})

	if hStats, ok := stats.(*metrics.HTTP); ok {
		handle("/stats", "Lists all aggregated metrics as a json blob.", hStats.JSONHandler())
	}

	wwwPath := gopath.Join("/", subdirPath)
	stripPath := ""
	if wwwPath != "/" {
		wwwPath = wwwPath + "/"
		stripPath = wwwPath
	}
	if len(debugWWWDir) > 0 {
		logger.Warnf("Serving web files from alternative www dir: %v\n", debugWWWDir)
		http.Handle(wwwPath, http.StripPrefix(stripPath, http.FileServer(http.Dir(debugWWWDir))))
	} else {
		http.Handle(wwwPath, http.StripPrefix(stripPath, http.FileServer(assetFS())))
	}

	// Leaps API
	globalBroker := api.NewGlobalMetadataBroker(time.Second*300, logger, stats)
	cmdBroker := api.NewCMDBroker(cmds, shellRunner{}, time.Second*300, logger, stats)

	http.HandleFunc(gopath.Join("/", subdirPath, "/leaps/ws"), func(w http.ResponseWriter, r *http.Request) {
		username := r.URL.Query().Get("username")
		uuid := util.GenerateUUID()

		if len(username) == 0 {
			http.Error(w, "Expected username header value in request", http.StatusBadRequest)
			logger.Errorln("Failed to create websocket: no username provided")
			return
		}

		conn, err := (&websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
		}).Upgrade(w, r, nil)

		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			logger.Errorf("Failed to create websocket: %v\n", err)
			return
		}

		jsonEmitter := apiio.NewJSONEmitter(&apiio.ConcurrentJSON{C: conn})
		globalBroker.NewEmitter(username, uuid, jsonEmitter)
		cmdBroker.NewEmitter(username, uuid, jsonEmitter)
		api.NewCuratorSession(username, uuid, jsonEmitter, curator, time.Second*300, logger, stats)

		jsonEmitter.ListenAndEmit()
	})

	logger.Infoln("Launching a leaps instance, use CTRL+C to close.")

	go func() {
		logger.Infof("Serving HTTP requests at: %v%v\n", httpAddress, subdirPath)
		if httperr := http.ListenAndServe(httpAddress, nil); httperr != nil {
			fmt.Fprintln(os.Stderr, fmt.Sprintf("HTTP listen error: %v\n", httperr))
		}
		closeChan <- true
	}()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Wait for termination signal
	select {
	case <-sigChan:
		close(closeChan)
	case <-closeChan:
	}
}

//------------------------------------------------------------------------------
