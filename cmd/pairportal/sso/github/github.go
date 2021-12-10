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

package github

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"

	"github.com/pairmesh/pairmesh/cmd/pairportal/config"
	"github.com/pairmesh/pairmesh/cmd/pairportal/db"
	"github.com/pairmesh/pairmesh/cmd/pairportal/db/models"
	"github.com/pairmesh/pairmesh/cmd/pairportal/db/queryutil"
	"github.com/pairmesh/pairmesh/cmd/pairportal/sso"

	"github.com/pkg/errors"
	"go.uber.org/zap"
	"golang.org/x/oauth2"
	stdgh "golang.org/x/oauth2/github"
	"gorm.io/gorm"
)

const apiGitHubUser = "https://api.github.com/user"

type github struct {
	conf   *oauth2.Config
	client *http.Client
}

// Setup initials the authentication provider
// https://docs.github.com/en/developers/apps/scopes-for-oauth-apps
func (gh *github) Setup(cfg *config.SSO) error {
	gh.conf = &oauth2.Config{
		ClientID:     cfg.GitHub.ClientID,
		ClientSecret: cfg.GitHub.ClientSecret,
		Scopes:       []string{"read:user", "user:email"},
		Endpoint:     stdgh.Endpoint,
	}

	// Overwrite HTTP Proxy
	if proxy, found := os.LookupEnv("https_proxy"); found && proxy != "" {
		u, err := url.Parse(proxy)
		if err == nil && u == nil {
			gh.client = &http.Client{Transport: &http.Transport{Proxy: http.ProxyURL(u)}}
		}
	}

	return nil
}

// AuthCodeURL returns a URL to OAuth 2.0 provider's consent page
// that asks for permissions for the required scopes explicitly.
func (gh *github) AuthCodeURL(redirect string, node string) string {
	var redirectURL string
	redirectURL = fmt.Sprintf("%s%s?p=%d", redirect, sso.URIAuthCodeCallback, sso.GitHub)
	if node != "" {
		redirectURL += fmt.Sprintf("&node=%s", node)
	}
	gh.conf.RedirectURL = redirectURL
	authCodeURL := gh.conf.AuthCodeURL(sso.NextState())
	return authCodeURL
}

func (gh *github) UserInfo(token *sso.Token) (*models.User, bool, error) {
	req, _ := http.NewRequest("GET", apiGitHubUser, nil)
	req.Header.Add("Authorization", "token "+token.AccessToken)
	req.Header.Add("Content-Type", "application/json")

	resp, err := gh.client.Do(req)
	if err != nil {
		zap.L().Error("read the user's info failed",
			zap.Any("sso", sso.GitHub),
			zap.Error(err))
		return nil, false, err
	}

	defer resp.Body.Close()

	type GithubUser struct {
		ID        int64  `json:"id"`
		Login     string `json:"login"`
		AvatarURL string `json:"avatar_url" `
		Location  string `json:"location"`
		Email     string `json:"email"`
	}

	var ghUser GithubUser
	err = json.NewDecoder(resp.Body).Decode(&ghUser)
	if err != nil {
		zap.L().Error("unmarshal user info failed",
			zap.Any("sso", sso.GitHub),
			zap.Error(err))
		return nil, false, err
	}

	var (
		user    models.User
		newUser bool
	)

	err = db.Tx(func(tx *gorm.DB) error {
		var githubUser models.GithubUser
		err = models.NewGithubUserQuerySet(tx).
			GithubIDEq(models.ID(ghUser.ID)).
			One(&githubUser)
		if err != nil && err != gorm.ErrRecordNotFound {
			return errors.WithStack(err)
		}

		// Create new user if not existing.
		newUser = err == gorm.ErrRecordNotFound
		if newUser {
			user = queryutil.BuildUser()
			user.Origin = sso.GitHub.String()
			user.Name = ghUser.Login
			user.Avatar = ghUser.AvatarURL
			user.Email = ghUser.Email

			ssoUser := &models.GithubUser{
				GithubID:  models.ID(ghUser.ID),
				Login:     ghUser.Login,
				AvatarURL: ghUser.AvatarURL,
				Location:  ghUser.Location,
			}
			return queryutil.CreateUser(&user, ssoUser)
		}

		// Update user information if user exists.
		err := models.NewUserQuerySet(tx).
			IDEq(githubUser.UserID).
			One(&user)
		if err != nil {
			return err
		}

		// Check if some information changed.
		changed := (user.Name != ghUser.Login) ||
			(user.Avatar != ghUser.AvatarURL)
		if !changed {
			return nil
		}

		updater := models.NewUserQuerySet(tx).
			IDEq(githubUser.UserID).
			GetUpdater()
		if user.Name != ghUser.Login {
			updater.SetName(ghUser.Login)
		}
		if user.Avatar != ghUser.AvatarURL {
			updater.SetAvatar(ghUser.AvatarURL)
		}

		return updater.Update()
	})

	return &user, newUser, err
}

func (gh *github) AccessToken(code string) (*sso.Token, error) {
	ctx := context.Background()
	token, err := gh.conf.Exchange(ctx, code)
	if err != nil {
		return nil, err
	}
	ssoToken := &sso.Token{
		AccessToken: token.AccessToken,
	}
	return ssoToken, nil
}

func init() {
	gh := github{
		client: &http.Client{},
	}
	sso.Register(sso.GitHub, &gh)
}
