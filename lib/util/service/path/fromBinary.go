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

package path

import (
	"path/filepath"

	"github.com/kardianos/osext"
)

var (
	executablePath string
	resolveError   error
)

func init() {
	// Get the location of the executing binary
	executablePath, resolveError = osext.ExecutableFolder()
}

/*
FromBinaryIfRelative - Takes a path and, if the path is relative, resolves the path from the
location of the binary file rather than the current working directory. Returns an error when the
path is relative and cannot be resolved.
*/
func FromBinaryIfRelative(path *string) error {
	if !filepath.IsAbs(*path) {
		if resolveError != nil {
			return resolveError
		}
		*path = filepath.Join(executablePath, *path)
	}
	return nil
}

/*
BinaryPath - Returns the path of the executing binary, or an error if it couldn't be resolved.
*/
func BinaryPath() (string, error) {
	return executablePath, resolveError
}
