//go:build integration || download_restore || nightly

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

package v2

import (
	"errors"
	"fmt"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/config/v2/coordinate"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/config/v2/parameter"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/deploy"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/manifest"
	project "github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/project/v2"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/project/v2/topologysort"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/util/test"
	"math/rand"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/api"
	config "github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/config/v2"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/rest"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/util"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/util/log"
	"github.com/spf13/afero"
	"gotest.tools/assert"
)

// AssertAllConfigsAvailability checks all configurations of a given project with given availability
func AssertAllConfigsAvailability(t *testing.T, fs afero.Fs, manifestPath string, specificProjects []string, specificEnvironment string, available bool) {
	loadedManifest, errs := manifest.LoadManifest(&manifest.ManifestLoaderContext{
		Fs:           fs,
		ManifestPath: manifestPath,
	})
	test.FailTestOnAnyError(t, errs, "loading of environments failed")

	var specificEnvs []string
	if specificEnvironment != "" {
		specificEnvs = append(specificEnvs, specificEnvironment)
	}
	environments, err := loadedManifest.FilterEnvironmentsByNames(specificEnvs)
	if err != nil {
		t.Fatalf("Failed to filter environments: %v", err)
	}

	cwd, err := filepath.Abs(filepath.Dir(manifestPath))
	assert.NilError(t, err)

	projects, errs := project.LoadProjects(fs, project.ProjectLoaderContext{
		KnownApis:       api.GetApiNameLookup(api.NewApis()),
		WorkingDir:      cwd,
		Manifest:        loadedManifest,
		ParametersSerde: config.DefaultParameterParsers,
	})
	test.FailTestOnAnyError(t, errs, "loading of projects failed")

	envNames := make([]string, 0, len(environments))

	for _, env := range environments {
		envNames = append(envNames, env.Name)
	}

	sortedConfigs, errs := topologysort.GetSortedConfigsForEnvironments(projects, envNames)
	test.FailTestOnAnyError(t, errs, "sorting configurations failed")

	checkString := "exist"
	if !available {
		checkString = "do NOT exist"
	}

	projectsToValidate := map[string]struct{}{}
	if len(specificProjects) > 0 {
		log.Info("Asserting configurations from projects: %s %s", specificProjects, checkString)
		for _, p := range specificProjects {
			projectsToValidate[p] = struct{}{}
		}
	} else {
		log.Info("Asserting configurations from all projects %s", checkString)
		for _, p := range projects {
			projectsToValidate[p.Id] = struct{}{}
		}
	}

	for envName, configs := range sortedConfigs {

		env := loadedManifest.Environments[envName]

		token, err := env.GetToken()
		assert.NilError(t, err)

		url, err := env.GetUrl()
		assert.NilError(t, err)

		client, err := rest.NewDynatraceClient(url, token)
		assert.NilError(t, err)

		entities := make(map[coordinate.Coordinate]parameter.ResolvedEntity)
		var parameters []topologysort.ParameterWithName

		for _, theConfig := range configs {
			coord := theConfig.Coordinate

			if theConfig.Skip {
				entities[coord] = parameter.ResolvedEntity{
					EntityName: coord.ConfigId,
					Coordinate: coord,
					Properties: parameter.Properties{},
					Skip:       true,
				}
				continue
			}

			configParameters, errs := topologysort.SortParameters(theConfig.Group, theConfig.Environment, theConfig.Coordinate, theConfig.Parameters)
			test.FailTestOnAnyError(t, errs, "sorting of parameter values failed")

			parameters = append(parameters, configParameters...)

			properties, errs := deploy.ResolveParameterValues(&theConfig, entities, parameters)
			test.FailTestOnAnyError(t, errs, "resolving of parameter values failed")

			properties[config.IdParameter] = "NO REAL ID NEEDED FOR CHECKING AVAILABILITY"

			configName, err := extractConfigName(&theConfig, properties)
			assert.NilError(t, err)

			entities[coord] = parameter.ResolvedEntity{
				EntityName: configName,
				Coordinate: coord,
				Properties: properties,
				Skip:       false,
			}

			if _, found := projectsToValidate[coord.Project]; found {
				AssertConfig(t, client, env, available, theConfig, coord.Type, configName)
			}
		}
	}
}

