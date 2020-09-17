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

package types

import (
	"io"
)

// Plugin interface for airship document plugins
type Plugin interface {
	Run(io.Reader, io.Writer) error
}

// Factory function for plugins. Functions of such type are used in the plugin
// registry to instantiate a plugin object
type Factory func(map[string]interface{}) (Plugin, error)
