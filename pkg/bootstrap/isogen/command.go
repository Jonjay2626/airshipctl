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

package isogen

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"opendev.org/airship/airshipctl/pkg/api/v1alpha1"
	"opendev.org/airship/airshipctl/pkg/bootstrap/cloudinit"
	"opendev.org/airship/airshipctl/pkg/config"
	"opendev.org/airship/airshipctl/pkg/container"
	"opendev.org/airship/airshipctl/pkg/document"
	"opendev.org/airship/airshipctl/pkg/log"
	"opendev.org/airship/airshipctl/pkg/util"
)

const (
	builderConfigFileName = "builder-conf.yaml"
)

// GenerateBootstrapIso will generate data for cloud init and start ISO builder container
// TODO (vkuzmin): Remove this public function and move another functions
// to the executor module when the phases will be ready
func GenerateBootstrapIso(cfgFactory config.Factory) error {
	ctx := context.Background()

	globalConf, err := cfgFactory()
	if err != nil {
		return err
	}

	root, err := globalConf.CurrentContextEntryPoint(config.BootstrapPhase)
	if err != nil {
		return err
	}
	docBundle, err := document.NewBundleByPath(root)
	if err != nil {
		return err
	}

	imageConfiguration := &v1alpha1.ImageConfiguration{}
	selector, err := document.NewSelector().ByObject(imageConfiguration, v1alpha1.Scheme)
	if err != nil {
		return err
	}
	doc, err := docBundle.SelectOne(selector)
	if err != nil {
		return err
	}

	err = doc.ToAPIObject(imageConfiguration, v1alpha1.Scheme)
	if err != nil {
		return err
	}
	if err = verifyInputs(imageConfiguration); err != nil {
		return err
	}

	log.Print("Creating ISO builder container")
	builder, err := container.NewContainer(
		&ctx, imageConfiguration.Container.ContainerRuntime,
		imageConfiguration.Container.Image)
	if err != nil {
		return err
	}

	err = createBootstrapIso(docBundle, builder, doc, imageConfiguration, log.DebugEnabled())
	if err != nil {
		return err
	}
	log.Print("Checking artifacts")
	return verifyArtifacts(imageConfiguration)
}

func verifyInputs(cfg *v1alpha1.ImageConfiguration) error {
	if cfg.Container.Volume == "" {
		return config.ErrMissingConfig{
			What: "Must specify volume bind for ISO builder container",
		}
	}

	if (cfg.Builder.UserDataFileName == "") || (cfg.Builder.NetworkConfigFileName == "") {
		return config.ErrMissingConfig{
			What: "UserDataFileName or NetworkConfigFileName are not specified in ISO builder config",
		}
	}

	vols := strings.Split(cfg.Container.Volume, ":")
	switch {
	case len(vols) == 1:
		cfg.Container.Volume = fmt.Sprintf("%s:%s", vols[0], vols[0])
	case len(vols) > 2:
		return config.ErrInvalidConfig{
			What: "Bad container volume format. Use hostPath:contPath",
		}
	}
	return nil
}

func getContainerCfg(
	cfg *v1alpha1.ImageConfiguration,
	builderCfgYaml []byte,
	userData []byte,
	netConf []byte,
) map[string][]byte {
	hostVol := strings.Split(cfg.Container.Volume, ":")[0]

	fls := make(map[string][]byte)
	fls[filepath.Join(hostVol, cfg.Builder.UserDataFileName)] = userData
	fls[filepath.Join(hostVol, cfg.Builder.NetworkConfigFileName)] = netConf
	fls[filepath.Join(hostVol, builderConfigFileName)] = builderCfgYaml
	return fls
}

func verifyArtifacts(cfg *v1alpha1.ImageConfiguration) error {
	hostVol := strings.Split(cfg.Container.Volume, ":")[0]
	metadataPath := filepath.Join(hostVol, cfg.Builder.OutputMetadataFileName)
	_, err := os.Stat(metadataPath)
	return err
}

func createBootstrapIso(
	docBundle document.Bundle,
	builder container.Container,
	doc document.Document,
	cfg *v1alpha1.ImageConfiguration,
	debug bool,
) error {
	cntVol := strings.Split(cfg.Container.Volume, ":")[1]
	log.Print("Creating cloud-init for ephemeral K8s")
	userData, netConf, err := cloudinit.GetCloudData(docBundle)
	if err != nil {
		return err
	}

	builderCfgYaml, err := doc.AsYAML()
	if err != nil {
		return err
	}

	fls := getContainerCfg(cfg, builderCfgYaml, userData, netConf)
	if err = util.WriteFiles(fls, 0600); err != nil {
		return err
	}

	vols := []string{cfg.Container.Volume}
	builderCfgLocation := filepath.Join(cntVol, builderConfigFileName)
	log.Printf("Running default container command. Mounted dir: %s", vols)
	if err := builder.RunCommand(
		[]string{},
		nil,
		vols,
		[]string{
			fmt.Sprintf("BUILDER_CONFIG=%s", builderCfgLocation),
			fmt.Sprintf("http_proxy=%s", os.Getenv("http_proxy")),
			fmt.Sprintf("https_proxy=%s", os.Getenv("https_proxy")),
			fmt.Sprintf("HTTP_PROXY=%s", os.Getenv("HTTP_PROXY")),
			fmt.Sprintf("HTTPS_PROXY=%s", os.Getenv("HTTPS_PROXY")),
			fmt.Sprintf("NO_PROXY=%s", os.Getenv("NO_PROXY")),
		},
		debug,
	); err != nil {
		return err
	}

	log.Print("ISO successfully built.")
	if !debug {
		log.Print("Removing container.")
		return builder.RmContainer()
	}

	log.Debugf("Debug flag is set. Container %s stopped but not deleted.", builder.GetID())
	return nil
}
