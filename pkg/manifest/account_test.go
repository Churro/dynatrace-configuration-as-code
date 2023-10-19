//go:build unit

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

package manifest

import (
	"encoding/json"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/manifest/internal/persistence"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestValidAccounts(t *testing.T) {
	t.Setenv("SECRET", "secret")
	acc := persistence.Account{
		Name:        "name",
		AccountUUID: uuid.New().String(),
		ApiUrl: &persistence.Url{
			Value: "https://example.com",
		},
		OAuth: persistence.OAuth{
			ClientID: persistence.AuthSecret{
				Name: "SECRET",
			},
			ClientSecret: persistence.AuthSecret{
				Name: "SECRET",
			},
			TokenEndpoint: &persistence.Url{
				Value: "https://example.com",
			},
		},
	}

	// account 2 has no api name
	acc2 := persistence.Account{
		Name:        "name2",
		AccountUUID: uuid.New().String(),
		OAuth: persistence.OAuth{
			ClientID: persistence.AuthSecret{
				Name: "SECRET",
			},
			ClientSecret: persistence.AuthSecret{
				Name: "SECRET",
			},
			TokenEndpoint: nil,
		},
	}

	v, err := parseAccounts(&LoaderContext{}, []persistence.Account{acc, acc2})
	assert.NoError(t, err)

	assert.Equal(t, v, map[string]Account{
		"name": {
			Name:        "name",
			AccountUUID: uuid.MustParse(acc.AccountUUID),
			ApiUrl: &URLDefinition{
				Type:  ValueURLType,
				Value: "https://example.com",
			},
			OAuth: OAuth{
				ClientID:     AuthSecret{Name: "SECRET", Value: "secret"},
				ClientSecret: AuthSecret{Name: "SECRET", Value: "secret"},
				TokenEndpoint: &URLDefinition{
					Type:  ValueURLType,
					Value: "https://example.com",
				},
			},
		},
		"name2": {
			Name:        "name2",
			AccountUUID: uuid.MustParse(acc2.AccountUUID),
			ApiUrl:      nil,
			OAuth: OAuth{
				ClientID:      AuthSecret{Name: "SECRET", Value: "secret"},
				ClientSecret:  AuthSecret{Name: "SECRET", Value: "secret"},
				TokenEndpoint: nil,
			},
		},
	})

}

func TestInvalidAccounts(t *testing.T) {
	t.Setenv("SECRET", "secret")

	// default account to permute
	validAccount := persistence.Account{
		Name:        "name",
		AccountUUID: uuid.New().String(),
		ApiUrl: &persistence.Url{
			Value: "https://example.com",
		},
		OAuth: persistence.OAuth{
			ClientID: persistence.AuthSecret{
				Name: "SECRET",
			},
			ClientSecret: persistence.AuthSecret{
				Name: "SECRET",
			},
			TokenEndpoint: &persistence.Url{
				Value: "https://example.com",
			},
		},
	}

	// validate that the default is valid
	_, err := parseAccounts(&LoaderContext{}, []persistence.Account{validAccount})
	assert.NoError(t, err)

	// tests
	t.Run("name is missing", func(t *testing.T) {
		a := validAccount
		a.Name = ""

		_, err := parseAccounts(&LoaderContext{}, []persistence.Account{a})
		assert.ErrorIs(t, err, errNameMissing)
	})

	t.Run("accountUUID is missing", func(t *testing.T) {
		a := validAccount
		a.AccountUUID = ""

		_, err := parseAccounts(&LoaderContext{}, []persistence.Account{a})
		assert.ErrorIs(t, err, errAccUidMissing)
	})

	t.Run("accountUUID is invalid", func(t *testing.T) {
		a := deepCopy(t, validAccount)
		a.AccountUUID = "this-is-not-a-valid-uuid"

		_, err := parseAccounts(&LoaderContext{}, []persistence.Account{a})
		uuidErr := invalidUUIDError{}
		if assert.ErrorAs(t, err, &uuidErr) {
			assert.Equal(t, uuidErr.uuid, "this-is-not-a-valid-uuid")
		}
	})

	t.Run("oAuth is set", func(t *testing.T) {
		a := deepCopy(t, validAccount)
		a.OAuth = persistence.OAuth{}

		_, err := parseAccounts(&LoaderContext{}, []persistence.Account{a})
		assert.ErrorContains(t, err, "oAuth is invalid")
	})

	t.Run("oAuth.id is not set", func(t *testing.T) {
		a := deepCopy(t, validAccount)
		a.OAuth.ClientID = persistence.AuthSecret{}

		_, err := parseAccounts(&LoaderContext{}, []persistence.Account{a})
		assert.ErrorContains(t, err, "ClientID: no name given or empty")

	})

	t.Run("oAuth.secret is not set", func(t *testing.T) {
		a := deepCopy(t, validAccount)
		a.OAuth.ClientSecret = persistence.AuthSecret{}

		_, err := parseAccounts(&LoaderContext{}, []persistence.Account{a})
		assert.ErrorContains(t, err, "ClientSecret: no name given or empty")
	})
}

// deepCopy marshals and then marshals the payload, thus only works for public members, thus only private spaced
func deepCopy(t *testing.T, in persistence.Account) persistence.Account {
	d, e := json.Marshal(in)
	assert.NoError(t, e)

	var o persistence.Account
	e = json.Unmarshal(d, &o)
	assert.NoError(t, e)
	return o
}