// Copyright 2021 PairMesh, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package i18n

import (
	"embed"
	"fmt"
	"strings"

	"github.com/jeremywohl/flatten"
	"gopkg.in/yaml.v3"
)

const defaultLocale = "zh_CN"

//go:embed locales/*.yaml
var locales embed.FS

type Locale struct {
	name string
	data map[string]string
}

var localesData = map[string]*Locale{}

var currentLocale *Locale

func init() {
	if err := loadLocales(); err != nil {
		panic(err)
	}

	if err := SetLocale(defaultLocale); err != nil {
		panic(err)
	}
}

func loadLocales() error {
	files, err := locales.ReadDir("locales")
	if err != nil {
		return err
	}

	for _, f := range files {
		data, err := locales.ReadFile("locales/" + f.Name())
		if err != nil {
			return err
		}

		m := map[string]interface{}{}
		err = yaml.Unmarshal(data, &m)
		if err != nil {
			return err
		}

		r, err := flatten.Flatten(m, "", flatten.DotStyle)
		if err != nil {
			return err
		}

		// Convert the locales, which must be (string -> string)
		c := map[string]string{}
		for k, v := range r {
			x, ok := v.(string)
			if !ok {
				return fmt.Errorf("invalid value %v of key %s", k, v)
			}
			c[k] = x
		}

		name := strings.TrimSuffix(f.Name(), ".yaml")
		locale := &Locale{
			name: name,
			data: c,
		}
		localesData[locale.name] = locale
	}

	return nil
}

// Locales returns the locales list.
func Locales() []string {
	var results []string
	for _, l := range localesData {
		results = append(results, l.name)
	}
	return results
}

// SetLocale sets the current locale file.
func SetLocale(name string) error {
	l, found := localesData[name]
	if !found {
		return fmt.Errorf("locale configuration file %s not found", name)
	}
	currentLocale = l
	return nil
}

// L returns the i18n locale string corresponding to the specified key and args.
func L(key string, args ...interface{}) string {
	v, found := currentLocale.data[key]
	if !found {
		return fmt.Sprintf("%s:NOT_DEFINE(%s)", currentLocale.name, key)
	}
	return fmt.Sprintf(v, args...)
}