func AssertConfig(t *testing.T, client rest.ConfigClient, environment manifest.EnvironmentDefinition, shouldBeAvailable bool, config config.Config, configType string, name string) {

	theApi := api.NewApis()[configType]

	var exists bool

	if config.Skip {
		exists, _, _ = client.ExistsByName(theApi, name)
		assert.Check(t, !exists, "Object should NOT be available, but was. environment.Environment: '%s', failed for '%s' (%s)", environment.Name, name, configType)
		return
	}

	description := fmt.Sprintf("%s %s on environment %s", configType, name, environment.Name)

	// To deal with delays of configs becoming available try for max 120 polling cycles (4min - at 2sec cycles) for expected state to be reached
	err := wait(description, 120, func() bool {
		exists, _, _ = client.ExistsByName(theApi, name)
		return (shouldBeAvailable && exists) || (!shouldBeAvailable && !exists)
	})
	assert.NilError(t, err)

	if shouldBeAvailable {
		assert.Check(t, exists, "Object should be available, but wasn't. environment.Environment: '%s', failed for '%s' (%s)", environment.Name, name, configType)
	} else {
		assert.Check(t, !exists, "Object should NOT be available, but was. environment.Environment: '%s', failed for '%s' (%s)", environment.Name, name, configType)
	}
}

func getTimestamp() string {
	return time.Now().Format("20060102150405")
}

func addSuffix(suffix string) func(line string) string {
	var f = func(name string) string {
		return name + "_" + suffix
	}
	return f
}

func getTransformerFunc(suffix string) func(line string) string {
	var f = func(name string) string {
		return util.ReplaceName(name, addSuffix(suffix))
	}
	return f
}

// Deletes all configs that end with "_suffix", where suffix == suffixTest+suffixTimestamp
func cleanupIntegrationTest(t *testing.T, loadedManifest manifest.Manifest, specificEnvironment, suffix string) {

	log.Info("### Cleaning up after integration test ###")

	var specificEnvs []string
	if specificEnvironment != "" {
		specificEnvs = append(specificEnvs, specificEnvironment)
	}
	environments, err := loadedManifest.FilterEnvironmentsByNames(specificEnvs)
	if err != nil {
		log.Fatal("Failed to filter environments: %v", err)
	}

	apis := api.NewApis()
	suffix = "_" + suffix

	for _, environment := range environments {

		token, err := environment.GetToken()
		assert.NilError(t, err)

		url, err := environment.GetUrl()
		assert.NilError(t, err)

		client, err := rest.NewDynatraceClient(url, token)
		assert.NilError(t, err)

		for _, api := range apis {
			if api.GetId() == "calculated-metrics-log" {
				t.Logf("Skipping cleanup of legacy log monitoring API")
				continue
			}

			values, err := client.List(api)
			if err != nil {
				t.Logf("Failed to cleanup any test configs of type %q: %v", api.GetId(), err)
			}

			for _, value := range values {
				// For the calculated-metrics-log API, the suffix is part of the ID, not name
				if strings.HasSuffix(value.Name, suffix) || strings.HasSuffix(value.Id, suffix) {
					err := client.DeleteById(api, value.Id)
					if err != nil {
						t.Logf("Failed to cleanup test config: %s (%s): %v", value.Name, api.GetId(), err)
					} else {
						log.Info("Cleaned up test config %s (%s)", value.Name, value.Id)
					}
				}
			}
		}
	}
}

