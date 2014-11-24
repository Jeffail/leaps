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

package util

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v2"
)

/*--------------------------------------------------------------------------------------------------
 */

var (
	version        string
	dateBuilt      string
	showVersion    *bool
	showConfigJSON *bool
	showConfigYAML *bool
	configPath     *string
)

func init() {
	showVersion = flag.Bool("version", false, "Display version info, then exit")
	showConfigJSON = flag.Bool("print-json", false, "Print loaded configuration as JSON, then exit")
	showConfigYAML = flag.Bool("print-yaml", false, "Print loaded configuration as YAML, then exit")
	configPath = flag.String("c", "", "Path to a configuration file")
}

/*--------------------------------------------------------------------------------------------------
 */

/*
Bootstrap - bootstraps the configuration loading, parsing and reporting for a service through cmd
flags. The argument configPtr should be a pointer to a serializable configuration object with all
default values.

Bootstrap allows a user to do the following:
- Print version and build info and exit
- Load an optional configuration file (supports JSON, YAML)
- Print the config file (supports JSON, YAML) and exit

NOTE: The user may request a version and build time stamp, in which case Bootstrap will print the
values of util.version and util.dateBuilt. To populate those values you must run go build with the
following:

-ldflags "-X github.com/jeffail/leaps/util.version $(VERSION) \
	-X github.com/jeffail/leaps/util.dateBuilt $(DATE)"

Returns a flag indicating whether the service should continue or not.
*/
func Bootstrap(configPtr interface{}) bool {
	// Ensure that cmd flags are parsed.
	if !flag.Parsed() {
		flag.Parse()
	}

	// If the user wants the version we print it.
	if *showVersion {
		fmt.Printf("Version: %v\nDate: %v\n", version, dateBuilt)
		return false
	}

	if len(*configPath) > 0 {
		// Read config file.
		configBytes, err := ioutil.ReadFile(*configPath)
		if err != nil {
			fmt.Fprintln(os.Stderr, fmt.Sprintf("Error reading config file: %v", err))
			return false
		}

		ext := filepath.Ext(*configPath)
		if ext == ".js" || ext == ".json" {
			if err = json.Unmarshal(configBytes, configPtr); err != nil {
				fmt.Fprintln(os.Stderr, fmt.Sprintf("Error parsing config file: %v", err))
				return false
			}
		} else if ext == ".yaml" {
			if err = yaml.Unmarshal(configBytes, configPtr); err != nil {
				fmt.Fprintln(os.Stderr, fmt.Sprintf("Error parsing config file: %v", err))
				return false
			}
		} else {
			fmt.Fprintln(os.Stderr, "Configuration file extension not recognised")
			return false
		}
	}

	// If the user wants the configuration to be printed we do so and then exit.
	if *showConfigJSON {
		if configJSON, err := json.MarshalIndent(configPtr, "", "\t"); err == nil {
			fmt.Println(string(configJSON))
		} else {
			fmt.Fprintln(os.Stderr, fmt.Sprintf("Configuration marshal error: %v", err))
		}
		return false
	} else if *showConfigYAML {
		if configYAML, err := yaml.Marshal(configPtr); err == nil {
			fmt.Println(string(configYAML))
		} else {
			fmt.Fprintln(os.Stderr, fmt.Sprintf("Configuration marshal error: %v", err))
		}
		return false
	}

	// Set a rand seed.
	rand.Seed(time.Now().Unix())

	return true
}

/*--------------------------------------------------------------------------------------------------
 */
