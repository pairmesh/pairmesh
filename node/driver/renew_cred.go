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

package driver

import (
	"context"
	"encoding/base64"
	"time"

	"go.uber.org/zap"
)

type credential struct {
	address string
	// The raw bytes representation of credential
	RawBytes []byte
	// The BASE64 representation of credential
	Base64 string
	// The renew credential time
	RenewedAt time.Time
	// The lease of credential
	Lease time.Duration
}

//nolint ; TODO: I suspect we forgot to call this function in some proper places
func (d *NodeDriver) renewCredential(ctx context.Context) {
	defer d.wg.Done()

	nextRenewTime := func() <-chan time.Time {
		// Next retry renew time
		expirationAt := d.credential.RenewedAt.Add(d.credential.Lease)
		if expirationAt.Before(time.Now()) {
			zap.L().Fatal("The credential is invalid due to expiration")
		}

		return time.After(time.Until(expirationAt) / 2)
	}

	renewCredTimer := nextRenewTime()
	for {
		select {
		case <-ctx.Done():
			zap.L().Info("Heartbeat and renew goroutine stopped")
			return

		case <-renewCredTimer:
			res, err := d.apiClient.RenewCredential(d.credential.Base64)
			if err != nil {
				zap.L().Error("Renew the credential failed", zap.Error(err))
				renewCredTimer = nextRenewTime()
				continue
			}

			// Parse the credential
			credential, err := base64.RawStdEncoding.DecodeString(res.Credential)
			if err != nil {
				zap.L().Error("Decode base64 credential failed", zap.Error(err), zap.String("credential", res.Credential))
				renewCredTimer = nextRenewTime()
				continue
			}
			d.credential.RawBytes = credential
			d.credential.Base64 = res.Credential
			d.credential.Lease = time.Duration(res.CredentialLease) * time.Second
			d.credential.RenewedAt = time.Now()
			d.rm.SetCredential(credential)
			renewCredTimer = nextRenewTime()
		}
	}
}
