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
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	"opendev.org/airship/airshipctl/pkg/remote/power"
	"opendev.org/airship/airshipctl/pkg/remote/redfish"
	"opendev.org/airship/airshipctl/testutil/redfishutils"
)

const (
	systemID   = "System.Embedded.1"
	isoURL     = "https://localhost:8080/ubuntu.iso"
	redfishURL = "redfish+https://localhost:2344/Systems/System.Embedded.1"
	username   = "admin"
	password   = "password"
)

func TestDoRemoteDirectMissingConfigOpts(t *testing.T) {
	rMock, err := redfishutils.NewClient(redfishURL, false, false, username, password)
	assert.NoError(t, err)

	ephemeralHost := baremetalHost{
		rMock,
		redfishURL,
		"doc-name",
		username,
		password,
	}

	settings := initSettings(t, withTestDataPath("noremote"))
	// there must be document.ErrDocNotFound
	err = ephemeralHost.DoRemoteDirect(settings)
	expectedErrorMessage := `document filtered by selector [Group="airshipit.org", Version="v1alpha1", ` +
		`Kind="RemoteDirectConfiguration"] found no documents`
	assert.Equal(t, expectedErrorMessage, fmt.Sprintf("%s", err))
	assert.Error(t, err)
}

func TestDoRemoteDirectMissingISOURL(t *testing.T) {
	rMock, err := redfishutils.NewClient(redfishURL, false, false, username, password)
	assert.NoError(t, err)

	ctx := context.Background()

	rMock.On("NodeID").Times(1).Return(systemID)
	rMock.On("SystemPowerStatus", ctx).Times(1).Return(power.StatusOn, nil)

	ephemeralHost := baremetalHost{
		rMock,
		redfishURL,
		"doc-name",
		username,
		password,
	}

	settings := initSettings(t, withTestDataPath("noisourl"))
	err = ephemeralHost.DoRemoteDirect(settings)
	expectedErrorMessage := `missing option: isoURL`
	assert.Equal(t, expectedErrorMessage, fmt.Sprintf("%s", err))
	assert.Error(t, err)
}

func TestDoRemoteDirectRedfish(t *testing.T) {
	rMock, err := redfishutils.NewClient(redfishURL, false, false, username, password)
	assert.NoError(t, err)

	ctx := context.Background()

	rMock.On("NodeID").Times(1).Return(systemID)
	rMock.On("SystemPowerStatus", ctx).Times(1).Return(power.StatusOn, nil)
	rMock.On("SetVirtualMedia", ctx, isoURL).Times(1).Return(nil)
	rMock.On("SetBootSourceByType", ctx).Times(1).Return(nil)
	rMock.On("NodeID").Times(1).Return(systemID)
	rMock.On("RebootSystem", ctx).Times(1).Return(nil)

	ephemeralHost := baremetalHost{
		rMock,
		redfishURL,
		"doc-name",
		username,
		password,
	}

	settings := initSettings(t, withTestDataPath("base"))
	err = ephemeralHost.DoRemoteDirect(settings)
	assert.NoError(t, err)
}

func TestDoRemoteDirectRedfishNodePoweredOff(t *testing.T) {
	rMock, err := redfishutils.NewClient(redfishURL, false, false, username, password)
	assert.NoError(t, err)

	ctx := context.Background()

	rMock.On("NodeID").Times(1).Return(systemID)
	rMock.On("SystemPowerStatus", ctx).Times(1).Return(power.StatusOff, nil)
	rMock.On("SystemPowerOn", ctx).Times(1).Return(nil)
	rMock.On("SetVirtualMedia", ctx, isoURL).Times(1).Return(nil)
	rMock.On("SetBootSourceByType", ctx).Times(1).Return(nil)
	rMock.On("NodeID").Times(1).Return(systemID)
	rMock.On("RebootSystem", ctx).Times(1).Return(nil)

	ephemeralHost := baremetalHost{
		rMock,
		redfishURL,
		"doc-name",
		username,
		password,
	}

	settings := initSettings(t, withTestDataPath("base"))
	err = ephemeralHost.DoRemoteDirect(settings)
	assert.NoError(t, err)
}

func TestDoRemoteDirectRedfishVirtualMediaError(t *testing.T) {
	rMock, err := redfishutils.NewClient(redfishURL, false, false, username, password)
	assert.NoError(t, err)

	ctx := context.Background()

	expectedErr := redfish.ErrRedfishClient{Message: "Unable to set virtual media."}

	rMock.On("NodeID").Times(1).Return(systemID)
	rMock.On("SystemPowerStatus", ctx).Times(1).Return(power.StatusOn, nil)
	rMock.On("SetVirtualMedia", ctx, isoURL).Times(1).Return(expectedErr)
	rMock.On("SetBootSourceByType", ctx).Times(1).Return(nil)
	rMock.On("NodeID").Times(1).Return(systemID)
	rMock.On("RebootSystem", ctx).Times(1).Return(nil)

	ephemeralHost := baremetalHost{
		rMock,
		redfishURL,
		"doc-name",
		username,
		password,
	}

	settings := initSettings(t, withTestDataPath("base"))

	err = ephemeralHost.DoRemoteDirect(settings)
	_, ok := err.(redfish.ErrRedfishClient)
	assert.True(t, ok)
}

func TestDoRemoteDirectRedfishBootSourceError(t *testing.T) {
	rMock, err := redfishutils.NewClient(redfishURL, false, false, username, password)
	assert.NoError(t, err)

	ctx := context.Background()

	rMock.On("NodeID").Times(1).Return(systemID)
	rMock.On("SystemPowerStatus", ctx).Times(1).Return(power.StatusOn, nil)
	rMock.On("SetVirtualMedia", ctx, isoURL).Times(1).Return(nil)

	expectedErr := redfish.ErrRedfishClient{Message: "Unable to set boot source."}
	rMock.On("SetBootSourceByType", ctx).Times(1).Return(expectedErr)
	rMock.On("NodeID").Times(1).Return(systemID)
	rMock.On("RebootSystem", ctx).Times(1).Return(nil)

	ephemeralHost := baremetalHost{
		rMock,
		redfishURL,
		"doc-name",
		username,
		password,
	}

	settings := initSettings(t, withTestDataPath("base"))

	err = ephemeralHost.DoRemoteDirect(settings)
	_, ok := err.(redfish.ErrRedfishClient)
	assert.True(t, ok)
}

func TestDoRemoteDirectRedfishRebootError(t *testing.T) {
	rMock, err := redfishutils.NewClient(redfishURL, false, false, username, password)
	assert.NoError(t, err)

	ctx := context.Background()

	rMock.On("NodeID").Times(1).Return(systemID)
	rMock.On("SystemPowerStatus", ctx).Times(1).Return(power.StatusOn, nil)
	rMock.On("SetVirtualMedia", ctx, isoURL).Times(1).Return(nil)
	rMock.On("SetVirtualMedia", ctx, isoURL).Times(1).Return(nil)
	rMock.On("SetBootSourceByType", ctx).Times(1).Return(nil)
	rMock.On("NodeID").Times(1).Return(systemID)

	expectedErr := redfish.ErrRedfishClient{Message: "Unable to set boot source."}
	rMock.On("RebootSystem", ctx).Times(1).Return(expectedErr)

	ephemeralHost := baremetalHost{
		rMock,
		redfishURL,
		"doc-name",
		username,
		password,
	}

	settings := initSettings(t, withTestDataPath("base"))

	err = ephemeralHost.DoRemoteDirect(settings)
	_, ok := err.(redfish.ErrRedfishClient)
	assert.True(t, ok)
}
