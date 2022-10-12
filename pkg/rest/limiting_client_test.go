//go:build unit

// @license
// Copyright 2022 Dynatrace LLC
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package rest

import (
	"errors"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/api"
	"github.com/golang/mock/gomock"
	"gotest.tools/assert"
	"testing"
)

var givenJson = []byte{1, 2, 3}
var givenError = errors.New("error")

func TestDecoratedClient_ReadByName(t *testing.T) {
	a := api.NewMockApi(gomock.NewController(t))

	client := NewMockDynatraceClient(gomock.NewController(t))
	limited := LimitClientParallelRequests(client, 1)

	client.EXPECT().ReadByName(a, "name").Return(givenJson, givenError)

	j, e := limited.ReadByName(a, "name")

	assert.DeepEqual(t, j, givenJson)
	assert.Equal(t, e, givenError)
}