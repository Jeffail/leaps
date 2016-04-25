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

package acl

import (
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/jeffail/util/log"
)

//--------------------------------------------------------------------------------------------------

// FileExistsConfig - A config object for the FileExists acl object.
type FileExistsConfig struct {
	Path          string `json:"path" yaml:"path"`
	ShowHidden    bool   `json:"show_hidden" yaml:"show_hidden"`
	RefreshPeriod int64  `json:"refresh_period_s" yaml:"refresh_period_s"`
}

// NewFileExistsConfig - Returns a default config object for a FileExists object.
func NewFileExistsConfig() FileExistsConfig {
	return FileExistsConfig{
		Path:          "",
		ShowHidden:    false,
		RefreshPeriod: 10,
	}
}

/*
FileExists - An acl authenticator type that validates document edit sessions by checking that the
document ID (the file path) exists. Can be configured to show hidden files.
*/
type FileExists struct {
	logger log.Modular
	config FileExistsConfig
	paths  []string
	mutex  *sync.RWMutex
}

// NewFileExists - Creates an File using the provided configuration.
func NewFileExists(config FileExistsConfig, logger log.Modular) *FileExists {
	fa := FileExists{
		logger: logger.NewModule(":fs_auth"),
		config: config,
		paths:  []string{},
		mutex:  &sync.RWMutex{},
	}
	go fa.loop()
	return &fa
}

//--------------------------------------------------------------------------------------------------

func (f *FileExists) getPaths() ([]string, error) {
	paths := []string{}
	if info, err := os.Stat(f.config.Path); err == nil {
		// If the path is a file then it is the only valid target.
		if info.Mode().IsRegular() {
			return []string{path.Clean(f.config.Path)}, nil
		}
	} else {
		return paths, err
	}
	if err := filepath.Walk(f.config.Path, func(p string, info os.FileInfo, err error) error {
		if !f.config.ShowHidden {
			// If not showing hidden files then skip when prefix is "."
			if len(info.Name()) > 1 && strings.HasPrefix(info.Name(), ".") {
				if info.Mode().IsRegular() {
					return nil
				}
				// Skip hidden directories
				return filepath.SkipDir
			}
		}
		if info.Mode().IsRegular() {
			if relPath, err := filepath.Rel(f.config.Path, p); err == nil {
				paths = append(paths, relPath)
			} else {
				f.logger.Errorf("Relative path conversion error: %v\n", err)
			}
		}
		return nil
	}); err != nil {
		return []string{}, err
	}
	return paths, nil
}

func (f *FileExists) loop() {
	for {
		f.mutex.Lock()
		p, err := f.getPaths()
		if err != nil {
			f.logger.Errorf("Failed to walk paths for authenticator: %v\n", err)
		}
		f.mutex.Unlock()

		f.paths = p
		time.Sleep(time.Duration(f.config.RefreshPeriod) * time.Second)
	}
}

//--------------------------------------------------------------------------------------------------

// Authenticate - Checks whether the documentID (file path) exists, returns EditAccess if it does.
func (f *FileExists) Authenticate(_, _, documentID string) AccessLevel {
	f.mutex.RLock()
	defer f.mutex.RUnlock()

	cleanPath := path.Clean(documentID)
	for _, p := range f.paths {
		if cleanPath == p {
			return EditAccess
		}
	}
	return NoAccess
}

// GetPaths - Returns the cached list of file paths available.
func (f *FileExists) GetPaths() []string {
	return f.paths
}

//--------------------------------------------------------------------------------------------------
