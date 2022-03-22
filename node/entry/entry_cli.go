//go:build linux

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

package entry

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"os/signal"
	"syscall"

	"github.com/pairmesh/pairmesh/constant"
	"github.com/pairmesh/pairmesh/node/api"
	"github.com/pairmesh/pairmesh/node/config"
	"github.com/pairmesh/pairmesh/node/device"
	"github.com/pairmesh/pairmesh/node/driver"
	"github.com/pairmesh/pairmesh/pkg/cmdutil"
	"github.com/pairmesh/pairmesh/pkg/logutil"
	"github.com/pairmesh/pairmesh/version"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

// Run startups the linux version of PairMesh
func Run() {
	var (
		authKey     string
		apiEndpoint string
		examples    = cmdutil.Examples{
			{
				Example: "pairmesh -k <AUTH_KEY>",
				Comment: "Start PairMesh with the specified auth key",
			},
			{
				Example: "pairmesh -a <API_ENDPOINT> -k <AUTH_KEY>",
				Comment: "Start PairMesh with customized api endpoint and specified auth key",
			},
			{
				Example: "pairmesh --version",
				Comment: "Print the version of PairMesh client",
			},
		}
	)

	rootCmd := &cobra.Command{
		Use: fmt.Sprintf("pairmesh -k %s [flags]", cmdutil.Underline("<AUTH_KEY>")),
		Long: fmt.Sprintf(`PairMesh is use to build a security layer-three local area network over WAN via
P2P communication. All traffic between peers will be transfer directly use P2P
network with encryption. Only the traffic from the peers which located behind
a symmetric NAT network will be relayed by STUN server. Both relay traffic and
P2P traffic are encrypted by %[1]s. Every tunnel has different secret key.

- Environment variable '%[2]s' is used to specify the api gateway address.
- Environment variable '%[3]s' is used to specify the portal web address.
- The pre-authentication parameter '-k %[4]s' or '--key %[4]s' is required.
- %[5]s required because %[6]s will create virtual network device.
- Environment variable '%[7]s' is used to specify the verbosity of debug log.`,
			cmdutil.Underline("chacha20-poly1305"),
			cmdutil.Underline("PAIRMESH_GATEWAY_API"),
			cmdutil.Underline("PAIRMESH_GATEWAY_MY"),
			cmdutil.Underline("<AUTH_KEY>"),
			cmdutil.Underline("Privileges"),
			cmdutil.Bold("PairMesh"),
			cmdutil.Underline(constant.EnvLogLevel)),
		Example:       examples.String(),
		Version:       version.NewVersion().FullInfo(),
		SilenceUsage:  true,
		SilenceErrors: true,
		PreRun: func(cmd *cobra.Command, args []string) {
			logutil.InitLogger()
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := &config.Config{}
			if err := cfg.Load(); err != nil {
				return err
			}

			zap.L().Info("Load configuration finished", zap.Reflect("config", cfg))

			// Overwrite the existing token
			if authKey != "" {
				cfg.Token = constant.PrefixAuthKey + " " + authKey
			}
			if cfg.IsGuest() {
				return fmt.Errorf("please use `-k <AUTH_KEY>` to specify the pre-authentication key")
			}

			// Normalize the gateway scheme
			gateway := config.APIGateway()
			if apiEndpoint != "" {
				gateway = apiEndpoint
			}
			u, err := url.Parse(gateway)
			if err != nil {
				return err
			}
			if u.Scheme != "http" && u.Scheme != "https" {
				return fmt.Errorf("unknown gateway scheme %s", u.Scheme)
			}
			if u.Path != "" {
				return fmt.Errorf("incorrect gateway address %s", gateway)
			}
			apiClient := api.New(gateway, cfg.Token, cfg.MachineID)

			err = exchangeAuthKeyIfNeed(apiClient, cfg)
			if err != nil {
				return errors.WithMessage(err, "exchange key failed")
			}

			dev, err := device.NewDevice()
			if err != nil {
				return err
			}

			drv := driver.New(cfg, dev, apiClient)
			defer drv.Terminate()

			if err = drv.Preflight(); err != nil {
				return err
			}

			ctx, cancel := context.WithCancel(context.Background())
			go drv.Drive(ctx)

			zap.L().Info("Driver initialized successfully")

			sc := make(chan os.Signal, 1)
			signal.Notify(sc, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

			sg := <-sc
			zap.L().Info("Got signal and prepare to terminate", zap.Stringer("signal", sg))

			cancel()
			drv.Terminate()

			zap.L().Info("Driver is terminated successfully")

			return nil
		},
	}

	rootCmd.Flags().StringVarP(&authKey, "key", "k", "", "The pre-authentication key of the node")
	rootCmd.Flags().StringVarP(&apiEndpoint, "api-endpoint", "a", "", "Specify the path of api endpoint")

	cmdutil.Run(rootCmd)
}
