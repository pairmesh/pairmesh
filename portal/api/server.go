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

package api

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/NYTimes/gziphandler"
	"github.com/gorilla/mux"
	"github.com/pairmesh/pairmesh/constant"
	"github.com/pairmesh/pairmesh/portal/db"
	"github.com/pairmesh/pairmesh/portal/db/models"
	"github.com/pingcap/fn"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

const credentialLease = time.Hour

type (
	publicKey struct {
		// BASE64 representation of the public key, which is used to cache the
		// base64 encoded raw data to avoid encoding every relayServer keepalive
		// request.
		base64 string
		raw    *rsa.PublicKey
	}

	relayServers struct {
		byAddr sync.Map
		byID   sync.Map
	}

	// server represents the HTTP server which serves for the current PairMesh portal.
	server struct {
		// relayAuthKey is used to authenticate the relayServer servers requests.
		relayAuthKey string

		// privateKey is used to sign the credential which is used to identify
		// the node hold the IP address of specified network id.
		// TODO: private key rotate
		privateKey *rsa.PrivateKey
		publicKey  publicKey

		// ServerID -> models.RelayServer
		relayServers relayServers
	}
)

// newServer returns a new gateway server instance and the gateway server is
// used to handle the HTTP requests/UDP packets and store the peer information.
func newServer(relayAuthKey string, privateKey *rsa.PrivateKey) *server {
	srv := &server{
		relayAuthKey: relayAuthKey,
		privateKey:   privateKey,
		publicKey: publicKey{
			base64: base64.RawStdEncoding.EncodeToString(x509.MarshalPKCS1PublicKey(&privateKey.PublicKey)),
			raw:    &privateKey.PublicKey,
		},
	}
	return srv

}

// preload the data that persistent in the database.
func (s *server) preload() error {
	// Load the relay servers from database.
	zap.L().Info("Preload the relay servers from database")

	var relayServers []models.RelayServer
	err := db.Tx(func(tx *gorm.DB) error {
		return models.NewRelayServerQuerySet(tx).All(&relayServers)
	})
	if err != nil {
		zap.L().Error("Preload relay servers node is failed", zap.Error(err))
		return err
	}

	for _, r := range relayServers {
		s.relayServers.byID.Store(r.ID, &r)
		s.relayServers.byAddr.Store(fmt.Sprintf("%s:%d", r.Host, r.Port), &r)
	}

	return nil
}

// routers returns the route which routes all HTTP API requests
func route(server *server, ssoSrv *ssoServer) http.Handler {
	// Preflight the HTTP service and register all APIs
	router := mux.NewRouter()

	// All APIs for SSO Login/out: `/api/v1/login`
	router.Handle("/api/v1/version/check", fn.Wrap(server.VersionCheck)).Methods(http.MethodGet)
	router.Handle("/api/v1/login/sso-methods", fn.Wrap(ssoSrv.SSOMethods)).Methods(http.MethodGet)
	router.Handle("/api/v1/login/auth/callback/github", fn.Wrap(ssoSrv.GithubAuthCallback)).Methods(http.MethodPost)
	router.Handle(constant.URILogout, http.HandlerFunc(ssoSrv.Logout)).Methods(http.MethodGet)

	// All HTTP APIs requested by the relayServer servers
	relayAPI := fn.NewGroup().Plugin(relayAuthKeyValidator(server.relayAuthKey))
	router.Handle(constant.URIRelay, relayAPI.Wrap(server.RelayKeepalive)).Methods(http.MethodPost)

	// All HTTP APIs requested by the PairMesh peers
	peerAPI := fn.NewGroup().Plugin(peerTokenValidator)
	router.Handle(constant.URIDevicePeerGraph, peerAPI.Wrap(server.PeerGraph)).Methods(http.MethodGet)
	router.Handle(constant.URIDevicePreflight, peerAPI.Wrap(server.Preflight)).Methods(http.MethodPost)
	router.Handle(constant.URIRenewCredential, peerAPI.Wrap(server.RenewCredential)).Methods(http.MethodPost)
	router.Handle(constant.URLKeyExchange, peerAPI.Wrap(server.ExchangeKey)).Methods(http.MethodPost)

	// All HTTP APIs authed by jwt or auth key
	httpAPI := fn.NewGroup().Plugin(tokenValidator)
	router.Handle("/api/v1/settings/user/profile", httpAPI.Wrap(server.UserProfileSetting)).Methods(http.MethodPut)
	router.Handle("/api/v1/keys", httpAPI.Wrap(server.KeyList)).Methods(http.MethodGet)
	router.Handle("/api/v1/key", httpAPI.Wrap(server.CreateKey)).Methods(http.MethodPost)
	router.Handle("/api/v1/key/{key_id}", httpAPI.Wrap(server.ChangeKey)).Methods(http.MethodPut)
	router.Handle("/api/v1/key/{key_id}", httpAPI.Wrap(server.DeleteKey)).Methods(http.MethodDelete)
	router.Handle("/api/v1/user/profile", httpAPI.Wrap(server.UserProfile)).Methods(http.MethodGet)
	router.Handle("/api/v1/user/{user_id}/devices", httpAPI.Wrap(server.UserDeviceList)).Methods(http.MethodGet)
	router.Handle("/api/v1/devices", httpAPI.Wrap(server.DeviceList)).Methods(http.MethodGet)
	router.Handle("/api/v1/device/{device_id}", httpAPI.Wrap(server.DeviceUpdate)).Methods(http.MethodPut)
	router.Handle("/api/v1/networks", httpAPI.Wrap(server.NetworkList)).Methods(http.MethodGet)
	router.Handle("/api/v1/network", httpAPI.Wrap(server.CreateNetwork)).Methods(http.MethodPost)
	router.Handle("/api/v1/network/{network_id}", httpAPI.Wrap(server.UpdateNetwork)).Methods(http.MethodPut)
	router.Handle("/api/v1/network/{network_id}", httpAPI.Wrap(server.DeleteNetwork)).Methods(http.MethodDelete)
	router.Handle("/api/v1/network/{network_id}/members", httpAPI.Wrap(server.NetworkMembers)).Methods(http.MethodGet)
	router.Handle("/api/v1/network/{network_id}/member/invite", httpAPI.Wrap(server.InviteMember)).Methods(http.MethodPost)
	router.Handle("/api/v1/network/{network_id}/member/{user_id}", httpAPI.Wrap(server.DeleteNetworkUser)).Methods(http.MethodDelete)
	router.Handle("/api/v1/network/{network_id}/member/{user_id}/role", httpAPI.Wrap(server.ChangeNetworkMemberRole)).Methods(http.MethodPut)
	router.Handle("/api/v1/invitations", httpAPI.Wrap(server.Invitations)).Methods(http.MethodGet)
	router.Handle("/api/v1/invitation/{invitation_id}", httpAPI.Wrap(server.HandleInvitation)).Methods(http.MethodPut)

	return gziphandler.GzipHandler(router)
}
