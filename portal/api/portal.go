// Copyright 2020 PairMesh.com
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// See the License for the specific language governing permissions and
// limitations under the License.

package api

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"strings"
	"sync"

	"github.com/pairmesh/pairmesh/internal/ledis"

	// grouped for init
	"github.com/pairmesh/pairmesh/pkg/fsutil"
	"github.com/pairmesh/pairmesh/pkg/jwt"
	"github.com/pairmesh/pairmesh/portal/config"
	"github.com/pairmesh/pairmesh/portal/db"
	"github.com/pairmesh/pairmesh/portal/sso"

	// Need this anonymous import because we relay on the github.init() func to register sso provider.
	_ "github.com/pairmesh/pairmesh/portal/sso/github"

	"go.uber.org/zap"
)

func serveHTTP(cfg *config.Config) (*http.Server, error) {
	setupMiddleware()

	// Preflight the database
	var err error
	if err = sso.Initialize(cfg.SSO); err != nil {
		return nil, fmt.Errorf("initialize sso is failed: %w", err)
	}

	if err = db.Initialize(cfg.MySQL); err != nil {
		return nil, fmt.Errorf("initialize mysql is failed: %w", err)
	}

	err = ledis.Initialize(cfg.DataDir)
	if err != nil {
		return nil, fmt.Errorf("initialize redis is failed: %w", err)
	}

	// Preflight the jwt
	err = jwt.Initialize(
		cfg.JWT.AccessSecret,
		cfg.JWT.RefreshSecret,
		jwt.AccessTokenTTL(cfg.JWT.AccessTokenTTL),
		jwt.RefreshTokenTTL(cfg.JWT.RefreshTokenTTL),
		jwt.WithStore(ledis.Shared()))
	if err != nil {
		return nil, fmt.Errorf("initialize jwt is failed: %w", err)
	}

	// Load the private key
	var key *rsa.PrivateKey
	if cfg.PrivateKey == "" {
		path := "pairportal.private.pem"
		if fsutil.IsExists(path) {
			zap.L().Info("Private privateKey path is not specified, but found in default path", zap.String("path", path))
			data, err := ioutil.ReadFile(path)
			if err != nil {
				return nil, fmt.Errorf("read private key is failed: %w", err)
			}
			key, err = x509.ParsePKCS1PrivateKey(data)
			if err != nil {
				return nil, fmt.Errorf("parse private key is failed: %w", err)
			}

		} else {
			zap.L().Info("Private privateKey path is not specified, generated automatically", zap.String("path", path))
			priv, err := rsa.GenerateKey(rand.Reader, 512)
			if err != nil {
				return nil, fmt.Errorf("generate private key is failed: %w", err)
			}
			data := x509.MarshalPKCS1PrivateKey(priv)
			err = ioutil.WriteFile(path, data, 0600)
			if err != nil {
				return nil, fmt.Errorf("save private key is failed: %w", err)
			}
			key = priv
		}
		cfg.PrivateKey = path
	}

	// Trim sso redirect so that tailing "/" will be removed
	redirect := strings.TrimRight(cfg.SSO.Redirect, "/")

	var (
		server    = newServer(cfg.Relay.AuthKey, key)
		ssoServer = newSSOServer(redirect)

		mux     = route(server, ssoServer)
		address = fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
	)

	if err := server.preload(); err != nil {
		return nil, fmt.Errorf("preload data is failed: %w", err)
	}

	srv := &http.Server{
		Handler: mux,
	}

	go func() {
		zap.L().Info("Listening on:	" + address)
		l, err := net.Listen("tcp", address)
		if err != nil {
			zap.L().Fatal("Listen local HTTP server failed", zap.Error(err))
			return
		}

		if cfg.TLSCert == "" {
			err = srv.Serve(l)
		} else {
			err = srv.ServeTLS(l, cfg.TLSCert, cfg.TLSKey)
		}

		if err == nil || err == http.ErrServerClosed {
			zap.L().Info("The web server is over")
			return
		}
		zap.L().Error("The web server is over", zap.Error(err))
	}()

	return srv, nil
}

// Serve serves the gateway service
func Serve(ctx context.Context, wg *sync.WaitGroup, cfg *config.Config) {
	defer wg.Done()

	srv, err := serveHTTP(cfg)
	if err != nil {
		zap.L().Error("Serve http is failed", zap.Error(err))
		return
	}

	// Wait for server shutdown.
	<-ctx.Done()
	ctx, cancel := context.WithTimeout(context.Background(), 5e9)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		zap.L().Error("Shutdown server failed", zap.Error(err))
	}

	zap.L().Info("The web server is shutdown gracefully")
}
