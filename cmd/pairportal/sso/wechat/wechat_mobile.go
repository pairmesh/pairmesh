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

package wechat

import (
	"errors"
	"net/http"

	"github.com/pairmesh/pairmesh/cmd/pairportal/config"
	"github.com/pairmesh/pairmesh/cmd/pairportal/db/models"
	"github.com/pairmesh/pairmesh/cmd/pairportal/sso"

	"golang.org/x/oauth2"
)

type wechatMobile struct {
	conf   *oauth2.Config
	client *http.Client
}

func (wx *wechatMobile) Setup(cfg *config.SSO) error {
	wx.conf = &oauth2.Config{
		ClientID:     cfg.WeChatMobile.ClientID,
		ClientSecret: cfg.WeChatMobile.ClientSecret,
		Scopes:       []string{"snsapi_userinfo"},
		Endpoint: oauth2.Endpoint{
			TokenURL: apiWeChatToken,
		},
	}
	return nil
}
func (wx *wechatMobile) AuthCodeURL(redirect string, node string) string {
	return ""
}

func (wx *wechatMobile) UserInfo(token *sso.Token) (*models.User, bool, error) {
	openId, ok := token.Raw[openid]
	if !ok {
		return nil, false, errors.New("openid is null")
	}
	url := apiWeChatUser + "?access_token=" + token.AccessToken + "&openid=" + openId
	return wechatUserInfo(wx.client, url)
}

func (wx *wechatMobile) AccessToken(code string) (*sso.Token, error) {
	url := wx.conf.Endpoint.TokenURL + "?appid=" + wx.conf.ClientID + "&secret=" + wx.conf.ClientSecret + "&code=" + code + "&grant_type=authorization_code"
	return wechatAccessToken(wx.client, url)
}

func init() {
	wx := wechatMobile{
		client: &http.Client{},
	}
	sso.Register(sso.WeChatMobile, &wx)
}
