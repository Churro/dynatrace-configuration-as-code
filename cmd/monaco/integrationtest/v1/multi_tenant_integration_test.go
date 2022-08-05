//go:build integration_v1
// +build integration_v1

/**
 * @license
 * Copyright 2020 Dynatrace LLC
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package v1

import (
	"path/filepath"
	"testing"

	"gotest.tools/assert"

	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/cmd/monaco/runner"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/api"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/environment"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/project"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/util"
	"github.com/spf13/afero"
)

var folder = AbsOrPanicFromSlash("test-resources/integration-multi-environment/")
var environmentsFile = filepath.Join(folder, "environments.yaml")

// Tests all environments with all projects
func TestIntegrationMultiEnvironment(t *testing.T) {

	RunLegacyIntegrationWithCleanup(t, folder, environmentsFile, "MultiEnvironment", func(fs afero.Fs) {

		environments, errs := environment.LoadEnvironmentList("", environmentsFile, fs)
		assert.Check(t, len(errs) == 0, "didn't expect errors loading test environments")

		projects, err := project.LoadProjectsToDeploy(fs, "", api.NewV1Apis(), folder)
		assert.NilError(t, err)

		cmd := runner.BuildCli(fs)
		cmd.SetArgs([]string{
			"deploy",
			"--verbose",
			"--environments", environmentsFile,
			folder,
		})
		err = cmd.Execute()

		AssertAllConfigsAvailability(projects, t, environments, true)

		assert.NilError(t, err)
	})
}

// Tests a dry run (validation)
func TestIntegrationValidationMultiEnvironment(t *testing.T) {
	t.Setenv("CONFIG_V1", "1")

	cmd := runner.BuildCli(util.CreateTestFileSystem())
	cmd.SetArgs([]string{
		"deploy",
		"--verbose",
		"--environments", environmentsFile,
		"--dry-run",
		folder,
	})
	err := cmd.Execute()

	assert.NilError(t, err)
}

// tests a single project
func TestIntegrationMultiEnvironmentSingleProject(t *testing.T) {

	RunLegacyIntegrationWithCleanup(t, folder, environmentsFile, "MultiEnvironmentSingleProject", func(fs afero.Fs) {

		environments, errs := environment.LoadEnvironmentList("", environmentsFile, fs)
		FailOnAnyError(errs, "loading of environments failed")

		projects, err := project.LoadProjectsToDeploy(fs, "cinema-infrastructure", api.NewV1Apis(), folder)
		assert.NilError(t, err)

		cmd := runner.BuildCli(fs)
		cmd.SetArgs([]string{
			"deploy",
			"--verbose",
			"--environments", environmentsFile,
			"-p", "cinema-infrastructure",
			folder,
		})
		err = cmd.Execute()

		AssertAllConfigsAvailability(projects, t, environments, true)

		assert.NilError(t, err)
	})
}

// Tests a single project with dependency
func TestIntegrationMultiEnvironmentSingleProjectWithDependency(t *testing.T) {

	RunLegacyIntegrationWithCleanup(t, folder, environmentsFile, "MultiEnvironmentSingleProjectWithDependency", func(fs afero.Fs) {

		environments, errs := environment.LoadEnvironmentList("", environmentsFile, fs)
		FailOnAnyError(errs, "loading of environments failed")

		projects, err := project.LoadProjectsToDeploy(fs, "star-trek", api.NewV1Apis(), folder)
		assert.NilError(t, err)

		assert.Check(t, len(projects) == 2, "Projects should be star-trek and the dependency cinema-infrastructure")

		cmd := runner.BuildCli(fs)
		cmd.SetArgs([]string{
			"deploy",
			"--verbose",
			"--environments", environmentsFile,
			"-p", "star-trek",
			folder,
		})
		err = cmd.Execute()

		AssertAllConfigsAvailability(projects, t, environments, true)

		assert.NilError(t, err)
	})
}

// tests a single environment
func TestIntegrationMultiEnvironmentSingleEnvironment(t *testing.T) {

	RunLegacyIntegrationWithCleanup(t, folder, environmentsFile, "MultiEnvironmentSingleEnvironment", func(fs afero.Fs) {

		environments, errs := environment.LoadEnvironmentList("", environmentsFile, fs)
		FailOnAnyError(errs, "loading of environments failed")

		projects, err := project.LoadProjectsToDeploy(fs, "star-trek", api.NewV1Apis(), folder)
		assert.NilError(t, err)

		// remove environment odt69781, just keep dav48679
		delete(environments, "odt69781")

		cmd := runner.BuildCli(fs)
		cmd.SetArgs([]string{
			"deploy",
			"--verbose",
			"--environments", environmentsFile,
			folder,
		})
		err = cmd.Execute()

		AssertAllConfigsAvailability(projects, t, environments, true)

		assert.NilError(t, err)
	})
}