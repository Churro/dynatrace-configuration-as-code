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

package classic

import (
	"fmt"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/api"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config"
	"github.com/google/go-cmp/cmp"
)

type (
	environmentName = string
	classicEndpoint = string
)

type Validator struct {
	uniqueNames map[environmentName]map[classicEndpoint][]config.Config
}

var apis = api.NewAPIs()

// Validate checks that for each classic config API type, only one config exists with any given name.
// As classic configs are identified by name, ValidateUniqueConfigNames returns errors if a name is used more than once for the same type.
func (v *Validator) Validate(c config.Config) error {
	if v.uniqueNames == nil {
		v.uniqueNames = make(map[environmentName]map[classicEndpoint][]config.Config)
	}

	a, ok := c.Type.(config.ClassicApiType)
	if !ok || apis[a.Api].NonUniqueName {
		return nil
	}

	if v.uniqueNames[c.Environment] == nil {
		v.uniqueNames[c.Environment] = make(map[classicEndpoint][]config.Config)
	}

	for _, c2 := range v.uniqueNames[c.Environment][a.Api] {
		n1, err := config.GetNameForConfig(c)
		if err != nil {
			return err
		}
		n2, err := config.GetNameForConfig(c2)
		if err != nil {
			return err
		}

		if cmp.Equal(n1, n2) {
			var nameDetails string
			if s, ok := n1.(string); ok {
				nameDetails = fmt.Sprintf(": %s", s)
			}

			return fmt.Errorf("duplicated config name found: configurations %s and %s define the same 'name' %q", c.Coordinate, c2.Coordinate, nameDetails)
		}
	}

	v.uniqueNames[c.Environment][a.Api] = append(v.uniqueNames[c.Environment][a.Api], c)
	return nil
}
