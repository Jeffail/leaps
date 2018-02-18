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

package metrics

import (
	"bytes"
	"errors"
	"sort"
	"strings"
)

//--------------------------------------------------------------------------------------------------

// Errors for the metrics package.
var (
	ErrInvalidMetricOutputType = errors.New("invalid metrics output type")
)

//--------------------------------------------------------------------------------------------------

// typeSpec - Constructor and a usage description for each metric output type.
type typeSpec struct {
	constructor func(conf Config) (Type, error)
	description string
}

var constructors = map[string]typeSpec{}

//--------------------------------------------------------------------------------------------------

// Config - The all encompassing configuration struct for all metric output types.
type Config struct {
	Type    string        `json:"type" yaml:"type"`
	HTTP    HTTPConfig    `json:"http_server" yaml:"http_server"`
	Riemann RiemannConfig `json:"riemann" yaml:"riemann"`
	Statsd  StatsdConfig  `json:"statsd" yaml:"statsd"`
}

// NewConfig - Returns a configuration struct fully populated with default values.
func NewConfig() Config {
	return Config{
		Type:    "none",
		HTTP:    NewHTTPConfig(),
		Riemann: NewRiemannConfig(),
		Statsd:  NewStatsdConfig(),
	}
}

//--------------------------------------------------------------------------------------------------

// Descriptions - Returns a formatted string of collated descriptions of each type.
func Descriptions() string {
	// Order our input types alphabetically
	names := []string{}
	for name := range constructors {
		names = append(names, name)
	}
	sort.Strings(names)

	buf := bytes.Buffer{}
	buf.WriteString("METRIC TARGETS\n")
	buf.WriteString(strings.Repeat("=", 80))
	buf.WriteString("\n\n")

	// Append each description
	for i, name := range names {
		buf.WriteString(name)
		buf.WriteString("\n")
		buf.WriteString(strings.Repeat("-", 80))
		buf.WriteString("\n")
		buf.WriteString(constructors[name].description)
		buf.WriteString("\n")
		if i != (len(names) - 1) {
			buf.WriteString("\n")
		}
	}
	return buf.String()
}

// New - Create a metric output type based on a configuration.
func New(conf Config) (Type, error) {
	if conf.Type == "none" {
		return DudType{}, nil
	}
	if c, ok := constructors[conf.Type]; ok {
		return c.constructor(conf)
	}
	return nil, ErrInvalidMetricOutputType
}

//--------------------------------------------------------------------------------------------------
