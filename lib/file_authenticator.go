package lib

import (
	"encoding/json"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/jeffail/util/log"
)

/*--------------------------------------------------------------------------------------------------
 */

/*
FileAuthenticatorConfig - A config object for the file system authentication object.
*/
type FileAuthenticatorConfig struct {
	SharePath     string `json:"share_directory" yaml:"share_path"`
	Path          string `json:"path" yaml:"path"`
	ShowHidden    bool   `json:"show_hidden" yaml:"show_hidden"`
	RefreshPeriod int64  `json:"refresh_period_s" yaml:"refresh_period_s"`
}

/*
DefaultFileAuthenticatorConfig - Returns a default config object for a FileAuthenticator.
*/
func DefaultFileAuthenticatorConfig() FileAuthenticatorConfig {
	return FileAuthenticatorConfig{
		SharePath:     "",
		Path:          "",
		ShowHidden:    false,
		RefreshPeriod: 10,
	}
}

/*--------------------------------------------------------------------------------------------------
 */

func (f *FileAuthenticator) getPaths() ([]string, error) {
	paths := []string{}
	if len(f.config.FileConfig.SharePath) == 0 {
		return paths, nil
	}
	if info, err := os.Stat(f.config.FileConfig.SharePath); err == nil {
		if info.Mode().IsRegular() {
			return []string{path.Clean(f.config.FileConfig.SharePath)}, nil
		}
	} else {
		return paths, err
	}
	if err := filepath.Walk(f.config.FileConfig.SharePath, func(p string, info os.FileInfo, err error) error {
		if !f.config.FileConfig.ShowHidden {
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
			if relPath, err := filepath.Rel(f.config.FileConfig.SharePath, p); err == nil {
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

func (f *FileAuthenticator) servePaths(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Supports GET verb only", http.StatusMethodNotAllowed)
		return
	}
	f.mutex.RLock()
	js, err := json.Marshal(struct {
		Paths []string `json:"paths"`
	}{
		Paths: f.paths,
	})
	f.mutex.RUnlock()
	if err != nil {
		f.logger.Errorf("Failed to marshal paths for server: %v\n", err)
		http.Error(w, "Internal server issue", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(js)
}

/*--------------------------------------------------------------------------------------------------
 */

/*
FileAuthenticator - A utility for using the filesystem as a way of validating that a document exists
and is available to edit. This is intended to be used in tandem with the file based document store.

The FileAuthenticator takes a directory as a config option. When a client wishes to connect to a
file the user token and document ID are given, where the document ID is the relative path to the
target file.

The FileAuthenticator will then verify that this file exists within the configured directory. This
is an attempt to isolate leaps, and avoid users connecting to paths such as ../../../etc/passwd.
*/
type FileAuthenticator struct {
	logger *log.Logger
	config TokenAuthenticatorConfig
	paths  []string
	mutex  *sync.RWMutex
}

/*
NewFileAuthenticator - Creates an FileAuthenticator using the provided configuration.
*/
func NewFileAuthenticator(config TokenAuthenticatorConfig, logger *log.Logger) *FileAuthenticator {
	fa := FileAuthenticator{
		logger: logger.NewModule(":fs_auth"),
		config: config,
		paths:  []string{},
		mutex:  &sync.RWMutex{},
	}
	go fa.loop()
	return &fa
}

func (f *FileAuthenticator) loop() {
	var err error
	for {
		f.mutex.Lock()
		f.paths, err = f.getPaths()
		if err != nil {
			f.logger.Errorf("Failed to walk paths for authenticator: %v\n", err)
		}
		f.mutex.Unlock()

		time.Sleep(time.Duration(f.config.FileConfig.RefreshPeriod) * time.Second)
	}
}

/*--------------------------------------------------------------------------------------------------
 */

/*
AuthoriseCreate - Always returns false.
*/
func (f *FileAuthenticator) AuthoriseCreate(token, userID string) bool {
	return false
}

/*
AuthoriseJoin - Checks whether the documentID file exists, returns true if it does, otherwise false.
*/
func (f *FileAuthenticator) AuthoriseJoin(token, documentID string) bool {
	f.mutex.RLock()
	defer f.mutex.RUnlock()

	cleanPath := path.Clean(documentID)
	for _, p := range f.paths {
		if cleanPath == p {
			return true
		}
	}
	return false
}

/*
RegisterHandlers - Register an endpoint for obtaining a list of available files.
*/
func (f *FileAuthenticator) RegisterHandlers(register PubPrivEndpointRegister) error {
	if len(f.config.FileConfig.Path) > 0 {
		return register.RegisterPublic(
			f.config.FileConfig.Path,
			"Get a list of files available for editing",
			f.servePaths,
		)
	}
	return nil
}

/*--------------------------------------------------------------------------------------------------
 */
