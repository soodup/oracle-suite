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
	"testing"
	"time"

	suite "github.com/chronicleprotocol/oracle-suite"
	feedConfig "github.com/chronicleprotocol/oracle-suite/pkg/config/feednext"
	"github.com/chronicleprotocol/oracle-suite/pkg/log/null"
	"github.com/chronicleprotocol/oracle-suite/pkg/util/timeutil"
	"github.com/hashicorp/hcl/v2"
	"github.com/stretchr/testify/require"
)

type morphTest struct {
	Ghost feedConfig.Config `hcl:"ghost,block"`

	// HCL fields:
	Remain  hcl.Body        `hcl:",remain"` // To ignore unknown blocks.
	Content hcl.BodyContent `hcl:",content"`
}

func (morphTest) DefaultEmbeds() [][]byte {
	return [][]byte{
		suite.ConfigContracts,
		suite.ConfigDefaults,
		suite.ConfigGhost,
		suite.ConfigGofer,
		suite.ConfigEthereum,
		suite.ConfigTransport,
	}
}

func TestConfig(t *testing.T) {
	tests := []struct {
		path string
		test func(*testing.T, *Morph)
	}{
		{
			path: "config-morph.hcl",
			test: func(t *testing.T, service *Morph) {
				config, _ := service.baseConfig.(*morphTest)
				require.Equal(t, len(config.Ghost.DataModels), 0)
				err := service.ForceUpdate()
				require.NoError(t, err)
				// Data Model was loaded
				require.Greater(t, len(config.Ghost.DataModels), 0)
			},
		},
	}
	for _, test := range tests {
		t.Run(test.path, func(t *testing.T) {
			var alternative morphTest
			cfg := MorphConfig{
				MorphFile:  "./testdata/" + test.path,
				Interval:   timeutil.NewTicker(time.Second * time.Duration(60)),
				BaseConfig: &alternative,
				Handlers:   nil,
				Logger:     null.New(),
			}
			service, err := NewMorphService(cfg)
			require.NoError(t, err)
			require.NotNil(t, service)
			test.test(t, service)
		})
	}
}
