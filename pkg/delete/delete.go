/*
 * @license
 * Copyright 2023 Dynatrace LLC
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

package delete

import (
	"context"
	"fmt"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log/field"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/api"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/client/dtclient"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/delete/internal/automation"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/delete/internal/bucket"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/delete/internal/classic"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/delete/internal/setting"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/delete/pointer"
	"reflect"
)

type ClientSet struct {
	Classic    dtclient.Client
	Settings   dtclient.Client
	Automation automation.Client
	Buckets    bucket.Client
}

type configurationType = string

// DeleteEntries is a map of configuration type to slice of delete pointers
type DeleteEntries = map[configurationType][]pointer.DeletePointer

// Configs removes all given entriesToDelete from the Dynatrace environment the given client connects to
func Configs(ctx context.Context, clients ClientSet, apis api.APIs, automationResources map[string]config.AutomationResource, entriesToDelete DeleteEntries) error {
	deleteErrors := 0
	for entryType, entries := range entriesToDelete {
		if entryType == api.DashboardShareSettings {
			log.Warn("Classic config of type %s cannot be deleted. Note, that they can be removed by deleting the associated dashboard.", api.DashboardShareSettings)
			continue
		}

		var err error
		if targetApi, isClassicAPI := apis[entryType]; isClassicAPI {
			err = classic.Delete(ctx, clients.Classic, targetApi, entries, entryType)
		} else if targetAutomation, isAutomationAPI := automationResources[entryType]; isAutomationAPI {
			if reflect.ValueOf(clients.Automation).IsNil() {
				log.WithCtxFields(ctx).WithFields(field.Type(entryType)).Warn("Skipped deletion of %d Automation configuration(s) of type %q as API client was unavailable.", len(entries), entryType)
				continue
			}
			err = automation.Delete(ctx, clients.Automation, targetAutomation, entries)
		} else if entryType == "bucket" {
			if reflect.ValueOf(clients.Buckets).IsNil() {
				log.WithCtxFields(ctx).WithFields(field.Type(entryType)).Warn("Skipped deletion of %d Grail Bucket configuration(s) as API client was unavailable.", len(entries))
				continue
			}
			err = bucket.Delete(ctx, clients.Buckets, entries)
		} else { // assume it's a Settings Schema
			err = setting.Delete(ctx, clients.Settings, entries)
		}

		if err != nil {
			log.WithFields(field.Error(err)).Error("Error during deletion: %v", err)
			deleteErrors += 1
		}
	}

	if deleteErrors > 0 {
		return fmt.Errorf("encountered %d errors", deleteErrors)
	}
	return nil
}

// All collects and deletes ALL configuration objects using the provided ClientSet.
// To delete specific configurations use Configs instead!
//
// Parameters:
//   - ctx (context.Context): The context in which the function operates.
//   - clients (ClientSet): A set of API clients used to collect and delete configurations from an environment.
func All(ctx context.Context, clients ClientSet, apis api.APIs) error {
	errs := 0

	if err := classic.DeleteAll(ctx, clients.Classic, apis); err != nil {
		log.Error("Failed to delete all classic API configurations: %v", err)
		errs++
	}

	if err := setting.DeleteAll(ctx, clients.Settings); err != nil {
		log.Error("Failed to delete all Settings 2.0 objects: %v", err)
		errs++
	}

	if reflect.ValueOf(clients.Automation).IsNil() {
		log.Warn("Skipped deletion of Automation configurations as API client was unavailable.")
	} else if err := automation.DeleteAll(ctx, clients.Automation); err != nil {
		log.Error("Failed to delete all Automation configurations: %v", err)
		errs++
	}

	if reflect.ValueOf(clients.Buckets).IsNil() {
		log.Warn("Skipped deletion of Grail Bucket configurations as API client was unavailable.")
	} else if err := bucket.DeleteAll(ctx, clients.Buckets); err != nil {
		log.Error("Failed to delete all Grail Bucket configurations: %v", err)
		errs++
	}

	if errs > 0 {
		return fmt.Errorf("failed to delete all configurations for %d types", errs)
	}
	return nil
}
