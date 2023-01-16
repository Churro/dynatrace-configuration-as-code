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

package classic

import (
	"fmt"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/api"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/rest"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestDownloadAllConfigs_FailedToFindConfigsToDownload(t *testing.T) {
	client := rest.NewMockDynatraceClient(gomock.NewController(t))
	client.EXPECT().List(gomock.Any()).Return([]api.Value{}, fmt.Errorf("NO"))
	downloader := NewDownloader(client)
	testAPI := api.NewApi("API_ID", "API_PATH", "", false, true, "", false)
	apiMap := api.ApiMap{"API_ID": testAPI}

	assert.Len(t, downloader.DownloadAll(apiMap, "project"), 0)
}

func TestDownloadAll_NoConfigsToDownloadFound(t *testing.T) {
	client := rest.NewMockDynatraceClient(gomock.NewController(t))
	client.EXPECT().List(gomock.Any()).Return([]api.Value{}, nil)
	downloader := NewDownloader(client)
	testAPI := api.NewApi("API_ID", "API_PATH", "", false, true, "", false)

	apiMap := api.ApiMap{"API_ID": testAPI}

	configurations := downloader.DownloadAll(apiMap, "project")
	assert.Len(t, configurations, 0)
}

func TestDownloadAll_ConfigsDownloaded(t *testing.T) {
	client := rest.NewMockDynatraceClient(gomock.NewController(t))
	client.EXPECT().List(gomock.Any()).DoAndReturn(func(a api.Api) ([]api.Value, error) {
		if a.GetId() == "API_ID_1" {
			return []api.Value{{Id: "API_ID_1", Name: "API_NAME_1"}}, nil
		} else if a.GetId() == "API_ID_2" {
			return []api.Value{{Id: "API_ID_2", Name: "API_NAME_2"}}, nil
		}
		return nil, nil
	}).Times(2)
	downloader := NewDownloader(client)
	testAPI1 := api.NewApi("API_ID_1", "API_PATH_1", "", false, true, "", false)
	testAPI2 := api.NewApi("API_ID_2", "API_PATH_2", "", false, true, "", false)
	client.EXPECT().ReadById(gomock.Any(), gomock.Any()).Return([]byte("{}"), nil)
	client.EXPECT().ReadById(gomock.Any(), gomock.Any()).Return([]byte("{}"), nil)

	apiMap := api.ApiMap{"API_ID_1": testAPI1, "API_ID_2": testAPI2}

	configurations := downloader.DownloadAll(apiMap, "project")
	assert.Len(t, configurations, 2)
}

func TestDownloadAll_ConfigsDownloaded_WithEmptyFilter(t *testing.T) {
	client := rest.NewMockDynatraceClient(gomock.NewController(t))
	client.EXPECT().List(gomock.Any()).DoAndReturn(func(a api.Api) ([]api.Value, error) {
		if a.GetId() == "API_ID_1" {
			return []api.Value{{Id: "API_ID_1", Name: "API_NAME_1"}}, nil
		} else if a.GetId() == "API_ID_2" {
			return []api.Value{{Id: "API_ID_2", Name: "API_NAME_2"}}, nil
		}
		return nil, nil
	}).Times(2)
	downloader := NewDownloader(client, WithAPIFilters(map[string]apiFilter{}))
	testAPI1 := api.NewApi("API_ID_1", "API_PATH_1", "", false, true, "", false)
	testAPI2 := api.NewApi("API_ID_2", "API_PATH_2", "", false, true, "", false)
	client.EXPECT().ReadById(gomock.Any(), gomock.Any()).Return([]byte("{}"), nil)
	client.EXPECT().ReadById(gomock.Any(), gomock.Any()).Return([]byte("{}"), nil)

	apiMap := api.ApiMap{"API_ID_1": testAPI1, "API_ID_2": testAPI2}

	configurations := downloader.DownloadAll(apiMap, "project")
	assert.Len(t, configurations, 2)
}

func TestDownloadAll_SingleConfigurationAPI(t *testing.T) {
	client := rest.NewMockDynatraceClient(gomock.NewController(t))
	client.EXPECT().ReadById(gomock.Any(), gomock.Any()).Return([]byte("{}"), nil)
	downloader := NewDownloader(client)
	testAPI1 := api.NewApi("API_ID_1", "API_PATH_1", "", true, true, "", false)
	apiMap := api.ApiMap{"API_ID_1": testAPI1}

	configurations := downloader.DownloadAll(apiMap, "project")
	assert.Len(t, configurations, 1)
}

func TestDownloadAll_ErrorFetchingConfig(t *testing.T) {
	client := rest.NewMockDynatraceClient(gomock.NewController(t))
	client.EXPECT().List(gomock.Any()).DoAndReturn(func(a api.Api) ([]api.Value, error) {
		if a.GetId() == "API_ID_1" {
			return []api.Value{{Id: "API_ID_1", Name: "API_NAME_1"}}, nil
		} else if a.GetId() == "API_ID_2" {
			return []api.Value{{Id: "API_ID_2", Name: "API_NAME_2"}}, nil
		}
		return nil, nil
	}).Times(2)

	downloader := NewDownloader(client)

	testAPI1 := api.NewApi("API_ID_1", "API_PATH_1", "", false, true, "", false)
	testAPI2 := api.NewApi("API_ID_2", "API_PATH_2", "", false, true, "", false)

	client.EXPECT().ReadById(gomock.Any(), gomock.Any()).DoAndReturn(func(a api.Api, id string) (json []byte, err error) {
		if a.GetId() == "API_ID_1" {
			return []byte("{}"), fmt.Errorf("NO")
		}
		return []byte("{}"), nil
	}).Times(2)

	apiMap := api.ApiMap{"API_ID_1": testAPI1, "API_ID_2": testAPI2}
	configurations := downloader.DownloadAll(apiMap, "project")
	assert.Len(t, configurations, 1)
}

func TestDownloadAll_SkipConfigThatShouldNotBePersisted(t *testing.T) {

	client := rest.NewMockDynatraceClient(gomock.NewController(t))
	client.EXPECT().List(gomock.Any()).DoAndReturn(func(a api.Api) ([]api.Value, error) {
		if a.GetId() == "API_ID_1" {
			return []api.Value{{Id: "API_ID_1", Name: "API_NAME_1"}}, nil
		} else if a.GetId() == "API_ID_2" {
			return []api.Value{{Id: "API_ID_2", Name: "API_NAME_2"}}, nil
		}
		return nil, nil
	}).Times(2)

	apiFilters := map[string]apiFilter{"API_ID_1": {
		shouldConfigBePersisted: func(_ map[string]interface{}) bool {
			return false
		},
	}}
	downloader := NewDownloader(client, WithAPIFilters(apiFilters))

	testAPI1 := api.NewApi("API_ID_1", "API_PATH_1", "", false, true, "", false)
	testAPI2 := api.NewApi("API_ID_2", "API_PATH_2", "", false, true, "", false)
	client.EXPECT().ReadById(gomock.Any(), gomock.Any()).Return([]byte("{}"), nil).Times(2)

	apiMap := api.ApiMap{"API_ID_1": testAPI1, "API_ID_2": testAPI2}

	configurations := downloader.DownloadAll(apiMap, "project")
	assert.Len(t, configurations, 1)
}

func TestDownloadAll_SkipConfigBeforeDownload(t *testing.T) {

	client := rest.NewMockDynatraceClient(gomock.NewController(t))
	client.EXPECT().List(gomock.Any()).DoAndReturn(func(a api.Api) ([]api.Value, error) {
		if a.GetId() == "API_ID_1" {
			return []api.Value{{Id: "API_ID_1", Name: "API_NAME_1"}}, nil
		} else if a.GetId() == "API_ID_2" {
			return []api.Value{{Id: "API_ID_2", Name: "API_NAME_2"}}, nil
		}
		return nil, nil
	}).Times(2)

	apiFilters := map[string]apiFilter{"API_ID_1": {
		shouldBeSkippedPreDownload: func(_ api.Value) bool {
			return true
		},
	}}
	downloader := NewDownloader(client, WithAPIFilters(apiFilters))

	testAPI1 := api.NewApi("API_ID_1", "API_PATH_1", "", false, true, "", false)
	testAPI2 := api.NewApi("API_ID_2", "API_PATH_2", "", false, true, "", false)
	client.EXPECT().ReadById(gomock.Any(), gomock.Any()).Return([]byte("{}"), nil)

	apiMap := api.ApiMap{"API_ID_1": testAPI1, "API_ID_2": testAPI2}

	configurations := downloader.DownloadAll(apiMap, "project")
	assert.Len(t, configurations, 1)
}

func TestDownloadAll_EmptyAPIMap_NothingIsDownloaded(t *testing.T) {
	client := rest.NewMockDynatraceClient(gomock.NewController(t))
	downloader := NewDownloader(client)

	configurations := downloader.DownloadAll(api.ApiMap{}, "project")
	assert.Len(t, configurations, 0)
}

func TestDownloadAll_APIWithoutAnyConfigAvailableAreNotDownloaded(t *testing.T) {
	client := rest.NewMockDynatraceClient(gomock.NewController(t))
	client.EXPECT().List(gomock.Any()).DoAndReturn(func(a api.Api) ([]api.Value, error) {
		if a.GetId() == "API_ID_1" {
			return []api.Value{{Id: "API_ID_1", Name: "API_NAME_1"}}, nil
		} else if a.GetId() == "API_ID_2" {
			return []api.Value{}, nil
		}
		return nil, nil
	}).Times(2)
	downloader := NewDownloader(client)
	testAPI1 := api.NewApi("API_ID_1", "API_PATH_1", "", false, true, "", false)
	testAPI2 := api.NewApi("API_ID_2", "API_PATH_2", "", false, true, "", false)
	client.EXPECT().ReadById(gomock.Any(), gomock.Any()).Return([]byte("{}"), nil)

	apiMap := api.ApiMap{"API_ID_1": testAPI1, "API_ID_2": testAPI2}

	configurations := downloader.DownloadAll(apiMap, "project")
	assert.Len(t, configurations, 1)
}

func TestDownloadAll_MalformedResponseFromAnAPI(t *testing.T) {
	client := rest.NewMockDynatraceClient(gomock.NewController(t))
	client.EXPECT().List(gomock.Any()).DoAndReturn(func(a api.Api) ([]api.Value, error) {
		if a.GetId() == "API_ID_1" {
			return []api.Value{{Id: "API_ID_1", Name: "API_NAME_1"}}, nil
		} else if a.GetId() == "API_ID_2" {
			return []api.Value{{Id: "API_ID_2", Name: "API_NAME_2"}}, nil
		}
		return nil, nil
	}).Times(2)
	downloader := NewDownloader(client)
	testAPI1 := api.NewApi("API_ID_1", "API_PATH_1", "", false, true, "", false)
	testAPI2 := api.NewApi("API_ID_2", "API_PATH_2", "", false, true, "", false)
	client.EXPECT().ReadById(gomock.Any(), gomock.Any()).Return([]byte("-1"), nil)
	client.EXPECT().ReadById(gomock.Any(), gomock.Any()).Return([]byte("{}"), nil)

	apiMap := api.ApiMap{"API_ID_1": testAPI1, "API_ID_2": testAPI2}

	configurations := downloader.DownloadAll(apiMap, "project")
	assert.Len(t, configurations, 1)
}

func TestWithParallelRequestLimitFromEnvOption(t *testing.T) {
	assert.Equal(t, defaultConcurrentDownloads, concurrentRequestLimitFromEnv())
	t.Setenv(concurrentRequestsEnvKey, "51")
	assert.Equal(t, 51, concurrentRequestLimitFromEnv())

}
