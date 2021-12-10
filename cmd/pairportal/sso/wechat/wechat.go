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
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/pairmesh/pairmesh/cmd/pairportal/config"
	"github.com/pairmesh/pairmesh/cmd/pairportal/db"
	"github.com/pairmesh/pairmesh/cmd/pairportal/db/models"
	"github.com/pairmesh/pairmesh/cmd/pairportal/db/queryutil"
	"github.com/pairmesh/pairmesh/cmd/pairportal/sso"

	"github.com/pkg/errors"
	"go.uber.org/zap"
	"golang.org/x/oauth2"
	"gorm.io/gorm"
)

const apiWeChatUser = "https://api.weixin.qq.com/sns/userinfo"
const apiWeChatToken = "https://api.weixin.qq.com/sns/oauth2/access_token"

const openid = "openid"

type wechatToken struct {
	AccessToken  string `json:"access_token"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
	Openid       string `json:"openid"`
	Scope        string `json:"scope"`
}

type wechatUser struct {
	Nickname   string `json:"nickname"`
	HeadImgUrl string `json:"headimgurl"`
	Openid     string `json:"openid"`
	UnionId    string `json:"unionid"`
	City       string `json:"city"`
}

type wechat struct {
	conf   *oauth2.Config
	client *http.Client
}

// Setup initials the wechat authentication provider
// https://docs.github.com/en/developers/apps/scopes-for-oauth-apps
func (wx *wechat) Setup(cfg *config.SSO) error {
	wx.conf = &oauth2.Config{
		ClientID:     cfg.WeChat.ClientID,
		ClientSecret: cfg.WeChat.ClientSecret,
		Scopes:       []string{"snsapi_login"},
		Endpoint: oauth2.Endpoint{
			AuthURL:  "https://open.weixin.qq.com/connect/qrconnect",
			TokenURL: apiWeChatToken,
		},
	}
	return nil
}
func (wx *wechat) AuthCodeURL(redirect string, node string) string {
	var redirectURL string
	redirectURL = fmt.Sprintf("%s%s?p=%d", redirect, sso.URIAuthCodeCallback, sso.WeChat)

	if node != "" {
		redirectURL += fmt.Sprintf("&node=%s", node)
	}
	wx.conf.RedirectURL = redirectURL
	authCodeURL := wx.conf.AuthCodeURL(sso.NextState(), oauth2.SetAuthURLParam("appid", wx.conf.ClientID))
	return authCodeURL
}

func wechatUserInfo(client *http.Client, url string) (*models.User, bool, error) {
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Add("Content-Type", "application/json; charset=utf-8")

	resp, err := client.Do(req)
	if err != nil {
		zap.L().Error("read the user's info failed",
			zap.Any("sso", sso.WeChat),
			zap.Error(err))
		return nil, false, err
	}

	defer resp.Body.Close()

	var wxUser wechatUser
	err = json.NewDecoder(resp.Body).Decode(&wxUser)
	if err != nil {
		zap.L().Error("unmarshal user info failed",
			zap.Any("sso", sso.WeChat),
			zap.Error(err))
		return nil, false, err
	}

	var (
		user    models.User
		newUser bool
	)

	err = db.Tx(func(tx *gorm.DB) error {
		var wUser models.WechatUser
		err = models.NewWechatUserQuerySet(tx).
			UnionIdEq(wxUser.UnionId).
			One(&wUser)
		if err != nil && err != gorm.ErrRecordNotFound {
			return errors.WithStack(err)
		}

		newUser = err == gorm.ErrRecordNotFound
		if newUser {
			user = queryutil.BuildUser()
			user.Origin = sso.WeChat.String()
			user.Name = wxUser.Nickname
			user.Avatar = wxUser.HeadImgUrl

			ssoUser := &models.WechatUser{
				UnionId:    wxUser.UnionId,
				Nickname:   wxUser.Nickname,
				HeadImgUrl: wxUser.HeadImgUrl,
				City:       wxUser.City,
			}
			err = queryutil.CreateUser(&user, ssoUser)
		} else {
			err = models.NewUserQuerySet(tx).
				IDEq(wUser.UserID).
				One(&user)
		}

		return errors.WithStack(err)
	})

	return &user, newUser, err
}

func (wx *wechat) UserInfo(token *sso.Token) (*models.User, bool, error) {
	openId, ok := token.Raw[openid]
	if !ok {
		return nil, false, errors.New("openid is null")
	}
	url := apiWeChatUser + "?access_token=" + token.AccessToken + "&openid=" + openId
	return wechatUserInfo(wx.client, url)
}

func wechatAccessToken(client *http.Client, url string) (*sso.Token, error) {
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Add("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		zap.L().Error("get token failed",
			zap.Any("sso", sso.WeChat),
			zap.Error(err))
		return nil, err
	}

	defer resp.Body.Close()

	var token wechatToken
	err = json.NewDecoder(resp.Body).Decode(&token)
	if err != nil {
		zap.L().Error("unmarshal token failed",
			zap.Any("sso", sso.WeChat),
			zap.Error(err))
		return nil, err
	}
	ssoToken := &sso.Token{
		AccessToken: token.AccessToken,
		Raw: map[string]string{
			openid: token.Openid,
		},
	}
	return ssoToken, nil
}

func (wx *wechat) AccessToken(code string) (*sso.Token, error) {
	url := wx.conf.Endpoint.TokenURL + "?appid=" + wx.conf.ClientID + "&secret=" + wx.conf.ClientSecret + "&code=" + code + "&grant_type=authorization_code"
	return wechatAccessToken(wx.client, url)
}

func init() {
	wx := wechat{
		client: &http.Client{},
	}
	sso.Register(sso.WeChat, &wx)
}
