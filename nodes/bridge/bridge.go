// Copyright 2020 Nokia
// Licensed under the BSD 3-Clause License.
// SPDX-License-Identifier: BSD-3-Clause

package bridge

import (
	"context"
	"fmt"
	"os/exec"
	"regexp"
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/srl-labs/containerlab/nodes"
	"github.com/srl-labs/containerlab/types"
)

var kindnames = []string{"bridge"}

const (
	iptCheckCmd = "-vL FORWARD -w 5"
	iptAllowCmd = "-I FORWARD -i %s -j ACCEPT -w 5"
)

func init() {
	nodes.Register(kindnames, func() nodes.Node {
		return new(bridge)
	})
}

type bridge struct {
	nodes.DefaultNode
}

func (s *bridge) Init(cfg *types.NodeConfig, opts ...nodes.NodeOption) error {
	s.Cfg = cfg
	for _, o := range opts {
		o(s)
	}
	s.Cfg.DeploymentStatus = "created" // since we do not create bridges with clab, the status is implied here
	return nil
}

func (*bridge) Deploy(_ context.Context) error { return nil }
func (*bridge) Delete(_ context.Context) error { return nil }
func (*bridge) GetImages() map[string]string   { return map[string]string{} }

func (b *bridge) PostDeploy(_ context.Context, _ map[string]nodes.Node) error {
	return b.installIPTablesBridgeFwdRule()
}

// installIPTablesBridgeFwdRule calls iptables to install `allow` rule for traffic passing through the bridge
// otherwise, communication over the bridge is not permitted.
func (b *bridge) installIPTablesBridgeFwdRule() (err error) {
	// first check if a rule already exists for this bridge to not create duplicates
	res, err := exec.Command("iptables", strings.Split(iptCheckCmd, " ")...).Output()

	re, _ := regexp.Compile(fmt.Sprintf("ACCEPT[^\n]+%s", b.Cfg.ShortName))

	if re.Match(res) {
		log.Debugf("found iptables forwarding rule targeting the bridge %q. Skipping creation of the forwarding rule.", b.Cfg.ShortName)
		return err
	}
	if err != nil {
		return fmt.Errorf("failed to add iptables forwarding rule for bridge %q: %w", b.Cfg.ShortName, err)
	}

	cmd := fmt.Sprintf(iptAllowCmd, b.Cfg.ShortName)

	log.Debugf("Installing iptables rules for bridge %q", b.Cfg.ShortName)

	stdOutErr, err := exec.Command("iptables", strings.Split(cmd, " ")...).CombinedOutput()

	log.Debugf("iptables install stdout for bridge %s:%s", b.Cfg.ShortName, stdOutErr)

	if err != nil {
		log.Warnf("iptables install stdout/stderr result is: %s", stdOutErr)
		return fmt.Errorf("unable to create iptables rules: %w", err)
	}

	return nil
}
