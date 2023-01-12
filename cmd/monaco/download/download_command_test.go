// @license
// Copyright 2021 Dynatrace LLC
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

//go:build unit

package download

import (
	"github.com/golang/mock/gomock"
	"github.com/spf13/afero"
	"gotest.tools/assert"

	"io"
	"strings"
	"testing"
)

// TestInvalidCliCommands is a very basic test testing that invalid commands error.
// It is not the goal to test the exact message that cobra generates, except if we supply the message.
// Otherwise, we would run into issues upon upgrading.
// On th other hand, we could use the exact message to review the exact messages customers will see.
func TestInvalidCliCommands(t *testing.T) {
	tests := []struct {
		name                  string
		args                  string
		errorContainsExpected []string
	}{
		{
			"no arguments provided",
			"",
			[]string{"sub-command is required"},
		},
		{
			"no arguments provided to direct download",
			"direct",
			[]string{"url and token have to be provided as positional argument"},
		},
		{
			"url is missing other required argument",
			"direct some.env.url.com",
			[]string{"url and token have to be provided as positional argument"},
		},
		{
			"manifest provided but missing specific environment",
			"manifest manifest.yaml",
			[]string{"manifest and environment name have to be provided as positional arguments"},
		},
		{
			"manifest is missing but environment is provider",
			"manifest some_env",
			[]string{"manifest and environment name have to be provided as positional arguments"},
		},
		{
			"unknown flag",
			"--test",
			[]string{"--test"},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			commandMock := createDownloadCommandMock(t)

			cmd := GetDownloadCommand(afero.NewOsFs(), commandMock)
			cmd.SetArgs(strings.Split(test.args, " "))
			cmd.SetOut(io.Discard) // skip output to ensure that the error message contains the error, not the help message
			err := cmd.Execute()

			// for all test cases there should be at least an error
			assert.Assert(t, err != nil, "there should be an error")

			// for most cases we can test the message in more detail
			for _, expected := range test.errorContainsExpected {
				assert.ErrorContains(t, err, expected)
			}

			// for testing not to forget adding expectations
			assert.Assert(t, len(test.errorContainsExpected) > 0, "no error conditions specified")
		})
	}
}

