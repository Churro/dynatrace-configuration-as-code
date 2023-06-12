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

package loggers

import "github.com/dynatrace/dynatrace-configuration-as-code/pkg/config/v2/coordinate"

// Field is an additional custom field that can be used for structural logging output
type Field struct {
	// Key is the key used for the field
	Key string
	// Value is the value used for the field and can be anything
	Value any
}

// F creates a new custom field for the logger
func F(key string, value interface{}) Field {
	return Field{Key: key, Value: value}
}

// CoordinateF builds a Field containing information taken from the provided coordinate
func CoordinateF(coordinate coordinate.Coordinate) Field {
	return Field{"coordinate", coordinate}
}

// EnvironmentF builds a Field containing environment information for structured logging
func EnvironmentF(environment string) Field {
	return Field{"environment", environment}
}