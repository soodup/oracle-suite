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

package relay

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"testing"
	"time"

	"github.com/defiweb/go-eth/types"
	"github.com/stretchr/testify/assert"

	"github.com/chronicleprotocol/oracle-suite/pkg/contract"
	"github.com/chronicleprotocol/oracle-suite/pkg/transport/messages"
	"github.com/chronicleprotocol/oracle-suite/pkg/util/bn"
)

func TestScribeWorker(t *testing.T) {
	testFeed := types.MustAddressFromHex("0x1111111111111111111111111111111111111111")
	mockLogger := newMockLogger(t)
	mockContract := newMockScribeContract(t)
	mockMuSigStore := newMockSignatureProvider(t)

	sw := &scribeWorker{
		log:        mockLogger,
		muSigStore: mockMuSigStore,
		contract:   mockContract,
		dataModel:  "ETH/USD",
		spread:     0.05,
		expiration: 10 * time.Minute,
	}

	t.Run("above spread", func(t *testing.T) {
		mockLogger.reset(t)
		mockContract.reset(t)
		mockMuSigStore.reset(t)

		ctx := context.Background()
		musigTime := time.Now()
		musigCommitment := types.MustAddressFromHex("0x1234567890123456789012345678901234567890")
		musigSignature := big.NewInt(1234567890)
		mockLogger.InfoFn = func(args ...any) {}
		mockLogger.DebugFn = func(args ...any) {}
		mockContract.AddressFn = func() types.Address { return types.Address{} }
		mockContract.WatFn = func(ctx context.Context) (string, error) {
			return "ETH/USD", nil
		}
		mockContract.BarFn = func(ctx context.Context) (int, error) {
			return 1, nil
		}
		mockContract.FeedsFn = func(ctx context.Context) ([]types.Address, []uint8, error) {
			return []types.Address{testFeed}, []uint8{1}, nil
		}
		mockContract.ReadFn = func(ctx context.Context) (contract.PokeData, error) {
			return contract.PokeData{
				Val: bn.DecFixedPoint(100, contract.ScribePricePrecision),
				Age: time.Now().Add(-1 * time.Minute),
			}, nil
		}
		mockMuSigStore.SignaturesByDataModelFn = func(model string) []*messages.MuSigSignature {
			assert.Equal(t, "ETH/USD", model)
			return []*messages.MuSigSignature{
				{
					MuSigMessage: &messages.MuSigMessage{
						MsgMeta: messages.MuSigMeta{Meta: messages.MuSigMetaTickV1{
							Wat: "ETH/USD",
							Val: bn.DecFixedPoint(110, contract.ScribePricePrecision),
							Age: musigTime,
						}},
					},
					Commitment:       musigCommitment,
					SchnorrSignature: musigSignature,
				},
			}
		}

		pokeCalled := false
		mockContract.PokeFn = func(ctx context.Context, pokeData contract.PokeData, schnorrData contract.SchnorrData) (*types.Hash, *types.Transaction, error) {
			pokeCalled = true
			assert.Equal(t, bn.DecFixedPoint(110, contract.ScribePricePrecision), pokeData.Val)
			assert.Equal(t, musigTime, pokeData.Age)
			assert.Equal(t, musigCommitment, schnorrData.Commitment)
			assert.Equal(t, musigSignature, schnorrData.Signature)
			return types.HashFromBigIntPtr(big.NewInt(1)), &types.Transaction{}, nil
		}

		sw.tryUpdate(ctx, time.Now())
		assert.True(t, pokeCalled)
	})

	t.Run("within spread", func(t *testing.T) {
		mockLogger.reset(t)
		mockContract.reset(t)
		mockMuSigStore.reset(t)

		ctx := context.Background()
		musigTime := time.Now()
		musigCommitment := types.MustAddressFromHex("0x1234567890123456789012345678901234567890")
		musigSignature := big.NewInt(1234567890)
		mockLogger.InfoFn = func(args ...any) {}
		mockLogger.DebugFn = func(args ...any) {}
		mockContract.AddressFn = func() types.Address { return types.Address{} }
		mockContract.WatFn = func(ctx context.Context) (string, error) {
			return "ETH/USD", nil
		}
		mockContract.BarFn = func(ctx context.Context) (int, error) {
			return 1, nil
		}
		mockContract.FeedsFn = func(ctx context.Context) ([]types.Address, []uint8, error) {
			return []types.Address{testFeed}, []uint8{1}, nil
		}
		mockContract.ReadFn = func(ctx context.Context) (contract.PokeData, error) {
			return contract.PokeData{
				Val: bn.DecFixedPoint(100, contract.ScribePricePrecision),
				Age: time.Now().Add(-1 * time.Minute),
			}, nil
		}
		mockMuSigStore.SignaturesByDataModelFn = func(model string) []*messages.MuSigSignature {
			assert.Equal(t, "ETH/USD", model)
			return []*messages.MuSigSignature{
				{
					MuSigMessage: &messages.MuSigMessage{
						MsgMeta: messages.MuSigMeta{Meta: messages.MuSigMetaTickV1{
							Wat: "ETH/USD",
							Val: bn.DecFixedPoint(100, contract.ScribePricePrecision),
							Age: musigTime,
						}},
					},
					Commitment:       musigCommitment,
					SchnorrSignature: musigSignature,
				},
			}
		}

		sw.tryUpdate(ctx, time.Now())
	})

	t.Run("expired", func(t *testing.T) {
		mockLogger.reset(t)
		mockContract.reset(t)
		mockMuSigStore.reset(t)

		ctx := context.Background()
		musigTime := time.Now()
		musigCommitment := types.MustAddressFromHex("0x1234567890123456789012345678901234567890")
		musigSignature := big.NewInt(1234567890)
		mockLogger.InfoFn = func(args ...any) {}
		mockLogger.DebugFn = func(args ...any) {}
		mockContract.AddressFn = func() types.Address { return types.Address{} }
		mockContract.WatFn = func(ctx context.Context) (string, error) {
			return "ETH/USD", nil
		}
		mockContract.BarFn = func(ctx context.Context) (int, error) {
			return 1, nil
		}
		mockContract.FeedsFn = func(ctx context.Context) ([]types.Address, []uint8, error) {
			return []types.Address{testFeed}, []uint8{1}, nil
		}
		mockContract.ReadFn = func(ctx context.Context) (contract.PokeData, error) {
			return contract.PokeData{
				Val: bn.DecFixedPoint(100, contract.ScribePricePrecision),
				Age: time.Now().Add(-15 * time.Minute),
			}, nil
		}
		mockMuSigStore.SignaturesByDataModelFn = func(model string) []*messages.MuSigSignature {
			assert.Equal(t, "ETH/USD", model)
			return []*messages.MuSigSignature{
				{
					MuSigMessage: &messages.MuSigMessage{
						MsgMeta: messages.MuSigMeta{Meta: messages.MuSigMetaTickV1{
							Wat: "ETH/USD",
							Val: bn.DecFixedPoint(100, contract.ScribePricePrecision),
							Age: musigTime,
						}},
					},
					Commitment:       musigCommitment,
					SchnorrSignature: musigSignature,
				},
			}
		}

		pokeCalled := false
		mockContract.PokeFn = func(ctx context.Context, pokeData contract.PokeData, schnorrData contract.SchnorrData) (*types.Hash, *types.Transaction, error) {
			pokeCalled = true
			assert.Equal(t, bn.DecFixedPoint(100, contract.ScribePricePrecision), pokeData.Val)
			assert.Equal(t, musigTime, pokeData.Age)
			assert.Equal(t, musigCommitment, schnorrData.Commitment)
			assert.Equal(t, musigSignature, schnorrData.Signature)
			return types.HashFromBigIntPtr(big.NewInt(1)), &types.Transaction{}, nil
		}

		sw.tryUpdate(ctx, time.Now())
		assert.True(t, pokeCalled)
	})

	t.Run("old signature", func(t *testing.T) {
		mockLogger.reset(t)
		mockContract.reset(t)
		mockMuSigStore.reset(t)

		ctx := context.Background()
		musigTime := time.Now().Add(-15 * time.Minute)
		musigCommitment := types.MustAddressFromHex("0x1234567890123456789012345678901234567890")
		musigSignature := big.NewInt(1234567890)
		mockLogger.InfoFn = func(args ...any) {}
		mockLogger.DebugFn = func(args ...any) {}
		mockContract.AddressFn = func() types.Address { return types.Address{} }
		mockContract.WatFn = func(ctx context.Context) (string, error) {
			return "ETH/USD", nil
		}
		mockContract.BarFn = func(ctx context.Context) (int, error) {
			return 1, nil
		}
		mockContract.FeedsFn = func(ctx context.Context) ([]types.Address, []uint8, error) {
			return []types.Address{testFeed}, []uint8{1}, nil
		}
		mockContract.ReadFn = func(ctx context.Context) (contract.PokeData, error) {
			return contract.PokeData{
				Val: bn.DecFixedPoint(100, contract.ScribePricePrecision),
				Age: time.Now().Add(-1 * time.Minute),
			}, nil
		}
		mockMuSigStore.SignaturesByDataModelFn = func(model string) []*messages.MuSigSignature {
			assert.Equal(t, "ETH/USD", model)
			return []*messages.MuSigSignature{
				{
					MuSigMessage: &messages.MuSigMessage{
						MsgMeta: messages.MuSigMeta{Meta: messages.MuSigMetaTickV1{
							Wat: "ETH/USD",
							Val: bn.DecFixedPoint(110, contract.ScribePricePrecision),
							Age: musigTime,
						}},
					},
					Commitment:       musigCommitment,
					SchnorrSignature: musigSignature,
				},
			}
		}

		sw.tryUpdate(ctx, time.Now())
	})

	t.Run("broken message", func(t *testing.T) {
		invalidMessages := []*messages.MuSigSignature{
			{
				MuSigMessage:     nil,
				Commitment:       types.ZeroAddress,
				SchnorrSignature: nil,
			},
			{
				MuSigMessage: &messages.MuSigMessage{
					MsgMeta: messages.MuSigMeta{Meta: nil},
				},
				Commitment:       types.ZeroAddress,
				SchnorrSignature: nil,
			},
			{
				MuSigMessage: &messages.MuSigMessage{
					MsgMeta: messages.MuSigMeta{Meta: nil},
				},
				Commitment:       types.ZeroAddress,
				SchnorrSignature: big.NewInt(1234567890),
			},
			{
				MuSigMessage: &messages.MuSigMessage{
					MsgMeta: messages.MuSigMeta{Meta: messages.MuSigMetaTickV1{
						Wat: "ETH/USD",
						Val: nil,
						Age: time.Now(),
					}},
				},
				Commitment:       types.ZeroAddress,
				SchnorrSignature: nil,
			},
			{
				MuSigMessage: &messages.MuSigMessage{
					MsgMeta: messages.MuSigMeta{Meta: messages.MuSigMetaTickV1{
						Wat: "ETH/USD",
						Val: bn.DecFixedPoint(110, contract.ScribePricePrecision),
						Age: time.Now(),
					}},
				},
				Commitment:       types.ZeroAddress,
				SchnorrSignature: nil,
			},
			{
				MuSigMessage: &messages.MuSigMessage{
					MsgMeta: messages.MuSigMeta{Meta: messages.MuSigMetaTickV1{
						Wat: "ETH/USD",
						Val: nil,
						Age: time.Now(),
					}},
				},
				Commitment:       types.ZeroAddress,
				SchnorrSignature: big.NewInt(1234567890),
			},
		}

		for i, m := range invalidMessages {
			t.Run(fmt.Sprintf("msg-%d", i+1), func(t *testing.T) {
				mockLogger.reset(t)
				mockContract.reset(t)
				mockMuSigStore.reset(t)

				ctx := context.Background()
				mockLogger.InfoFn = func(args ...any) {}
				mockLogger.DebugFn = func(args ...any) {}
				mockContract.AddressFn = func() types.Address { return types.Address{} }
				mockContract.WatFn = func(ctx context.Context) (string, error) {
					return "ETH/USD", nil
				}
				mockContract.BarFn = func(ctx context.Context) (int, error) {
					return 1, nil
				}
				mockContract.FeedsFn = func(ctx context.Context) ([]types.Address, []uint8, error) {
					return []types.Address{testFeed}, []uint8{1}, nil
				}
				mockContract.ReadFn = func(ctx context.Context) (contract.PokeData, error) {
					return contract.PokeData{
						Val: bn.DecFixedPoint(100, contract.ScribePricePrecision),
						Age: time.Now().Add(-15 * time.Minute),
					}, nil
				}
				mockMuSigStore.SignaturesByDataModelFn = func(model string) []*messages.MuSigSignature {
					assert.Equal(t, "ETH/USD", model)
					return []*messages.MuSigSignature{m}
				}

				sw.tryUpdate(ctx, time.Now())
			})
		}
	})

	t.Run("wat call error", func(t *testing.T) {
		mockLogger.reset(t)
		mockContract.reset(t)
		mockMuSigStore.reset(t)

		ctx := context.Background()
		mockContract.AddressFn = func() types.Address { return types.Address{} }
		mockContract.WatFn = func(ctx context.Context) (string, error) {
			return "", errors.New("error")
		}

		errLogCalled := false
		mockLogger.ErrorFn = func(args ...any) {
			errLogCalled = true
		}

		sw.tryUpdate(ctx, time.Now())
		assert.True(t, errLogCalled)
	})

	t.Run("read call error", func(t *testing.T) {
		mockLogger.reset(t)
		mockContract.reset(t)
		mockMuSigStore.reset(t)

		ctx := context.Background()
		mockContract.AddressFn = func() types.Address { return types.Address{} }
		mockContract.WatFn = func(ctx context.Context) (string, error) {
			return "ETH/USD", nil
		}
		mockContract.ReadFn = func(ctx context.Context) (contract.PokeData, error) {
			return contract.PokeData{}, errors.New("error")
		}

		errLogCalled := false
		mockLogger.ErrorFn = func(args ...any) {
			errLogCalled = true
		}

		sw.tryUpdate(ctx, time.Now())
		assert.True(t, errLogCalled)
	})

	t.Run("bar call error", func(t *testing.T) {
		mockLogger.reset(t)
		mockContract.reset(t)
		mockMuSigStore.reset(t)

		ctx := context.Background()
		mockContract.AddressFn = func() types.Address { return types.Address{} }
		mockContract.WatFn = func(ctx context.Context) (string, error) {
			return "ETH/USD", nil
		}
		mockContract.ReadFn = func(ctx context.Context) (contract.PokeData, error) {
			return contract.PokeData{}, errors.New("error")
		}
		mockContract.BarFn = func(ctx context.Context) (int, error) {
			return 0, errors.New("network error")
		}

		errLogCalled := false
		mockLogger.ErrorFn = func(args ...any) {
			errLogCalled = true
		}

		sw.tryUpdate(ctx, time.Now())
		assert.True(t, errLogCalled)
	})

	t.Run("feeds call error", func(t *testing.T) {
		mockLogger.reset(t)
		mockContract.reset(t)
		mockMuSigStore.reset(t)

		ctx := context.Background()
		mockContract.AddressFn = func() types.Address { return types.Address{} }
		mockContract.WatFn = func(ctx context.Context) (string, error) {
			return "ETH/USD", nil
		}
		mockContract.ReadFn = func(ctx context.Context) (contract.PokeData, error) {
			return contract.PokeData{}, errors.New("error")
		}
		mockContract.BarFn = func(ctx context.Context) (int, error) {
			return 1, nil
		}
		mockContract.FeedsFn = func(ctx context.Context) ([]types.Address, []uint8, error) {
			return nil, nil, errors.New("error")
		}

		errLogCalled := false
		mockLogger.ErrorFn = func(args ...any) {
			errLogCalled = true
		}

		sw.tryUpdate(ctx, time.Now())
		assert.True(t, errLogCalled)
	})

	t.Run("delay", func(t *testing.T) {
		mockLogger.reset(t)
		mockContract.reset(t)
		mockMuSigStore.reset(t)

		sw.delay = 1 * time.Minute
		defer func() { sw.delay = 0 }()

		ctx := context.Background()
		musigTime := time.Now()
		musigCommitment := types.MustAddressFromHex("0x1234567890123456789012345678901234567890")
		musigSignature := big.NewInt(1234567890)
		mockLogger.InfoFn = func(args ...any) {}
		mockLogger.DebugFn = func(args ...any) {}
		mockContract.AddressFn = func() types.Address { return types.Address{} }
		mockContract.WatFn = func(ctx context.Context) (string, error) {
			return "ETH/USD", nil
		}
		mockContract.BarFn = func(ctx context.Context) (int, error) {
			return 1, nil
		}
		mockContract.FeedsFn = func(ctx context.Context) ([]types.Address, []uint8, error) {
			return []types.Address{testFeed}, []uint8{1}, nil
		}
		mockContract.ReadFn = func(ctx context.Context) (contract.PokeData, error) {
			return contract.PokeData{
				Val: bn.DecFixedPoint(100, contract.ScribePricePrecision),
				Age: musigTime.Add(-1 * time.Minute),
			}, nil
		}
		mockMuSigStore.SignaturesByDataModelFn = func(model string) []*messages.MuSigSignature {
			assert.Equal(t, "ETH/USD", model)
			return []*messages.MuSigSignature{
				{
					MuSigMessage: &messages.MuSigMessage{
						MsgMeta: messages.MuSigMeta{Meta: messages.MuSigMetaTickV1{
							Wat: "ETH/USD",
							Val: bn.DecFixedPoint(110, contract.ScribePricePrecision),
							Age: musigTime,
						}},
					},
					Commitment:       musigCommitment,
					SchnorrSignature: musigSignature,
				},
			}
		}

		// It should not poke because of delay.
		sw.tryUpdate(ctx, musigTime)

		// Still before delay.
		sw.tryUpdate(ctx, musigTime.Add(30*time.Second))

		// After delay.
		mockContract.PokeFn = func(ctx context.Context, pokeData contract.PokeData, schnorrData contract.SchnorrData) (*types.Hash, *types.Transaction, error) {
			return types.HashFromBigIntPtr(big.NewInt(1)), &types.Transaction{}, nil
		}
		sw.tryUpdate(ctx, time.Now().Add(2*time.Minute))
	})
}
