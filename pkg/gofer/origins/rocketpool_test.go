//  Copyright (C) 2020 Maker Ecosystem Growth Holdings, INC.
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

package origins

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/mock"

	"github.com/chronicleprotocol/oracle-suite/pkg/ethereum"
	ethereumMocks "github.com/chronicleprotocol/oracle-suite/pkg/ethereum/mocks"

	"github.com/stretchr/testify/suite"
)

type RockerPoolSuite struct {
	suite.Suite
	addresses ContractAddresses
	client    *ethereumMocks.Client
	origin    *BaseExchangeHandler
}

func (suite *RockerPoolSuite) SetupSuite() {
	suite.addresses = ContractAddresses{
		"RETH/ETH": "0xae78736Cd615f374D3085123A210448E74Fc6393",
	}
}
func (suite *RockerPoolSuite) TearDownSuite() {
	suite.addresses = nil
}

func (suite *RockerPoolSuite) SetupTest() {
	suite.client = &ethereumMocks.Client{}
	o, err := NewRockerPool(suite.client, suite.addresses, []int64{0, 10, 20})
	suite.NoError(err)
	suite.origin = NewBaseExchangeHandler(o, nil)
}

func (suite *RockerPoolSuite) TearDownTest() {
	suite.origin = nil
	suite.client = nil
}

func (suite *RockerPoolSuite) Origin() Handler {
	return suite.origin
}

func TestRockerPoolSuite(t *testing.T) {
	suite.Run(t, new(RockerPoolSuite))
}

func (suite *RockerPoolSuite) TestSuccessResponse() {
	resp := [][]byte{
		common.BigToHash(big.NewInt(0.94 * 1e18)).Bytes(),
		common.BigToHash(big.NewInt(0.98 * 1e18)).Bytes(),
		common.BigToHash(big.NewInt(0.99 * 1e18)).Bytes(),
	}
	suite.client.On(
		"CallBlocks",
		mock.Anything,
		ethereum.Call{
			Address: ethereum.HexToAddress("0xae78736Cd615f374D3085123A210448E74Fc6393"),
			Data:    ethereum.HexToBytes("0xe6aa216c"),
		},
		[]int64{0, 10, 20},
	).Return(resp, nil).Once()

	pair := Pair{Base: "RETH", Quote: "ETH"}

	results1 := suite.origin.Fetch([]Pair{pair})
	suite.Require().NoError(results1[0].Error)
	suite.Equal(0.97, results1[0].Price.Price)
	suite.Greater(results1[0].Price.Timestamp.Unix(), int64(0))

	suite.client.AssertNumberOfCalls(suite.T(), "CallBlocks", 1)
}

func (suite *RockerPoolSuite) TestSuccessResponse_Inverted() {
	resp := [][]byte{
		common.BigToHash(big.NewInt(0.94 * 1e18)).Bytes(),
		common.BigToHash(big.NewInt(0.98 * 1e18)).Bytes(),
		common.BigToHash(big.NewInt(0.99 * 1e18)).Bytes(),
	}
	suite.client.On(
		"CallBlocks",
		mock.Anything,
		ethereum.Call{
			Address: ethereum.HexToAddress("0xae78736Cd615f374D3085123A210448E74Fc6393"),
			Data:    ethereum.HexToBytes("0x4346f03e0000000000000000000000000000000000000000000000000de0b6b3a7640000"),
		},
		[]int64{0, 10, 20},
	).Return(resp, nil).Once()

	pair := Pair{Base: "ETH", Quote: "RETH"}

	results1 := suite.origin.Fetch([]Pair{pair})
	suite.Require().NoError(results1[0].Error)
	suite.Equal(0.97, results1[0].Price.Price)
	suite.Greater(results1[0].Price.Timestamp.Unix(), int64(0))

	suite.client.AssertNumberOfCalls(suite.T(), "CallBlocks", 1)
}

func (suite *RockerPoolSuite) TestFailOnWrongPair() {
	pair := Pair{Base: "x", Quote: "y"}
	cr := suite.origin.Fetch([]Pair{pair})
	suite.Require().EqualError(cr[0].Error, "failed to get contract address for pair: x/y")
}
