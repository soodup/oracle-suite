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
	"fmt"
	"time"

	"github.com/hashicorp/hcl/v2"

	"github.com/chronicleprotocol/oracle-suite/pkg/config"
	"github.com/chronicleprotocol/oracle-suite/pkg/log"
	"github.com/chronicleprotocol/oracle-suite/pkg/util/timeutil"
)

// Service interface defines the function type of which services should implement to reflect with latest config
type Service interface {
	// UpdateConfig will be called every time Morph fetches the latest config from on-chain
	// wanted is the pointer to field in base config, which is specified for each service as latest config
	// i.e. For FeedService, Ghost.DataModels will be passed
	UpdateConfig(wanted any) error
}

type Handler struct {
	Service Service
	Wanted  any
}

type Config struct {
	MorphFile string `hcl:"cache_path"`
	Interval  uint32 `hcl:"interval"`

	// HCL fields:
	Range   hcl.Range       `hcl:",range"`
	Content hcl.BodyContent `hcl:",content"`
}

type Dependencies struct {
	BaseConfig config.HasDefaults
	Handlers   []Handler
	Logger     log.Logger
}

func (c *Config) ConfigureMorph(d Dependencies) (*Morph, error) {
	if c.Interval == 0 {
		return nil, hcl.Diagnostics{&hcl.Diagnostic{
			Summary:  "Validation error",
			Detail:   "Interval cannot be zero",
			Severity: hcl.DiagError,
		}}
	}

	cfg := MorphConfig{
		MorphFile:  c.MorphFile,
		Interval:   timeutil.NewTicker(time.Second * time.Duration(c.Interval)),
		BaseConfig: d.BaseConfig,
		Handlers:   d.Handlers,
		Logger:     d.Logger,
	}
	morph, err := NewMorphService(cfg)
	if err != nil {
		return nil, &hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Runtime error",
			Detail:   fmt.Sprintf("Failed to create the Morph service: %v", err),
			Subject:  c.Range.Ptr(),
		}
	}
	return morph, nil
}