// RunIntegrationWithCleanup runs an integration test and cleans up the created configs afterwards
// This is done by using InMemoryFileReader, which rewrites the names of the read configs internally. It ready all the
// configs once and holds them in memory. Any subsequent modification of a config (applying them to an environment)
// is done based on the data in memory. The re-writing of config names ensures, that they have an unique name and don't
// conflict with other configs created by other integration tests.
//
// After the test run, the unique name also helps with finding the applied configs in all the environments and calling
// the respective DELETE api.
//
// The new naming scheme of created configs is defined in a transformer function. By default, this is:
//
// <original name>_<current timestamp><defined suffix>
// e.g. my-config_1605258980000_Suffix

func RunIntegrationWithCleanup(t *testing.T, configFolder, manifestPath, specificEnvironment, suffixTest string, testFunc func(fs afero.Fs)) {

	fs := util.CreateTestFileSystem()
	runIntegrationWithCleanup(t, fs, configFolder, manifestPath, specificEnvironment, suffixTest, nil, testFunc)
}

func RunIntegrationWithCleanupOnGivenFs(t *testing.T, testFs afero.Fs, configFolder, manifestPath, specificEnvironment, suffixTest string, testFunc func(fs afero.Fs)) {
	runIntegrationWithCleanup(t, testFs, configFolder, manifestPath, specificEnvironment, suffixTest, nil, testFunc)
}

func RunIntegrationWithCleanupGivenEnvs(t *testing.T, configFolder, manifestPath, specificEnvironment, suffixTest string, envVars map[string]string, testFunc func(fs afero.Fs)) {
	fs := util.CreateTestFileSystem()

	runIntegrationWithCleanup(t, fs, configFolder, manifestPath, specificEnvironment, suffixTest, envVars, testFunc)
}

func runIntegrationWithCleanup(t *testing.T, testFs afero.Fs, configFolder, manifestPath, specificEnvironment, suffixTest string, envVars map[string]string, testFunc func(fs afero.Fs)) {
	loadedManifest, errs := manifest.LoadManifest(&manifest.ManifestLoaderContext{
		Fs:           testFs,
		ManifestPath: manifestPath,
	})
	test.FailTestOnAnyError(t, errs, "loading of manifest failed")

	configFolder, _ = filepath.Abs(configFolder)

	suffix := appendUniqueSuffixToIntegrationTestConfigs(t, testFs, configFolder, suffixTest)

	t.Cleanup(func() {
		cleanupIntegrationTest(t, loadedManifest, specificEnvironment, suffix)
	})

	for k, v := range envVars {
		t.Setenv(k, v) // register both just in case
		t.Setenv(fmt.Sprintf("%s_%s", k, suffix), v)
	}

	testFunc(testFs)
}

func appendUniqueSuffixToIntegrationTestConfigs(t *testing.T, fs afero.Fs, configFolder string, generalSuffix string) string {
	rand.Seed(time.Now().UnixNano())
	randomNumber := rand.Intn(10000)

	suffix := fmt.Sprintf("%s_%d_%s", getTimestamp(), randomNumber, generalSuffix)
	transformers := []func(string) string{getTransformerFunc(suffix)}

	err := util.RewriteConfigNames(configFolder, fs, transformers)
	if err != nil {
		t.Fatalf("Error rewriting configs names: %s", err)
		return suffix
	}

	return suffix
}

func wait(description string, maxPollCount int, condition func() bool) error {

	for i := 0; i <= maxPollCount; i++ {

		if condition() {
			return nil
		}
		time.Sleep(2 * time.Second)
	}

	log.Error("Error: Waiting for '%s' timed out!", description)

	return errors.New("Waiting for '" + description + "' timed out!")
}

func extractConfigName(conf *config.Config, properties parameter.Properties) (string, error) {
	val, found := properties[config.NameParameter]

	if !found {
		return "", fmt.Errorf("missing `name` for config")
	}

	name, success := val.(string)

	if !success {
		return "", fmt.Errorf("`name` in config is not of type string")
	}

	return name, nil
}
