//  Copyright (C) 2021-2023 Chronicle Labs, Inc.
//
//  This program is free software: you can redistribute it and/or modify
//  it under the terms of the GNU Affero General Public License as
//  published by the Free Software Foundation, either version 3 of the
//  License, or (at your option) any later version.
//
//  This program is distributed in the hope that it will be useful,
//  but WITHOUT ANY WARRANTY; without even the implied warranty of
//  MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
//  GNU Affero General Public License for more details.
//
//  You should have received a copy of the GNU Affero General Public License
//  along with this program.  If not, see <http://www.gnu.org/licenses/>.

package morph

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/hashicorp/hcl/v2"

	"github.com/chronicleprotocol/oracle-suite/pkg/config"
	"github.com/chronicleprotocol/oracle-suite/pkg/log"
	"github.com/chronicleprotocol/oracle-suite/pkg/log/null"
	"github.com/chronicleprotocol/oracle-suite/pkg/util/timeutil"
)

type EnvVarsConfig struct {
	EnvVars map[string]string `hcl:"env_vars"`

	// HCL fields:
	Range   hcl.Range       `hcl:",range"`
	Content hcl.BodyContent `hcl:",content"`
}

type Morph struct {
	ctx    context.Context
	waitCh chan error

	morphFile  string
	interval   *timeutil.Ticker
	baseConfig config.HasDefaults
	handlers   []Handler
	log        log.Logger
}

type MorphConfig struct { //nolint:revive
	MorphFile  string
	Interval   *timeutil.Ticker
	BaseConfig config.HasDefaults
	Handlers   []Handler
	Logger     log.Logger
}

const MorphLoggerTag = "MORPH"

// NewMorphService creates Morph, which proceeds the following steps:
// - Periodically pull the config/env from on-chain
// - Merge into the local config cache, if failed in merging, do not merge anymore
// - If not found local config cache, use embedded config instead
// - Notify to every handler services with proper config field
func NewMorphService(cfg MorphConfig) (*Morph, error) {
	m := &Morph{
		waitCh:     make(chan error),
		log:        cfg.Logger.WithField("tag", MorphLoggerTag),
		morphFile:  cfg.MorphFile,
		interval:   cfg.Interval,
		baseConfig: cfg.BaseConfig,
		handlers:   cfg.Handlers,
	}
	if cfg.Logger == nil {
		cfg.Logger = null.New()
	}
	return m, nil
}

func (m *Morph) Start(ctx context.Context) error {
	if m.ctx != nil {
		return errors.New("service can be started only once")
	}
	if ctx == nil {
		return errors.New("context must not be nil")
	}
	m.ctx = ctx
	m.log.
		WithFields(log.Fields{
			"interval": m.interval.Duration(),
		}).
		Debug("Starting")
	m.interval.Start(m.ctx)
	go m.reloadRoutine()
	go m.contextCancelHandler()
	return nil
}

func (m *Morph) Wait() <-chan error {
	return m.waitCh
}

func (m *Morph) ForceUpdate() error {
	// Load env variables from external file
	var vars EnvVarsConfig
	err := config.LoadFiles(&vars, []string{m.morphFile})
	if err != nil {
		m.log.
			WithError(err).
			WithField("cache_path", m.morphFile).
			Error("Failed loading local config cache")
		return err
	}

	// Set env variables to OS ENV
	for key, value := range vars.EnvVars {
		os.Setenv(key, value)
	}

	// Load again hcl config with default embedded
	err = config.LoadEmbeds(&m.baseConfig, m.baseConfig.DefaultEmbeds())

	// Cleanup OS ENV
	for key := range vars.EnvVars {
		os.Setenv(key, "")
	}

	if err != nil {
		fields := log.Fields{}
		for key, value := range vars.EnvVars {
			fields[key] = value
		}
		m.log.WithError(err).WithFields(fields).Error("Failed loading config with env vars")
		return err
	}

	// Notify updates to handlers
	for _, handler := range m.handlers {
		if handler.Service != nil {
			err := handler.Service.UpdateConfig(handler.Wanted)
			if err != nil {
				m.log.WithError(err).Warn("Failed updating config")
			}
		}
	}
	return nil
}

func (m *Morph) reloadRoutine() {
	for {
		select {
		case <-m.ctx.Done():
			return
		case <-m.interval.TickCh():
			err := m.ForceUpdate()
			if err != nil {
				fmt.Println(err)
			}
			return
		}
	}
}

func (m *Morph) contextCancelHandler() {
	defer func() { close(m.waitCh) }()
	defer m.log.Info("Stopped")
	<-m.ctx.Done()
}
