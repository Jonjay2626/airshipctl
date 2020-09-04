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

package render

import (
	"io"
	"strings"

	"opendev.org/airship/airshipctl/pkg/config"
	"opendev.org/airship/airshipctl/pkg/document"
)

// Render prints out filtered documents
func (s *Settings) Render(cfgFactory config.Factory, phaseName string, out io.Writer) error {
	cfg, err := cfgFactory()
	if err != nil {
		return err
	}

	path, err := cfg.CurrentContextEntryPoint(phaseName)
	if err != nil {
		return err
	}

	docBundle, err := document.NewBundleByPath(path)
	if err != nil {
		return err
	}

	groupVersion := strings.Split(s.APIVersion, "/")
	group := ""
	version := groupVersion[0]
	if len(groupVersion) > 1 {
		group = groupVersion[0]
		version = strings.Join(groupVersion[1:], "/")
	}

	sel := document.NewSelector().ByLabel(s.Label).ByAnnotation(s.Annotation).ByGvk(group, version, s.Kind)
	filteredBundle, err := docBundle.SelectBundle(sel)
	if err != nil {
		return err
	}

	return filteredBundle.Write(out)
}
