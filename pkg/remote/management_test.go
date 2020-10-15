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

package remote

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"opendev.org/airship/airshipctl/pkg/config"
	"opendev.org/airship/airshipctl/pkg/document"
	"opendev.org/airship/airshipctl/pkg/remote/redfish"
	redfishdell "opendev.org/airship/airshipctl/pkg/remote/redfish/vendors/dell"
	"opendev.org/airship/airshipctl/testutil"
)

type Configuration func(*config.Config)

// initSettings initializes the global airshipctl settings with test data by accepting functions as arguments that
// manipulate configuration sections.
func initSettings(t *testing.T, configs ...Configuration) *config.Config {
	t.Helper()

	config := testutil.DummyConfig()
	for _, cfg := range configs {
		cfg(config)
	}
	return config
}

// withManagementConfig initializes the management config when used as an argument to initSettings.
func withManagementConfig(cfg *config.ManagementConfiguration) Configuration {
	return func(settings *config.Config) {
		settings.ManagementConfiguration["dummy_management_config"] = cfg
	}
}

// withTestDataPath sets the test data path when used as an argument to initSettings.
func withTestDataPath(path string) Configuration {
	return func(settings *config.Config) {
		manifest, err := settings.CurrentContextManifest()
		if err != nil {
			panic(fmt.Sprintf("Unable to initialize management tests. Current Context error %q", err))
		}
		manifest.TargetPath = "testdata"
		manifest.MetadataPath = "metadata.yaml"
		dummyRepo := testutil.DummyRepository()
		dummyRepo.URLString = fmt.Sprintf("http://dummy.url.com/%s.git", path)
		manifest.Repositories = map[string]*config.Repository{manifest.PhaseRepositoryName: dummyRepo}
	}
}

func TestNewManagerEphemeralHost(t *testing.T) {
	settings := initSettings(t, withTestDataPath("base"))

	manager, err := NewManager(settings, config.BootstrapPhase, ByLabel(document.EphemeralHostSelector))
	require.NoError(t, err)
	require.Equal(t, 1, len(manager.Hosts))

	assert.Equal(t, "ephemeral", manager.Hosts[0].NodeID())
}

func TestNewManagerByName(t *testing.T) {
	settings := initSettings(t, withTestDataPath("base"))

	manager, err := NewManager(settings, config.BootstrapPhase, ByName("master-1"))
	require.NoError(t, err)
	require.Equal(t, 1, len(manager.Hosts))

	assert.Equal(t, "node-master-1", manager.Hosts[0].NodeID())
}

func TestNewManagerMultipleNodes(t *testing.T) {
	settings := initSettings(t, withTestDataPath("base"))

	manager, err := NewManager(settings, config.BootstrapPhase, ByLabel("airshipit.org/test-node=true"))
	require.NoError(t, err)
	require.Equal(t, 2, len(manager.Hosts))

	assert.Equal(t, "node-master-1", manager.Hosts[0].NodeID())
	assert.Equal(t, "node-master-2", manager.Hosts[1].NodeID())
}

func TestNewManagerMultipleSelectors(t *testing.T) {
	settings := initSettings(t, withTestDataPath("base"))

	manager, err := NewManager(settings, config.BootstrapPhase, ByName("master-1"),
		ByLabel("airshipit.org/test-node=true"))
	require.NoError(t, err)
	require.Equal(t, 1, len(manager.Hosts))

	assert.Equal(t, "node-master-1", manager.Hosts[0].NodeID())
}

func TestNewManagerMultipleSelectorsNoMatch(t *testing.T) {
	settings := initSettings(t, withTestDataPath("base"))

	manager, err := NewManager(settings, config.BootstrapPhase, ByName("master-2"),
		ByLabel(document.EphemeralHostSelector))

	// Must return ErrNoHostsFound here, without check for specific error, test can panic
	require.Equal(t, ErrNoHostsFound{}, err)
	require.Equal(t, 0, len(manager.Hosts))
	assert.Error(t, err)
}

func TestNewManagerByNameNoHostFound(t *testing.T) {
	settings := initSettings(t, withTestDataPath("base"))

	_, err := NewManager(settings, config.BootstrapPhase, ByName("bad-name"))
	assert.Error(t, err)
}

func TestNewManagerNoSelectors(t *testing.T) {
	settings := initSettings(t, withTestDataPath("base"))

	_, err := NewManager(settings, config.BootstrapPhase)
	assert.Error(t, err)
}

func TestNewManagerByLabelNoHostsFound(t *testing.T) {
	settings := initSettings(t, withTestDataPath("base"))

	_, err := NewManager(settings, config.BootstrapPhase, ByLabel("bad-label=true"))
	assert.Error(t, err)
}

func TestNewManagerRedfish(t *testing.T) {
	cfg := &config.ManagementConfiguration{Type: redfish.ClientType}
	settings := initSettings(t, withManagementConfig(cfg), withTestDataPath("base"))

	_, err := NewManager(settings, config.BootstrapPhase, ByLabel(document.EphemeralHostSelector))
	assert.NoError(t, err)
}

func TestNewManagerRedfishDell(t *testing.T) {
	cfg := &config.ManagementConfiguration{Type: redfishdell.ClientType}
	settings := initSettings(t, withManagementConfig(cfg), withTestDataPath("base"))

	_, err := NewManager(settings, config.BootstrapPhase, ByLabel(document.EphemeralHostSelector))
	assert.NoError(t, err)
}

func TestNewManagerUnknownRemoteType(t *testing.T) {
	badCfg := &config.ManagementConfiguration{Type: "bad-remote-type"}
	settings := initSettings(t, withManagementConfig(badCfg), withTestDataPath("base"))

	_, err := NewManager(settings, config.BootstrapPhase, ByLabel(document.EphemeralHostSelector))
	assert.Error(t, err)
}

func TestNewManagerMissingBMCAddress(t *testing.T) {
	settings := initSettings(t, withTestDataPath("emptyurl"))

	_, err := NewManager(settings, config.BootstrapPhase, ByLabel(document.EphemeralHostSelector))
	assert.Error(t, err)
}

func TestNewManagerMissingCredentials(t *testing.T) {
	settings := initSettings(t, withTestDataPath("emptyurl"))

	_, err := NewManager(settings, config.BootstrapPhase, ByName("no-creds"))
	assert.Error(t, err)
}

func TestNewManagerBadPhase(t *testing.T) {
	settings := initSettings(t, withTestDataPath("base"))

	_, err := NewManager(settings, "bad-phase", ByLabel(document.EphemeralHostSelector))
	assert.Error(t, err)
}