func TestValidCommands(t *testing.T) {
	tests := []struct {
		name  string
		args  string
		setup func(command *MockCommand)
	}{
		{
			"direct download no specific apis",
			"direct test.url token --project test",
			func(cmd *MockCommand) {
				cmd.EXPECT().DownloadConfigs(gomock.Any(), directDownloadOptions{
					environmentUrl: "test.url",
					envVarName:     "token",
					downloadCommandOptions: downloadCommandOptions{
						projectName:     "test",
						outputFolder:    "",
						forceOverwrite:  false,
						specificAPIs:    []string{},
						specificSchemas: []string{},
					},
				})
			},
		},
		{
			"direct download with default project",
			"direct test.url token",
			func(cmd *MockCommand) {
				cmd.EXPECT().DownloadConfigs(gomock.Any(), directDownloadOptions{
					environmentUrl: "test.url",
					envVarName:     "token",
					downloadCommandOptions: downloadCommandOptions{
						projectName:     "project",
						outputFolder:    "",
						forceOverwrite:  false,
						specificAPIs:    []string{},
						specificSchemas: []string{},
					},
				})
			},
		},
		{
			"direct download - skip download of settings",
			"direct test.url token --skip-settings",
			func(cmd *MockCommand) {
				cmd.EXPECT().DownloadConfigs(gomock.Any(), directDownloadOptions{
					environmentUrl: "test.url",
					envVarName:     "token",
					downloadCommandOptions: downloadCommandOptions{
						projectName:     "project",
						outputFolder:    "",
						forceOverwrite:  false,
						specificAPIs:    []string{},
						specificSchemas: []string{},
						skipSettings:    true,
					},
				})
			},
		},
		{
			"direct download with specific apis (multiple flags)",
			"direct test.url token --project test --specific-api test --specific-api test2",
			func(cmd *MockCommand) {
				cmd.EXPECT().DownloadConfigs(gomock.Any(), directDownloadOptions{
					environmentUrl: "test.url",
					envVarName:     "token",
					downloadCommandOptions: downloadCommandOptions{
						projectName:     "test",
						outputFolder:    "",
						forceOverwrite:  false,
						specificAPIs:    []string{"test", "test2"},
						specificSchemas: []string{},
					},
				})
			},
		},
		{
			"direct download with specific apis (single flag)",
			"direct test.url token --project test --specific-api test,test2",
			func(cmd *MockCommand) {
				cmd.EXPECT().DownloadConfigs(gomock.Any(), directDownloadOptions{
					environmentUrl: "test.url",
					envVarName:     "token",
					downloadCommandOptions: downloadCommandOptions{
						projectName:     "test",
						outputFolder:    "",
						forceOverwrite:  false,
						specificAPIs:    []string{"test", "test2"},
						specificSchemas: []string{},
					},
				})
			},
		},
		{
			"direct download with specific apis (mixed flags)",
			"direct test.url token --project test --specific-api test,test2 --specific-api test3",
			func(cmd *MockCommand) {
				cmd.EXPECT().DownloadConfigs(gomock.Any(), directDownloadOptions{
					environmentUrl: "test.url",
					envVarName:     "token",
					downloadCommandOptions: downloadCommandOptions{
						projectName:     "test",
						outputFolder:    "",
						forceOverwrite:  false,
						specificAPIs:    []string{"test", "test2", "test3"},
						specificSchemas: []string{},
					},
				})
			},
		},
		{
			"direct download with specific settings (single flag)",
			"direct test.url token --project test --specific-settings builtin:alerting.profile,builtin:problem.notifications",
			func(cmd *MockCommand) {
				cmd.EXPECT().DownloadConfigs(gomock.Any(), directDownloadOptions{
					environmentUrl: "test.url",
					envVarName:     "token",
					downloadCommandOptions: downloadCommandOptions{
						projectName:     "test",
						specificAPIs:    []string{},
						specificSchemas: []string{"builtin:alerting.profile", "builtin:problem.notifications"},
					},
				})
			},
		},
		{
			"direct download with specific settings (mixed flags)",
			"direct test.url token --project test --specific-settings builtin:alerting.profile,builtin:problem.notifications --specific-settings builtin:metric.metadata",
			func(cmd *MockCommand) {
				cmd.EXPECT().DownloadConfigs(gomock.Any(), directDownloadOptions{
					environmentUrl: "test.url",
					envVarName:     "token",
					downloadCommandOptions: downloadCommandOptions{
						projectName:     "test",
						specificAPIs:    []string{},
						specificSchemas: []string{"builtin:alerting.profile", "builtin:problem.notifications", "builtin:metric.metadata"},
					},
				})
			},
		},
		{
			"direct download with outputfolder",
			"direct test.url token --output-folder myDownloads",
			func(cmd *MockCommand) {
				cmd.EXPECT().DownloadConfigs(gomock.Any(), directDownloadOptions{
					environmentUrl: "test.url",
					envVarName:     "token",
					downloadCommandOptions: downloadCommandOptions{
						projectName:     "project",
						outputFolder:    "myDownloads",
						forceOverwrite:  false,
						specificAPIs:    []string{},
						specificSchemas: []string{},
					},
				})
			},
		},
		{
			"direct download with output-folder and force overwrite",
			"direct test.url token --output-folder myDownloads --force",
			func(cmd *MockCommand) {
				cmd.EXPECT().DownloadConfigs(gomock.Any(), directDownloadOptions{
					environmentUrl: "test.url",
					envVarName:     "token",
					downloadCommandOptions: downloadCommandOptions{
						projectName:     "project",
						outputFolder:    "myDownloads",
						forceOverwrite:  true,
						specificAPIs:    []string{},
						specificSchemas: []string{},
					},
				})
			},
		},
		{
			"manifest download no specific apis",
			"manifest test.yaml test_env",
			func(cmd *MockCommand) {
				cmd.EXPECT().DownloadConfigsBasedOnManifest(gomock.Any(), manifestDownloadOptions{
					manifestFile:            "test.yaml",
					specificEnvironmentName: "test_env",
					downloadCommandOptions: downloadCommandOptions{
						projectName:     "project",
						outputFolder:    "",
						forceOverwrite:  false,
						specificAPIs:    []string{},
						specificSchemas: []string{},
					},
				})
			},
		},
		{
			"manifest download - skip download of settings ",
			"manifest test.yaml test_env --skip-settings",
			func(cmd *MockCommand) {
				cmd.EXPECT().DownloadConfigsBasedOnManifest(gomock.Any(), manifestDownloadOptions{
					manifestFile:            "test.yaml",
					specificEnvironmentName: "test_env",
					downloadCommandOptions: downloadCommandOptions{
						projectName:     "project",
						outputFolder:    "",
						forceOverwrite:  false,
						specificAPIs:    []string{},
						specificSchemas: []string{},
						skipSettings:    true,
					},
				})
			},
		},
		{
			"manifest download with specific apis (multiple flags)",
			"manifest test.yaml test_env --specific-api test --specific-api test2",
			func(cmd *MockCommand) {
				cmd.EXPECT().DownloadConfigsBasedOnManifest(gomock.Any(), manifestDownloadOptions{
					manifestFile:            "test.yaml",
					specificEnvironmentName: "test_env",
					downloadCommandOptions: downloadCommandOptions{
						projectName:     "project",
						outputFolder:    "",
						forceOverwrite:  false,
						specificAPIs:    []string{"test", "test2"},
						specificSchemas: []string{},
					},
				})
			},
		},
		{
			"manifest download with specific apis (single flag)",
			"manifest test.yaml test_env --specific-api test,test2",
			func(cmd *MockCommand) {
				cmd.EXPECT().DownloadConfigsBasedOnManifest(gomock.Any(), manifestDownloadOptions{
					manifestFile:            "test.yaml",
					specificEnvironmentName: "test_env",
					downloadCommandOptions: downloadCommandOptions{
						projectName:     "project",
						outputFolder:    "",
						forceOverwrite:  false,
						specificAPIs:    []string{"test", "test2"},
						specificSchemas: []string{},
					},
				})
			},
		},
		{
			"manifest download with specific apis (mixed flags)",
			"manifest test.yaml test_env --specific-api test,test2 --specific-api test3",
			func(cmd *MockCommand) {
				cmd.EXPECT().DownloadConfigsBasedOnManifest(gomock.Any(), manifestDownloadOptions{
					manifestFile:            "test.yaml",
					specificEnvironmentName: "test_env",
					downloadCommandOptions: downloadCommandOptions{
						projectName:     "project",
						outputFolder:    "",
						forceOverwrite:  false,
						specificAPIs:    []string{"test", "test2", "test3"},
						specificSchemas: []string{},
					},
				})
			},
		},
		{
			"manifest download with specific apis (single flag)",
			"manifest test.yaml test_env --specific-settings builtin:alerting.profile,builtin:problem.notifications",
			func(cmd *MockCommand) {
				cmd.EXPECT().DownloadConfigsBasedOnManifest(gomock.Any(), manifestDownloadOptions{
					manifestFile:            "test.yaml",
					specificEnvironmentName: "test_env",
					downloadCommandOptions: downloadCommandOptions{
						projectName:     "project",
						specificAPIs:    []string{},
						specificSchemas: []string{"builtin:alerting.profile", "builtin:problem.notifications"},
					},
				})
			},
		},
		{
			"manifest download with specific apis (mixed flags)",
			"manifest test.yaml test_env --specific-settings builtin:alerting.profile,builtin:problem.notifications --specific-settings builtin:metric.metadata",
			func(cmd *MockCommand) {
				cmd.EXPECT().DownloadConfigsBasedOnManifest(gomock.Any(), manifestDownloadOptions{
					manifestFile:            "test.yaml",
					specificEnvironmentName: "test_env",
					downloadCommandOptions: downloadCommandOptions{
						projectName:     "project",
						specificAPIs:    []string{},
						specificSchemas: []string{"builtin:alerting.profile", "builtin:problem.notifications", "builtin:metric.metadata"},
					},
				})
			},
		},
		{
			"manifest download with project",
			"manifest test.yaml test_env --project testproject",
			func(cmd *MockCommand) {
				cmd.EXPECT().DownloadConfigsBasedOnManifest(gomock.Any(), manifestDownloadOptions{
					manifestFile:            "test.yaml",
					specificEnvironmentName: "test_env",
					downloadCommandOptions: downloadCommandOptions{
						projectName:     "testproject",
						outputFolder:    "",
						forceOverwrite:  false,
						specificAPIs:    []string{},
						specificSchemas: []string{},
					},
				})
			},
		},
		{
			"manifest download with outputfolder",
			"manifest test.yaml test_env --output-folder myDownloads",
			func(cmd *MockCommand) {
				cmd.EXPECT().DownloadConfigsBasedOnManifest(gomock.Any(), manifestDownloadOptions{
					manifestFile:            "test.yaml",
					specificEnvironmentName: "test_env",
					downloadCommandOptions: downloadCommandOptions{
						projectName:     "project",
						outputFolder:    "myDownloads",
						forceOverwrite:  false,
						specificAPIs:    []string{},
						specificSchemas: []string{},
					},
				})
			},
		},
		{
			"manifest download with output-folder and force overwrite",
			"manifest test.yaml test_env --output-folder myDownloads --force",
			func(cmd *MockCommand) {
				cmd.EXPECT().DownloadConfigsBasedOnManifest(gomock.Any(), manifestDownloadOptions{
					manifestFile:            "test.yaml",
					specificEnvironmentName: "test_env",
					downloadCommandOptions: downloadCommandOptions{
						projectName:     "project",
						outputFolder:    "myDownloads",
						forceOverwrite:  true,
						specificAPIs:    []string{},
						specificSchemas: []string{},
					},
				})
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			commandMock := createDownloadCommandMock(t)
			test.setup(commandMock)

			cmd := GetDownloadCommand(afero.NewOsFs(), commandMock)
			cmd.SetArgs(strings.Split(test.args, " "))
			cmd.SetOut(io.Discard) // skip output to ensure that the error message contains the error, not the help message
			err := cmd.Execute()

			assert.NilError(t, err, "no error expected")
		})
	}
}

// createDownloadCommandMock creates the mock for the download command
// finish is called automatically by the testing system to ensure expected calls
func createDownloadCommandMock(t *testing.T) *MockCommand {
	mockCtrl := gomock.NewController(t)
	commandMock := NewMockCommand(mockCtrl)
	return commandMock
}
