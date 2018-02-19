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
	"bufio"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/Jeffail/leaps/lib/util/service/log"
)

//--------------------------------------------------------------------------------------------------

// FileExistsConfig - A config object for the FileExists acl object.
type FileExistsConfig struct {
	Path            string   `json:"path" yaml:"path"`
	ShowHidden      bool     `json:"show_hidden" yaml:"show_hidden"`
	RefreshPeriod   int64    `json:"refresh_period_s" yaml:"refresh_period_s"`
	ReservedIgnores []string `json:"ignore_files" yaml:"ignore_files"`
}

// NewFileExistsConfig - Returns a default config object for a FileExists object.
func NewFileExistsConfig() FileExistsConfig {
	return FileExistsConfig{
		Path:            "",
		ShowHidden:      false,
		RefreshPeriod:   10,
		ReservedIgnores: []string{".leapsignore"},
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

// extractIgnores - Parses a .leapsignore file for ignore patterns (one per line).
func (f *FileExists) extractIgnores(ignoreFilePath string) []string {
	ignorePatterns := []string{}
	for _, p := range f.config.ReservedIgnores {
		ignorePatterns = append(ignorePatterns, p)
	}

	file, err := os.Open(ignoreFilePath)
	if err != nil {
		f.logger.Errorf("Failed to read .leapsignore file: %v\n", err)
		return ignorePatterns
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		ignorePatterns = append(ignorePatterns, filepath.Clean(scanner.Text()))
	}
	if err := scanner.Err(); err != nil {
		f.logger.Errorf("Failed to read .leapsignore file: %v\n", err)
	}
	return ignorePatterns
}

/*
checkPatterns - Checks an array of ignore patterns against a path.

Pattern rules:

Pattern match rules either:
	- Pattern is an exact match of the full relative path (foo/bar/*.jpg matches foo/bar/test.jpg)
or:
	- Pattern is an exact match of the base of the path (*.jpg matches foo/bar/test.jpg)

./foo.jpg will only match foo.jpg
foo.jpg will match foo.jpg and bar/foo.jpg
*/
func (f *FileExists) checkPatterns(patterns []string, path string) bool {
	for _, pattern := range patterns {
		// Cleaned pattern and full relative path
		matched, err := filepath.Match(filepath.Clean(pattern), path)
		if err != nil {
			f.logger.Errorf("Pattern match error: %s: %v\n", pattern, err)
			return false
		}
		// Or non-cleaned pattern versus base of relative path
		if !matched {
			matched, err = filepath.Match(pattern, filepath.Base(path))
			if err != nil {
				f.logger.Errorf("Pattern match error: %s: %v\n", pattern, err)
				return false
			}
		}
		if matched {
			return true
		}
	}
	return false
}

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
	// map[directoryTree][]ignorePaths
	ignorePatterns := map[string][]string{}
	if err := filepath.Walk(f.config.Path, func(p string, info os.FileInfo, err error) error {
		// Get path relative to root search directory.
		relPath, err := filepath.Rel(f.config.Path, p)
		if err != nil {
			f.logger.Errorf("Relative path conversion error: %v\n", err)
			// Stop walking files
			return err
		}
		// If we've found a .leapsignore file then parse it for ignore patterns.
		if info.Mode().IsRegular() && info.Name() == ".leapsignore" {
			dirTree := filepath.Dir(relPath)
			if dirTree == "." {
				dirTree = ""
			}
			ignorePatterns[dirTree] = append(ignorePatterns[dirTree], f.extractIgnores(relPath)...)
			return nil
		}
		for dirTree, patterns := range ignorePatterns {
			if strings.Contains(relPath, dirTree) {
				relToPatternDir, err := filepath.Rel(dirTree, relPath)
				if err != nil {
					f.logger.Errorf("Relative path conversion error: %v\n", err)
					// Stop walking files
					return err
				}
				if f.checkPatterns(patterns, relToPatternDir) {
					// Otherwise, check all ignore patterns for a match.
					if info.Mode().IsRegular() {
						return nil
					}
					return filepath.SkipDir
				}
			}
		}
		// If not showing hidden files then skip when prefix is "."
		if !f.config.ShowHidden && len(info.Name()) > 1 && strings.HasPrefix(info.Name(), ".") {
			if info.Mode().IsRegular() {
				return nil
			}
			// Skip hidden directories
			return filepath.SkipDir
		}
		if info.Mode().IsRegular() {
			paths = append(paths, relPath)
		}
		return nil
	}); err != nil {
		return []string{}, err
	}
	return paths, nil
}

func (f *FileExists) loop() {
	for {
		p, err := f.getPaths()
		if err != nil {
			f.logger.Errorf("Failed to walk paths for authenticator: %v\n", err)
		}

		f.mutex.Lock()
		f.paths = p
		f.mutex.Unlock()

		time.Sleep(time.Duration(f.config.RefreshPeriod) * time.Second)
	}
}

//--------------------------------------------------------------------------------------------------

// Authenticate - Checks whether the documentID (file path) exists, returns EditAccess if it does.
func (f *FileExists) Authenticate(_ interface{}, _, documentID string) AccessLevel {
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
