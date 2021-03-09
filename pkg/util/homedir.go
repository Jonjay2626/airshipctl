/*
 Licensed under the Apache License, Version 2.0 (the "License");
 you may not use this file except in compliance with the License.
 You may obtain a copy of the License at

     https://www.apache.org/licenses/LICENSE-2.0

 Unless required by applicable law or agreed to in writing, software
 distributed under the License is distributed on an "AS IS" BASIS,
 WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 See the License for the specific language governing permissions and
 limitations under the License.
*/

package util

import (
	"os"
	"path/filepath"
)

// UserHomeDir is a utility function that wraps os.UserHomeDir and returns no
// errors. If the user has no home directory, the returned value will be the
// empty string
func UserHomeDir() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		homeDir = ""
	}
	return homeDir
}

// ExpandTilde expands the path to include the home directory if the path
// is prefixed with `~`. If it isn't prefixed with `~`, the path is
// returned as-is.
// Original source code: https://github.com/mitchellh/go-homedir/blob/master/homedir.go#L55-L77
func ExpandTilde(path string) (string, error) {
	if len(path) == 0 {
		return path, nil
	}

	if path[0] != '~' {
		return path, nil
	}

	if len(path) > 1 && path[1] != '/' {
		return "", &ErrCantExpandTildePath{}
	}

	return filepath.Join(UserHomeDir(), path[1:]), nil
}
