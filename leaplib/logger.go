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
	"log"
	"os"
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
	config LoggerConfig
	logger *log.Logger
}

/*
CreateLogger - Create a new logger.
*/
func CreateLogger(config LoggerConfig) *LeapsLogger {
	return &LeapsLogger{
		config: config,
		logger: log.New(os.Stdout, "[Leaps] ", log.LstdFlags),
	}
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
