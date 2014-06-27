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

package leaplib

import (
	"errors"
	"github.com/jeffail/gabs"
	"log"
	"os"
	"time"
)

/*--------------------------------------------------------------------------------------------------
 */

/* Constants that define various log levels
 */
const (
	LeapError = 0
	LeapWarn  = 1
	LeapInfo  = 2
	LeapDebug = 3
)

/*
LoggerConfig - Holds configuration options for the global leaps logger.
*/
type LoggerConfig struct {
	LogLevel   int     `json:"level"`
	TargetPath *string `json:"output_path,omitempty"`
}

/*
DefaultLoggerConfig - Returns a fully defined logger configuration with the default values for
each field.
*/
func DefaultLoggerConfig() LoggerConfig {
	return LoggerConfig{
		LogLevel:   LeapInfo,
		TargetPath: nil,
	}
}

/*--------------------------------------------------------------------------------------------------
 */

/*
LeapsLogger - A logger object configured along with leaps, which all leaps components log to.
*/
type LeapsLogger struct {
	config      LoggerConfig
	logger      *log.Logger
	stats       *gabs.Container
	jobChan     chan func()
	requestChan chan chan<- string
}

/*
CreateLogger - Create a new logger.
*/
func CreateLogger(config LoggerConfig) *LeapsLogger {
	stats, _ := gabs.Consume(map[string]interface{}{})
	stats.SetP(time.Now().Unix(), "curator.started_ts")

	logger := LeapsLogger{
		config:      config,
		logger:      log.New(os.Stdout, "[Leaps] ", log.LstdFlags),
		stats:       stats,
		jobChan:     make(chan func(), 100),
		requestChan: make(chan chan<- string, 5),
	}

	go logger.loop()

	return &logger
}

/*
Close - Close the logger, a closed logger will no longer respond to requests for stats or log events
and probably isn't even worth doing. I don't know why I wrote this. Help me.
*/
func (l *LeapsLogger) Close() {
	jChan, rChan := l.jobChan, l.requestChan
	l.jobChan = nil
	l.requestChan = nil
	close(jChan)
	close(rChan)
}

/*--------------------------------------------------------------------------------------------------
 */

/*
Log - Log an event.
*/
func (l *LeapsLogger) Log(level int, prefix, message string) {
	if level <= l.config.LogLevel {
		var logLevel string
		switch level {
		case LeapError:
			logLevel = "error"
		case LeapWarn:
			logLevel = "warn"
		case LeapInfo:
			logLevel = "info"
		case LeapDebug:
			logLevel = "debug"
		}
		l.logger.Printf("[%v] [%v] | %v\n", logLevel, prefix, message)
	}
}

/*--------------------------------------------------------------------------------------------------
 */

/*
GetStats - Returns a string containing the JSON serialized structure of logger stats at the time of
the request.
*/
func (l *LeapsLogger) GetStats(timeout time.Duration) (*string, error) {
	responseChan := make(chan string, 1)
	select {
	case l.requestChan <- responseChan:
	default:
		return nil, errors.New("request was blocked")
	}

	select {
	case stats := <-responseChan:
		return &stats, nil
	case <-time.After(timeout):
	}
	return nil, errors.New("request timed out")
}

/*
IncrementStat - Increment the integer value of a particular stat. If the stat doesn't yet exist it
is created. Stats that are incremented are integer only, and this is enforced.
*/
func (l *LeapsLogger) IncrementStat(path string) {
	l.jobChan <- func() {
		root := l.stats.S("curator")
		if target, ok := root.Path(path).Data().(int); ok {
			root.SetP(target+1, path)
		} else {
			root.SetP(1, path)
		}
	}
}

/*
DecrementStat - Decrement the integer value of a particular stat. If the stat doesn't yet exist it
is created. Stats that are decremented are integer only, and this is enforced.
*/
func (l *LeapsLogger) DecrementStat(path string) {
	l.jobChan <- func() {
		root := l.stats.S("curator")
		if target, ok := root.Path(path).Data().(int); ok {
			root.SetP(target-1, path)
		} else {
			root.SetP(-1, path)
		}
	}
}

/*
SetStat - Sets a stat to whatever value you give it, it's currently up to the caller to ensure the
value will be JSON serializable. Calls are made asynchronously and do not return errors or
indications of success.
*/
func (l *LeapsLogger) SetStat(path string, value interface{}) {
	l.jobChan <- func() {
		l.stats.S("curator").SetP(value, path)
	}
}

/*--------------------------------------------------------------------------------------------------
 */

/*
loop - Internal loop of the logger, accepts requests for stats updating.
*/
func (l *LeapsLogger) loop() {
	running := true
	for running {
		select {
		case post, open := <-l.jobChan:
			if !open {
				running = false
				break
			}
			post()
		case req, open := <-l.requestChan:
			if !open {
				running = false
				break
			}
			select {
			case req <- l.stats.String():
			default:
			}
		}
	}
}

/*--------------------------------------------------------------------------------------------------
 */
