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

package mocks

import (
	"context"
	"math/big"

	"github.com/defiweb/go-eth/types"
	"github.com/stretchr/testify/mock"
)

type Client struct {
	mock.Mock
}

func (e *Client) BlockNumber(ctx context.Context) (*big.Int, error) {
	args := e.Called(ctx)
	return args.Get(0).(*big.Int), args.Error(1)
}

func (e *Client) Block(ctx context.Context) (*types.Block, error) {
	args := e.Called(ctx)
	return args.Get(0).(*types.Block), args.Error(1)
}

func (e *Client) Call(ctx context.Context, call types.Call) ([]byte, error) {
	args := e.Called(ctx, call)
	return args.Get(0).([]byte), args.Error(1)
}

func (e *Client) CallBlocks(ctx context.Context, call types.Call, blocks []int64) ([][]byte, error) {
	args := e.Called(ctx, call, blocks)
	return args.Get(0).([][]byte), args.Error(1)
}

func (e *Client) MultiCall(ctx context.Context, calls []types.Call) ([][]byte, error) {
	args := e.Called(ctx, calls)
	return args.Get(0).([][]byte), args.Error(1)
}

func (e *Client) Storage(ctx context.Context, address types.Address, key types.Hash) ([]byte, error) {
	args := e.Called(ctx, address, key)
	return args.Get(0).([]byte), args.Error(1)
}

func (e *Client) Balance(ctx context.Context, address types.Address) (*big.Int, error) {
	args := e.Called(ctx, address)
	return args.Get(0).(*big.Int), args.Error(1)
}

func (e *Client) SendTransaction(ctx context.Context, transaction *types.Transaction) (*types.Hash, error) {
	args := e.Called(ctx, transaction)
	return args.Get(0).(*types.Hash), args.Error(1)
}

func (e *Client) FilterLogs(ctx context.Context, query types.FilterLogsQuery) ([]types.Log, error) {
	args := e.Called(ctx, query)
	return args.Get(0).([]types.Log), args.Error(1)
}
