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
	"math/rand"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/pairmesh/pairmesh/pkg/cmdutil"
	"github.com/pairmesh/pairmesh/pkg/logutil"
	"github.com/pairmesh/pairmesh/portal/api"
	"github.com/pairmesh/pairmesh/portal/config"
	"github.com/pairmesh/pairmesh/version"

	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

func main() {
	var (
		cfgPath  string
		examples = cmdutil.Examples{
			{
				Example: "pairportal -c /path/to/config.yaml",
				Comment: "Specify the path of configuration file",
			},
			{
				Example: "pairportal --version",
				Comment: "Print the version of pairportal",
			},
		}
	)

	rootCmd := &cobra.Command{
		Use: fmt.Sprintf("pairportal -c %s [flags]", cmdutil.Underline("<CONFIG>")),
		Long: fmt.Sprintf(`PairMesh portal is the web console of the mesh. the relay server(s)
will register to it and it will handle the all HTTP APIs.

The parameter '-c %[1]s' or '--config %[1]s' is required. 
But fields values in the configuration are optinal.`,
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
			zap.L().Info("Setup virtual devices successfully")

			sc := make(chan os.Signal, 1)
			signal.Notify(sc, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

			var wg sync.WaitGroup
			wg.Add(1)
			go api.Serve(ctx, &wg, cfg)

			sg := <-sc

			zap.L().Info("The portal is terminating due to signal", zap.Stringer("signal", sg))
			cancel()
			wg.Wait()
			zap.L().Info("See you again, bye!")
			return nil
		},
	}

	rootCmd.Flags().StringVarP(&cfgPath, "config", "c", "", "Specify the path of configuration file")

	cmdutil.Run(rootCmd)
}
