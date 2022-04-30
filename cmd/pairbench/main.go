// Copyright 2022 PairMesh, Inc.
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

package main

import (
	"errors"
	"fmt"
	"strings"

	"github.com/pairmesh/pairmesh/bench"
	"github.com/pairmesh/pairmesh/bench/client"
	"github.com/pairmesh/pairmesh/bench/config"
	"github.com/pairmesh/pairmesh/bench/server"
	"github.com/pairmesh/pairmesh/pkg/cmdutil"
	"github.com/pairmesh/pairmesh/pkg/logutil"
	"github.com/pairmesh/pairmesh/version"
	"github.com/spf13/cobra"
)

func main() {
	var (
		role        string
		mode        string
		port        uint16
		host        string
		isBounceStr string
		clients     uint16
		payload     uint16
		duration    uint16
		examples    = cmdutil.Examples{
			{
				Example: "pairbench -r server -p 9736",
				Comment: "Start pairbench in server role as an echo server, serving port 9736",
			},
			{
				Example: "pairbench -r client -e 100.68.80.110 -p 9736 -c 12 -l 42 -d 60",
				Comment: `Start pairbench in client role, connecting to echo server 100.68.80.110 port 9736, 
spawning 12 clients, each request contains 42 bytes, and test for 60 seconds.`,
			},
		}
	)
	rootCmd := &cobra.Command{
		Use: fmt.Sprintf("pairbench -r %s [flags]", cmdutil.Underline("<MODE>")),
		Long: fmt.Sprintf(`pairbench starts with server or client role.

- In server role, besides '-m %[1]s', additional parameters '-p %[2]s', '-b %[3]s' are optional.

- In client role, besides '-m %[4]s', additional parameters '-e %[5]s', '-p %[2]s', '-c %[6]s', '-l %[7]s', '-d %[8]s' are optional.
`,
			cmdutil.Underline("server"),
			cmdutil.Underline("<PORT>"),
			cmdutil.Underline("<BOUNCE>"),
			cmdutil.Underline("client"),
			cmdutil.Underline("ENDPOINT"),
			cmdutil.Underline("<CLIENTS>"),
			cmdutil.Underline("<PAYLOAD>"),
			cmdutil.Underline("<DURATION>"),
		),
		Example:       examples.String(),
		Version:       version.NewVersion().String(),
		SilenceUsage:  false,
		SilenceErrors: false,
		PreRun: func(cmd *cobra.Command, args []string) {
			logutil.InitLogger()
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			// Validate and convert mode
			mode = strings.ToLower(mode)
			if mode != "relay" && mode != "echo" {
				return errors.New("invalid mode param. Only relay or echo are valid")
			}
			modeType := bench.ModeType(mode)

			// Validate and convert isBounce
			var isBounce bool
			isBounceStr = strings.ToLower(isBounceStr)
			if isBounceStr == "true" {
				isBounce = true
			} else if isBounceStr == "false" {
				isBounce = false
			} else {
				return errors.New("invalid bounce param. Only true or false are valid")
			}

			// Validate and convert role
			role = strings.ToLower(role)
			switch {
			case role == "server":
				serverCfg := config.NewServerConfig(modeType, port, isBounce)
				return server.Run(&serverCfg)
			case role == "client":
				if clients > bench.MaxClientCount {
					return fmt.Errorf("invalid client number. Cannot be more than %d", bench.MaxClientCount)
				}

				if payload > bench.MaxPayload {
					return fmt.Errorf("invalid payload length. Cannot be more than %d", bench.MaxPayload)
				}

				clientCfg := config.NewClientConfig(
					modeType,
					host,
					port,
					isBounce,
					clients,
					payload,
					duration,
				)
				return client.Run(&clientCfg)
			default:
				return errors.New("invalid role. Only server or client are valid roles")
			}
		},
	}

	rootCmd.Flags().StringVarP(&role, "role", "r", "server", "Specify the role of pairbench, server or client")
	rootCmd.Flags().StringVarP(&mode, "mode", "m", "relay", "Specify the mode of pairbench, relay or echo")
	rootCmd.Flags().StringVarP(&host, "endpoint", "e", "127.0.0.1", "Specify the server endpoint when in client role")
	rootCmd.Flags().Uint16VarP(&port, "port", "p", 9736, "Specify the portal of the server")
	rootCmd.Flags().StringVarP(&isBounceStr, "bounce", "b", "true", "Specify whether server would echo back all data from clients. Otherwise simply echo back 'OK'")
	rootCmd.Flags().Uint16VarP(&clients, "client", "c", 1, fmt.Sprintf("Specify the number of clients when in client role, max %d", bench.MaxClientCount))
	rootCmd.Flags().Uint16VarP(&payload, "payload", "l", 128, fmt.Sprintf("Specify the payload in bytes, max %d", bench.MaxPayload))
	rootCmd.Flags().Uint16VarP(&duration, "duration", "d", 60, "Specify the test duration in seconds")
	cmdutil.Run(rootCmd)
}
