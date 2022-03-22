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

package sso

import (
	"errors"
	"fmt"
	"math/rand"
	"strings"
	"sync"
	"time"

	"github.com/pairmesh/pairmesh/portal/config"
	"github.com/pairmesh/pairmesh/portal/db/models"
	"go.uber.org/zap"
)

// Vendor represents the current sso provider (github, ...)
type Vendor int

const (
	GitHub Vendor = iota
)

const URIAuthCodeCallback = "/login/auth/callback"

var errIllegalParam = errors.New("illegal parameter(s)")

var vendorName = map[Vendor]string{
	GitHub: "GitHub",
}

// String implements the fmt.Stringer interface
func (v Vendor) String() string {
	n, found := vendorName[v]
	if !found {
		return fmt.Sprintf("unknown(%d)", v)
	}
	return strings.ToLower(n)
}

type Token struct {
	AccessToken string
	Raw         map[string]string //extra data for diff platform
}

// Provider represents the sso vendor
type Provider interface {
	Setup(cfg *config.SSO) error
	AuthCodeURL(redirect, client string) string
	AccessToken(code string) (*Token, error)
	UserInfo(token *Token) (*models.User, bool, error)
}

type providerMgr struct {
	sync.RWMutex

	r         *rand.Rand
	providers map[Vendor]Provider
}

var gMgr = &providerMgr{
	r:         rand.New(rand.NewSource(time.Now().Unix())),
	providers: make(map[Vendor]Provider),
}

func (pm *providerMgr) init(cfg *config.SSO) error {
	for key, p := range pm.providers {
		zap.L().Info("register the cfg", zap.Any("cfg", vendorName[key]))
		if err := p.Setup(cfg); err != nil {
			return err
		}
	}
	return nil
}

func (pm *providerMgr) register(name Vendor, h Provider) error {
	if h == nil {
		return fmt.Errorf("the sso is nil")
	}

	pm.Lock()
	defer pm.Unlock()

	if _, ok := pm.providers[name]; ok {
		return nil
	}
	pm.providers[name] = h
	return nil
}

func (pm *providerMgr) provider(name Vendor) Provider {
	pm.RLock()
	defer pm.RUnlock()

	if p, ok := pm.providers[name]; ok {
		return p
	}

	return nil
}

func (pm *providerMgr) randString(len int) string {
	bytes := make([]byte, len)
	for i := 0; i < len; i++ {
		b := pm.r.Intn(26) + 65
		bytes[i] = byte(b)
	}
	return string(bytes)
}

// WithName returns the provider with the name
func WithName(name Vendor) Provider {
	return gMgr.provider(name)
}

// Register registers a provider with the name
func Register(name Vendor, p Provider) {
	err := gMgr.register(name, p)
	if err != nil {
		zap.L().Error("Error registering a provider")
	}
}

// Initialize  init the provider(s) successfully, if not, crash it
func Initialize(sso *config.SSO) error {
	if sso == nil {
		return errIllegalParam
	}
	return gMgr.init(sso)
}

// NextState generate a new oauth2's state code
func NextState() string {
	return gMgr.randString(8)
}

// Link represents a sso vendor name &　the auth code link pair
type Link struct {
	Name string
	Link string
}

// AuthCodeLinks generates the all sso vendor name &　the auth code link pairs
// If the node port equals `0`, which means the login request is not started
// from the PairMesh node.
func AuthCodeLinks(redirect string, client string) []Link {
	//sort for html render
	names := []Vendor{
		GitHub,
		// Other 3rd providers...
	}

	var ret []Link
	for _, v := range names {
		url := gMgr.provider(v).AuthCodeURL(redirect, client)
		ret = append(ret, Link{
			Name: vendorName[v],
			Link: url,
		})
	}

	return ret
}
