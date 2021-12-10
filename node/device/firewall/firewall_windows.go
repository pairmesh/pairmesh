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

package firewall

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"
	"time"

	"go.uber.org/zap"
)

const (
	firewallNameIn      = "PairMesh-In"
	firewallNameProcess = "PairMesh-Process"
)

func Setup(addr string) {
	// Clear all firewall rules
	begin := time.Now()

	// Clear previous firewall rules.
	clear()

	defer func() {
		zap.L().Info("Setup the firewall rules", zap.Duration("duration", time.Since(begin)))
	}()

	exe, err := os.Executable()
	if err != nil {
		zap.L().Error("Cannot retrieve the executable", zap.Error(err))
	}

	processRule := []string{
		"advfirewall",
		"firewall",
		"add",
		"rule",
		"name=" + firewallNameProcess,
		"dir=in",
		"action=allow",
		"edge=yes",
		fmt.Sprintf("program=\"%s\"", exe),
		"protocol=udp",
		"profile=any",
		"enable=yes",
	}

	cmd := exec.Command("netsh", processRule...)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}

	out, err := cmd.CombinedOutput()
	if err != nil {
		zap.L().Error("Setup the firewall rules(PairMesh-Process) failed", zap.Error(err), zap.String("output", string(out)))
	}

	localAddrRule := []string{
		"advfirewall",
		"firewall",
		"add",
		"rule",
		"name=" + firewallNameIn,
		"dir=in",
		"action=allow",
		"localip=" + addr,
		"profile=any",
		"enable=yes",
	}

	cmd = exec.Command("netsh", localAddrRule...)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}

	out, err = cmd.CombinedOutput()
	if err != nil {
		zap.L().Error("Setup the firewall rules(PairMesh-In) failed", zap.Error(err), zap.String("output", string(out)))
	}
}

func clear() {
	// Clear all firewall rules
	begin := time.Now()

	defer func() {
		zap.L().Info("Clear the firewall rules", zap.Duration("duration", time.Since(begin)))
	}()

	clearIn := []string{
		"advfirewall",
		"firewall",
		"delete",
		"rule",
		"name=" + firewallNameIn,
		"dir=in",
	}

	cmd := exec.Command("netsh", clearIn...)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}

	out, err := cmd.CombinedOutput()
	if err != nil {
		zap.L().Error("Clear the firewall rules(PairMesh-In) failed", zap.Error(err), zap.String("output", string(out)))
	}

	clearProcess := []string{
		"advfirewall",
		"firewall",
		"delete",
		"rule",
		"name=" + firewallNameProcess,
		"dir=in",
	}
	cmd = exec.Command("netsh", clearProcess...)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}

	out, err = cmd.CombinedOutput()
	if err != nil {
		zap.L().Error("Clear the firewall rules(PairMesh-Process) failed", zap.Error(err), zap.String("output", string(out)))
	}
}
