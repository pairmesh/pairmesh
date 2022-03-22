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

package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/pairmesh/pairmesh/pkg/cmdutil"
	"github.com/pairmesh/pairmesh/pkg/logutil"
	"github.com/pairmesh/pairmesh/relay/config"
	"github.com/pairmesh/pairmesh/relay/server"
	"github.com/pairmesh/pairmesh/version"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

func main() {
	var (
		cfgPath  string
		examples = cmdutil.Examples{
			{
				Example: "pairrelay -c /path/to/config.yaml",
				Comment: "Specify the path of configuration file",
			},
			{
				Example: "pairrelay --version",
				Comment: "Print the version of meshrelay",
			},
		}
	)

	rootCmd := &cobra.Command{
		Use: fmt.Sprintf("pairrelay -c %s [flags]", cmdutil.Underline("<CONFIG>")),
		Long: fmt.Sprintf(`PairRelay will relay traffics when peer cannot communicate to each other directly.
And the relay server(s) is responsible to peer discovery. 

- The parameter '-c %[1]s' or '--config %[1]s' is required.
`,
			cmdutil.Underline("<CONFIG>")),
		Example:       examples.String(),
		Version:       version.NewVersion().String(),
		SilenceUsage:  true,
		SilenceErrors: true,
		PreRun: func(cmd *cobra.Command, args []string) {
			logutil.InitLogger()
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if cfgPath == "" {
				return cmd.Help()
			}

			cfg, err := config.FromPath(cfgPath)
			if err != nil {
				return err
			}

			ctx, cancel := context.WithCancel(context.Background())

			go func() {
				sc := make(chan os.Signal, 1)
				signal.Notify(sc, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
				sg := <-sc
				zap.L().Info("The relay is terminating due to signal", zap.Stringer("signal", sg))
				cancel()
			}()

			var wg sync.WaitGroup

			if err := server.Serve(ctx, &wg, cfg); err != nil {
				zap.L().Error("Serve relay server failed", zap.Error(err))
			}

			zap.L().Info("The relay server is shutdown gracefully")
			cancel()

			wg.Wait()
			zap.L().Info("See you again, bye!")

			return nil
		},
	}

	rootCmd.Flags().StringVarP(&cfgPath, "config", "c", "", "Specify the path of configuration file")
	cmdutil.Run(rootCmd)
}
