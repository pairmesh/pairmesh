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
	"context"
	"encoding/base64"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/pairmesh/pairmesh/errcode"
	"github.com/pairmesh/pairmesh/pkg/jwt"
	"github.com/pairmesh/pairmesh/portal/db"
	"github.com/pairmesh/pairmesh/portal/db/models"
	"github.com/pairmesh/pairmesh/protocol"
	"github.com/pairmesh/pairmesh/security"
	"github.com/pingcap/fn"

	"github.com/pkg/errors"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// PeerGraph respond the peer graph of the device which send the request.
func (s *server) PeerGraph(ctx context.Context) (*protocol.PeerGraphResponse, error) {
	userID := models.ID(jwt.UserIDFromContext(ctx))

	var (
		peers          []protocol.Peer
		relayServerIDs = map[models.ID]struct{}{}
		networks       []protocol.Network
	)

	err := db.Tx(func(tx *gorm.DB) error {
		// Update the last seen time
		_ = models.NewDeviceQuerySet(tx).
			UserIDEq(userID).
			MachineIDEq(jwt.MachineIDFromContext(ctx)).
			GetUpdater().
			SetLastSeen(time.Now()).
			Update()

		userDevices := map[models.ID][]models.ID{}

		devices, err := models.PeerDevices(tx, userID)
		if err != nil {
			return err
		}
		if len(devices) == 0 {
			// Retrieve all self devices
			var selfDevices []models.Device
			err := models.NewDeviceQuerySet(tx).UserIDEq(userID).All(&selfDevices)
			if err != nil {
				return err
			}
			devices = selfDevices
		}
		for _, d := range devices {
			peers = append(peers, protocol.Peer{
				ID:       protocol.PeerID(d.ID),
				UserID:   protocol.UserID(d.UserID),
				Name:     d.Name,
				IPv4:     d.Address,
				ServerID: protocol.ServerID(d.RelayServerID),
				Active:   d.LastSeen.After(time.Now().Add(-600 * time.Second)), // Last seen in 10 minutes.
			})

			relayServerIDs[d.RelayServerID] = struct{}{}
			userDevices[d.UserID] = append(userDevices[d.UserID], d.ID)
		}

		// Retrieve all networks
		var networkIDs []models.ID
		var userNetworks []models.NetworkUser
		err = models.NewNetworkUserQuerySet(tx).PreloadNetwork().UserIDEq(userID).All(&userNetworks)
		if err != nil {
			return err
		}

		var networsByID = map[models.ID]*protocol.Network{}
		for _, n := range userNetworks {
			networkIDs = append(networkIDs, n.NetworkID)
			networsByID[n.NetworkID] = &protocol.Network{
				ID:   protocol.NetworkID(n.ID),
				Name: n.Network.Name,
			}
		}

		if len(networkIDs) < 1 {
			return nil
		}

		// Retrieve all peers
		// TODO: use SQL to improve the performance
		var topology []models.NetworkUser
		err = models.NewNetworkUserQuerySet(tx).NetworkIDIn(networkIDs...).All(&topology)
		if err != nil {
			return err
		}
		for _, d := range topology {
			devices := userDevices[d.UserID]
			var peers []protocol.PeerID
			for _, d := range devices {
				peers = append(peers, protocol.PeerID(d))
			}
			networsByID[d.NetworkID].Peers = append(networsByID[d.NetworkID].Peers, peers...)
		}

		for _, n := range networsByID {
			networks = append(networks, *n)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	var relayServers []protocol.RelayServer
	for id := range relayServerIDs {
		v, ok := s.relayServers.byID.Load(id)
		if !ok {
			zap.L().Error("Relay server not found", zap.Any("relay_id", id))
			continue
		}
		relayServer := v.(*models.RelayServer)
		relayServers = append(relayServers, protocol.RelayServer{
			ID:        protocol.ServerID(relayServer.ID),
			Name:      relayServer.Name,
			Region:    relayServer.Region,
			Host:      relayServer.Host,
			Port:      relayServer.Port,
			STUNPort:  relayServer.STUNPort,
			PublicKey: relayServer.PublicKey,
		})
	}

	resp := &protocol.PeerGraphResponse{
		RelayServers: relayServers,
		Peers:        peers,
		Networks:     networks,
	}

	return resp, nil
}

func (s *server) randomRelayServerID() models.ID {
	// Assign a new relay server for it.
	var randomRelayID models.ID
	s.relayServers.byID.Range(func(key, value interface{}) bool {
		randomRelayID = value.(*models.RelayServer).ID
		return false
	})
	return randomRelayID
}

// Preflight returns the parameters for startup a node
func (s *server) Preflight(ctx context.Context, r *http.Request, req *protocol.PreflightRequest) (*protocol.PreflightResponse, error) {
	userID := models.ID(jwt.UserIDFromContext(ctx))
	machineID := jwt.MachineIDFromContext(ctx)

	device := &models.Device{}
	err := db.Tx(func(tx *gorm.DB) error {
		err := models.NewDeviceQuerySet(tx).
			UserIDEq(userID).
			MachineIDEq(machineID).
			One(device)
		if err != nil && err != gorm.ErrRecordNotFound {
			return err
		}

		deviceNotExists := err == gorm.ErrRecordNotFound
		// Insert device if
		if deviceNotExists {
			count, err := models.NewDeviceQuerySet(tx).UserIDEq(userID).Count()
			if err != nil && err != gorm.ErrRecordNotFound {
				return err
			}

			if count >= models.MaxOwnDevice {
				return errcode.ErrDeviceExceed
			}

			address, err := models.NextIP(tx)
			if err != nil {
				return err
			}

			device = &models.Device{
				UserID:        userID,
				RelayServerID: s.randomRelayServerID(),
				OS:            req.OS,
				Version:       versionFromContext(ctx),
				Name:          req.Host,
				MachineID:     machineID,
				LastSeen:      time.Now(),
				Address:       address,
			}

			return tx.Create(device).Error
		}

		// Update relay server if previous dead.
		_, found := s.relayServers.byID.Load(device.RelayServerID)
		if !found {
			updater := models.NewDeviceQuerySet(tx).IDEq(device.ID).GetUpdater()
			// Check if relay server alive?
			device.RelayServerID = s.randomRelayServerID()
			updater.SetRelayServerID(device.RelayServerID)
			return updater.SetLastSeen(time.Now()).Update()
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	val, found := s.relayServers.byID.Load(device.RelayServerID)
	if !found {
		return nil, errors.Errorf("relay server %d not found", device.RelayServerID)
	}
	relayServer := val.(*models.RelayServer)

	// Sign the node id  to prevent the client counterfeit.
	credential, err := security.Credential(s.privateKey, protocol.UserID(userID), protocol.PeerID(device.ID), net.ParseIP(device.Address), credentialLease)
	if err != nil {
		return nil, err
	}

	resp := &protocol.PreflightResponse{
		ID:     protocol.PeerID(device.ID),
		UserID: protocol.UserID(device.UserID),
		Name:   device.Name,
		IPv4:   device.Address,
		PrimaryServer: protocol.RelayServer{
			ID:        protocol.ServerID(relayServer.ID),
			Name:      relayServer.Name,
			Region:    relayServer.Region,
			Host:      relayServer.Host,
			Port:      relayServer.Port,
			STUNPort:  relayServer.STUNPort,
			PublicKey: relayServer.PublicKey,
		},
		Credential:      base64.RawStdEncoding.EncodeToString(credential),
		CredentialLease: uint64(credentialLease / time.Second),
	}

	return resp, nil
}

// RelayKeepalive handles the RelayKeepaliveRequest HTTP POST request
func (s *server) RelayKeepalive(req *protocol.RelayKeepaliveRequest) (*protocol.RelayKeepaliveResponse, error) {
	if req.STUNPort == 0 || req.Port == 0 || req.Host == "" {
		return nil, errcode.ErrIllegalRequest
	}

	v, ok := s.relayServers.byAddr.Load(fmt.Sprintf("%s:%d", req.Host, req.Port))
	// New relay servers
	if !ok {
		relayServer := &models.RelayServer{
			Name:        req.Name,
			Region:      req.Region,
			Host:        req.Host,
			Port:        req.Port,
			STUNPort:    req.STUNPort,
			PublicKey:   req.PublicKey,
			StartedAt:   time.Unix(req.StartedAt/int64(time.Second), req.StartedAt%int64(time.Second)),
			KeepaliveAt: time.Now(),
		}

		err := db.Tx(func(tx *gorm.DB) error { return tx.Create(relayServer).Error })
		if err != nil {
			return nil, err
		}

		s.relayServers.byID.Store(relayServer.ID, relayServer)
		s.relayServers.byAddr.Store(fmt.Sprintf("%s:%d", relayServer.Host, relayServer.Port), relayServer)
	} else {
		relayServer := v.(*models.RelayServer)
		relayServer.KeepaliveAt = time.Now()

		err := db.Tx(func(tx *gorm.DB) error {
			updater := models.NewRelayServerQuerySet(tx).
				IDEq(relayServer.ID).
				GetUpdater()
			updater.SetKeepaliveAt(relayServer.KeepaliveAt)

			// Changed fields
			if relayServer.PublicKey != req.PublicKey {
				relayServer.PublicKey = req.PublicKey
				updater.SetPublicKey(req.PublicKey)
			}

			if relayServer.STUNPort != req.STUNPort {
				relayServer.STUNPort = req.STUNPort
				updater.SetSTUNPort(req.STUNPort)
			}

			if relayServer.StartedAt.Unix() != req.StartedAt {
				relayServer.StartedAt = time.Unix(req.StartedAt/int64(time.Second), req.StartedAt%int64(time.Second))
				updater.SetStartedAt(relayServer.StartedAt)
			}

			return updater.Update()
		})
		if err != nil {
			return nil, err
		}
	}

	res := &protocol.RelayKeepaliveResponse{
		PublicKey: s.publicKey.base64,
	}

	return res, nil
}

// RenewCredential handles the `RenewCredentialRequest` POST request.
func (s *server) RenewCredential(req *protocol.RenewCredentialRequest) (*protocol.RenewCredentialResponse, error) {
	credential, err := base64.RawStdEncoding.DecodeString(req.Credential)
	if err != nil {
		return nil, err
	}
	userID, peerID, ip, valid := security.VerifyCredential(s.publicKey.raw, credential)
	if !valid {
		return nil, fmt.Errorf("invalid credential: %v", req.Credential)
	}

	newCredential, err := security.Credential(s.privateKey, userID, peerID, ip, credentialLease)
	if err != nil {
		return nil, err
	}

	res := &protocol.RenewCredentialResponse{
		Credential:      base64.RawStdEncoding.EncodeToString(newCredential),
		CredentialLease: uint64(credentialLease / time.Second),
	}
	return res, nil
}

type VersionCheckResponse struct {
	NewVersion      bool   `json:"new_version"`
	NewVersionCode  string `json:"new_version_code"`
	DownloadAddress string `json:"download_address"`
}

func (s *server) VersionCheck(form *fn.Form) (*VersionCheckResponse, error) {
	version := form.Get("version")
	platform := form.Get("platform")
	if version == "" || platform == "" {
		return nil, errcode.ErrIllegalRequest
	}
	//TODO
	return nil, nil
}
