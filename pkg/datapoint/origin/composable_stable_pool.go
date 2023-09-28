package origin

import (
	"fmt"
	"math/big"

	"github.com/defiweb/go-eth/hexutil"
	"github.com/defiweb/go-eth/types"

	"github.com/chronicleprotocol/oracle-suite/pkg/datapoint/value"
)

const Bytes32Length = 32

type Bytes32 [Bytes32Length]byte

var ZeroBytes32 = Bytes32{}

func Bytes32FromBytes(b []byte) (Bytes32, error) {
	if len(b) > Bytes32Length {
		return ZeroBytes32, fmt.Errorf("bytes too long %d", len(b))
	}
	var bytes32 Bytes32
	copy(bytes32[Bytes32Length-len(b):], b)
	return bytes32, nil
}

func Bytes32FromHex(h string) (Bytes32, error) {
	b, err := hexutil.HexToBytes(h)
	if err != nil {
		return ZeroBytes32, err
	}
	return Bytes32FromBytes(b)
}

func MustBytes32FromBytes(b []byte) Bytes32 {
	bytes32, err := Bytes32FromBytes(b)
	if err != nil {
		panic(err)
	}
	return bytes32
}

func MustBytes32FromHex(h string) Bytes32 {
	bytes32, err := Bytes32FromHex(h)
	if err != nil {
		panic(err)
	}
	return bytes32
}

func (b Bytes32) Bytes() []byte {
	return b[:]
}

func (b Bytes32) String() string {
	return hexutil.BytesToHex(b[:])
}

type ComposableStablePoolConfig struct {
	Pair            value.Pair
	ContractAddress types.Address
}

type LastJoinExitData struct {
	LastJoinExitAmplification *big.Int
	LastPostJoinExitInvariant *big.Int
}

type TokenRateCache struct {
	Rate     *big.Int
	OldRate  *big.Int
	Duration *big.Int
	Expires  *big.Int
}

type AmplificationParameter struct {
	Value      *big.Int
	IsUpdating bool
	Precision  *big.Int
}

type Extra struct {
	AmplificationParameter              AmplificationParameter
	ScalingFactors                      []*big.Int
	LastJoinExit                        LastJoinExitData
	TokensExemptFromYieldProtocolFee    []bool
	TokenRateCaches                     []TokenRateCache
	ProtocolFeePercentageCacheSwapType  *big.Int
	ProtocolFeePercentageCacheYieldType *big.Int
}

type ComposableStablePoolFullConfig struct {
	Pair              value.Pair
	ContractAddress   types.Address
	PoolID            Bytes32
	Vault             types.Address
	Tokens            []types.Address
	BptIndex          int
	RateProviders     []types.Address
	Balances          []*big.Int
	TotalSupply       *big.Int
	SwapFeePercentage *big.Int
	Extra             Extra
}

type ComposableStablePool struct {
	config ComposableStablePoolFullConfig
}

func NewComposableStablePool(config ComposableStablePoolConfig) (*ComposableStablePool, error) {
	return &ComposableStablePool{
		config: ComposableStablePoolFullConfig{
			Pair:            config.Pair,
			ContractAddress: config.ContractAddress,
		},
	}, nil
}

func NewComposableStablePoolFull(config ComposableStablePoolFullConfig) (*ComposableStablePool, error) {
	return &ComposableStablePool{
		config,
	}, nil
}

// CreateInitCalls create the calls for `multicall` to get vault address and pool id
func (c *ComposableStablePool) CreateInitCalls() ([]types.Call, error) {
	if c.config.ContractAddress == types.ZeroAddress {
		return nil, fmt.Errorf("unknown contract address: %s", c.config.Pair.String())
	}

	var calls []types.Call
	// Calls for `getPoolID`
	callData, _ := getPoolID.EncodeArgs()
	calls = append(calls, types.Call{
		To:    &c.config.ContractAddress,
		Input: callData,
	})
	// Calls for `getVault`
	callData, _ = getVault.EncodeArgs()
	calls = append(calls, types.Call{
		To:    &c.config.ContractAddress,
		Input: callData,
	})
	// Calls for `getBptIndex`
	callData, _ = getBptIndex.EncodeArgs()
	calls = append(calls, types.Call{
		To:    &c.config.ContractAddress,
		Input: callData,
	})
	// Calls for `getRateProviders`
	callData, _ = getRateProviders.EncodeArgs()
	calls = append(calls, types.Call{
		To:    &c.config.ContractAddress,
		Input: callData,
	})
	return calls, nil
}

// DecodeInitCalls decode the returned bytes of `multicall` that was triggered for `createInitCalls`
func (c *ComposableStablePool) DecodeInitCalls(resp [][]byte) error {
	if len(resp) != 4 {
		return fmt.Errorf("not matched response for init calls: %s, %d", c.config.Pair.String(), len(resp))
	}
	var poolID = MustBytes32FromBytes(resp[0])
	var vault = types.MustAddressFromBytes(resp[1][len(resp[1])-types.AddressLength:])
	var bptIndex = new(big.Int).SetBytes(resp[2]).Int64()
	var rateProviders []types.Address
	if err := getRateProviders.DecodeValues(resp[3], &rateProviders); err != nil {
		return fmt.Errorf("failed decoding rate providers calls: %s, %w", c.config.Pair.String(), err)
	}
	c.config.PoolID = poolID
	c.config.Vault = vault
	c.config.BptIndex = int(bptIndex)
	c.config.RateProviders = rateProviders
	return nil
}

func (c *ComposableStablePool) CreatePoolTokensCall() (types.Call, error) {
	if c.config.PoolID.String() == "" || c.config.Vault == types.ZeroAddress {
		return types.Call{}, fmt.Errorf("unknown vault or pool id: %s", c.config.Pair.String())
	}

	// Calls for `getPoolTokens`
	callData, _ := getPoolTokens.EncodeArgs(c.config.PoolID.Bytes())
	return types.Call{
		To:    &c.config.Vault,
		Input: callData,
	}, nil
}

func (c *ComposableStablePool) DecodePoolTokensCall(resp []byte) error {
	var tokens []types.Address
	var balances []*big.Int
	if err := getPoolTokens.DecodeValues(resp, &tokens, &balances, nil); err != nil {
		return fmt.Errorf("failed decoding pool tokens calls: %s, %w", c.config.Pair.String(), err)
	}
	c.config.Tokens = tokens
	c.config.Balances = balances
	return nil
}

func (c *ComposableStablePool) CreatePoolParamsCalls() ([]types.Call, error) {
	if c.config.ContractAddress == types.ZeroAddress {
		return nil, fmt.Errorf("unknown contract address: %s", c.config.Pair.String())
	}

	var calls []types.Call
	// Calls for `getSwapFeePercentage`
	callData, _ := getSwapFeePercentage.EncodeArgs()
	calls = append(calls, types.Call{
		To:    &c.config.ContractAddress,
		Input: callData,
	})
	// Calls for `getAmplificationParameter`
	callData, _ = getAmplificationParameter.EncodeArgs()
	calls = append(calls, types.Call{
		To:    &c.config.ContractAddress,
		Input: callData,
	})
	// Calls for `getScalingFactors`
	callData, _ = getScalingFactors.EncodeArgs()
	calls = append(calls, types.Call{
		To:    &c.config.ContractAddress,
		Input: callData,
	})
	// Calls for `getLastJoinExitData`
	callData, _ = getLastJoinExitData.EncodeArgs()
	calls = append(calls, types.Call{
		To:    &c.config.ContractAddress,
		Input: callData,
	})
	// Calls for `getTotalSupply`
	callData, _ = getTotalSupply.EncodeArgs()
	calls = append(calls, types.Call{
		To:    &c.config.ContractAddress,
		Input: callData,
	})
	// Calls for `getProtocolFeePercentageCache(SWAP)`
	callData, _ = getProtocolFeePercentageCache.EncodeArgs(0)
	calls = append(calls, types.Call{
		To:    &c.config.ContractAddress,
		Input: callData,
	})
	// Calls for `getProtocolFeePercentageCache(YIELD)`
	callData, _ = getProtocolFeePercentageCache.EncodeArgs(2)
	calls = append(calls, types.Call{
		To:    &c.config.ContractAddress,
		Input: callData,
	})
	for _, token := range c.config.Tokens {
		// Calls for `isTokenExemptFromYieldProtocolFee(token)`
		callData, _ = isTokenExemptFromYieldProtocolFee.EncodeArgs(token)
		calls = append(calls, types.Call{
			To:    &c.config.ContractAddress,
			Input: callData,
		})
	}
	return calls, nil
}

func (c *ComposableStablePool) DecodePoolParamsCalls(resp [][]byte) error {
	if len(resp) != 7+len(c.config.Tokens) {
		return fmt.Errorf("not matched response for pool params calls: %s, %d", c.config.Pair.String(), len(resp))
	}
	var swapFeePercentage = new(big.Int).SetBytes(resp[0])
	var amplificationParameter, amplificationPrecision *big.Int
	var isUpdating bool
	if err := getAmplificationParameter.DecodeValues(resp[1], &amplificationParameter, &isUpdating, &amplificationPrecision); err != nil {
		return fmt.Errorf("failed decoding amplification parameter calls: %s, %w", c.config.Pair.String(), err)
	}
	var scalingFactors []*big.Int
	if err := getScalingFactors.DecodeValues(resp[2], &scalingFactors); err != nil {
		return fmt.Errorf("failed decoding scaling factors calls: %s, %w", c.config.Pair.String(), err)
	}
	var lastJoinExitAmplification, lastPostJoinExitInvariant *big.Int
	if err := getLastJoinExitData.DecodeValues(resp[3], &lastJoinExitAmplification, &lastPostJoinExitInvariant); err != nil {
		return fmt.Errorf("failed decoding last join exit calls: %s, %w", c.config.Pair.String(), err)
	}
	var totalSupply = new(big.Int).SetBytes(resp[4])
	var feePercentageCacheSwap = new(big.Int).SetBytes(resp[5])
	var feePercentageCacheYield = new(big.Int).SetBytes(resp[6])
	n := 7
	for i := 0; i < len(c.config.Tokens); i++ {
		var isTokenExempt bool
		if new(big.Int).SetBytes(resp[n]).Cmp(big.NewInt(0)) > 0 {
			isTokenExempt = true
		}
		n++
		c.config.Extra.TokensExemptFromYieldProtocolFee = append(c.config.Extra.TokensExemptFromYieldProtocolFee, isTokenExempt)
	}

	c.config.SwapFeePercentage = swapFeePercentage
	c.config.Extra.AmplificationParameter.Value = amplificationParameter
	c.config.Extra.AmplificationParameter.IsUpdating = isUpdating
	c.config.Extra.AmplificationParameter.Precision = amplificationPrecision
	c.config.Extra.ScalingFactors = scalingFactors
	c.config.Extra.LastJoinExit.LastJoinExitAmplification = lastJoinExitAmplification
	c.config.Extra.LastJoinExit.LastPostJoinExitInvariant = lastPostJoinExitInvariant
	c.config.TotalSupply = totalSupply
	c.config.Extra.ProtocolFeePercentageCacheSwapType = feePercentageCacheSwap
	c.config.Extra.ProtocolFeePercentageCacheYieldType = feePercentageCacheYield
	return nil
}

func (c *ComposableStablePool) CreateTokenRateCacheCalls() ([]types.Call, error) {
	if len(c.config.Tokens) < 1 || len(c.config.Tokens) != len(c.config.RateProviders) {
		return nil, fmt.Errorf("not found tokens in the pool: %s", c.config.Pair.String())
	}

	var calls []types.Call
	for i, token := range c.config.Tokens {
		if token == c.config.ContractAddress || c.config.RateProviders[i] == types.ZeroAddress {
			continue
		}
		// Calls for `getTokenRateCache(token)`
		callData, _ := getTokenRateCache.EncodeArgs(token)
		calls = append(calls, types.Call{
			To:    &c.config.ContractAddress,
			Input: callData,
		})
	}
	return calls, nil
}

func (c *ComposableStablePool) DecodeTokenRateCacheCalls(resp [][]byte) error {
	c.config.Extra.TokenRateCaches = make([]TokenRateCache, len(c.config.Tokens))
	n := 0

	for i, token := range c.config.Tokens {
		if token == c.config.ContractAddress || c.config.RateProviders[i] == types.ZeroAddress {
			continue
		}
		if n >= len(resp) {
			return fmt.Errorf("invalid response for rate cache calls: %s, %d", c.config.Pair.String(), len(resp))
		}
		var rate, oldRate, duration, expires *big.Int
		if err := getTokenRateCache.DecodeValues(resp[n], &rate, &oldRate, &duration, &expires); err != nil {
			return fmt.Errorf("failed decoding token rate cache calls: %s, %w", c.config.Pair.String(), err)
		}
		c.config.Extra.TokenRateCaches[i] = TokenRateCache{
			Rate:     rate,
			OldRate:  oldRate,
			Duration: duration,
			Expires:  expires,
		}
		n++
	}
	return nil
}

func (c *ComposableStablePool) CalcAmountOut(tokenIn ERC20Details, tokenOut ERC20Details, amountIn *big.Int) (*big.Int, *big.Int, error) {
	indexIn := -1
	indexOut := -1
	for i, address := range c.config.Tokens {
		if address == tokenIn.address {
			indexIn = i
		}
		if address == tokenOut.address {
			indexOut = i
		}
	}
	if indexIn < 0 || indexOut < 0 || indexIn == indexOut {
		return nil, nil, fmt.Errorf("not found tokens in %s: %s, %s", c.config.Pair.String(), tokenIn.symbol, tokenOut.symbol)
	}

	var amountOut, feeAmount *big.Int
	var err error
	if tokenIn.address == c.config.ContractAddress || tokenOut.address == c.config.ContractAddress {
		amountOut, feeAmount, err = c.swapWithBptGivenIn(indexIn, indexOut, amountIn)
	} else {
		amountOut, feeAmount, err = c.swapGivenIn(indexIn, indexOut, amountIn)
	}
	return amountOut, feeAmount, err
}

// onRegularSwap implements same functionality with the following url:
// https://github.com/balancer/balancer-v2-monorepo/blob/master/pkg/pool-stable/contracts/ComposableStablePool.sol#L283
func (c *ComposableStablePool) onRegularSwap(
	amountIn *big.Int,
	registeredBalances []*big.Int,
	registeredIndexIn,
	registeredIndexOut int,
) (*big.Int, error) {
	// Adjust indices and balances for BPT token
	// uint256[] memory balances = _dropBptItem(registeredBalances);
	// uint256 indexIn = _skipBptIndex(indexIn);
	// uint256 indexOut = _skipBptIndex(indexOut);

	droppedBalances := c.dropBptItem(registeredBalances)
	indexIn := c.skipBptIndex(registeredIndexIn)
	indexOut := c.skipBptIndex(registeredIndexOut)

	// (uint256 currentAmp, ) = _getAmplificationParameter();
	// uint256 invariant = StableMath._calculateInvariant(currentAmp, balances);
	currentAmp := c.config.Extra.AmplificationParameter.Value
	invariant, err := CalculateInvariant(currentAmp, droppedBalances, false)
	if err != nil {
		return nil, err
	}

	// StableMath._calcOutGivenIn(currentAmp, balances, indexIn, indexOut, amountGiven, invariant);
	return CalcOutGivenIn(currentAmp, droppedBalances, indexIn, indexOut, amountIn, invariant)
}

// onSwapGivenIn implements same functionality with the following url:
// https://github.com/balancer/balancer-v2-monorepo/blob/master/pkg/pool-stable/contracts/ComposableStablePool.sol#L242
func (c *ComposableStablePool) onSwapGivenIn(
	amountIn *big.Int,
	registeredBalances []*big.Int,
	indexIn,
	indexOut int,
) (*big.Int, error) {

	return c.onRegularSwap(amountIn, registeredBalances, indexIn, indexOut)
}

// swapWithBptGivenIn implements same functionality with the following url:
// https://github.com/balancer/balancer-v2-monorepo/blob/master/pkg/pool-stable/contracts/ComposableStablePool.sol#L314
func (c *ComposableStablePool) swapWithBptGivenIn(indexIn, indexOut int, amountIn *big.Int) (*big.Int, *big.Int, error) {
	var amountCalculated, feeAmount *big.Int

	// bool isGivenIn = swapRequest.kind == IVault.SwapKind.GIVEN_IN;
	// _upscaleArray(registeredBalances, scalingFactors);
	// swapRequest.amount = _upscale(
	//	swapRequest.amount,
	//	scalingFactors[isGivenIn ? indexIn : indexOut]
	balancesUpscaled := c.upscaleArray(c.config.Balances, c.config.Extra.ScalingFactors)
	tokenAmountInScaled := c.upscale(amountIn, c.config.Extra.ScalingFactors[indexIn])

	// (
	//	uint256 preJoinExitSupply,
	//	uint256[] memory balances,
	//	uint256 currentAmp,
	//	uint256 preJoinExitInvariant
	// ) = _beforeJoinExit(registeredBalances);
	preJoinExitSupply, balances, currentAmp, preJoinExitInvariant, err := c.beforeJoinExit(balancesUpscaled)
	if err != nil {
		return nil, nil, err
	}
	if indexOut == c.config.BptIndex {
		// _doJoinSwap(
		//	isGivenIn,
		//	swapRequest.amount,
		//	balances,
		//	skipBptIndex(registeredIndexIn),
		//	currentAmp,
		//	preJoinExitSupply,
		//	preJoinExitInvariant
		// )
		amountCalculated, _, feeAmount, err = c.doJoinSwap(
			true,
			tokenAmountInScaled,
			balances,
			c.skipBptIndex(indexIn),
			currentAmp,
			preJoinExitSupply,
			preJoinExitInvariant,
		)
		if err != nil {
			return nil, nil, err
		}
	} else {
		// _doExitSwap(
		//	isGivenIn,
		//	swapRequest.amount,
		//	balances,
		//	skipBptIndex(registeredIndexOut),
		//	currentAmp,
		//	preJoinExitSupply,
		//	preJoinExitInvariant
		// )
		amountCalculated, _, feeAmount, err = c.doExitSwap(
			true,
			tokenAmountInScaled,
			balances,
			c.skipBptIndex(indexOut),
			currentAmp,
			preJoinExitSupply,
			preJoinExitInvariant,
		)
		if err != nil {
			return nil, nil, err
		}
	}
	if amountCalculated == nil {
		return nil, nil, fmt.Errorf("INVALID_AMOUNT_OUT_CALCULATED")
	}
	// _downscaleDown(amountCalculated, scalingFactors[registeredIndexOut]) // Amount out, round down
	return DivDownFixed(amountCalculated, c.config.Extra.ScalingFactors[indexOut]), feeAmount, nil
}

// exitSwapExactBptInForTokenOut implements same functionality with the following url:
// https://github.com/balancer/balancer-v2-monorepo/blob/master/pkg/pool-stable/contracts/ComposableStablePool.sol#L504
func (c *ComposableStablePool) exitSwapExactBptInForTokenOut(
	bptAmount *big.Int,
	balances []*big.Int,
	indexOut int,
	currentAmp *big.Int,
	actualSupply *big.Int,
	preJoinExitInvariant *big.Int,
) (*big.Int, *big.Int, *big.Int, error) {

	amountOut, feeAmount, err := calcTokenOutGivenExactBptIn(
		currentAmp, balances, indexOut, bptAmount, actualSupply, preJoinExitInvariant, c.config.SwapFeePercentage)
	if err != nil {
		return nil, nil, nil, err
	}

	balances[indexOut].Sub(balances[indexOut], amountOut)
	postJoinExitSupply := new(big.Int).Sub(actualSupply, bptAmount)

	return amountOut, postJoinExitSupply, feeAmount, nil
}

// doJoinSwap implements same functionality with the following url:
// https://github.com/balancer/balancer-v2-monorepo/blob/master/pkg/pool-stable/contracts/ComposableStablePool.sol#L375
func (c *ComposableStablePool) doJoinSwap(
	isGivenIn bool,
	amount *big.Int,
	balances []*big.Int,
	indexIn int,
	currentAmp *big.Int,
	actualSupply *big.Int,
	preJoinExitInvariant *big.Int,
) (*big.Int, *big.Int, *big.Int, error) {

	if isGivenIn {
		return c.joinSwapExactTokenInForBptOut(amount, balances, indexIn, currentAmp, actualSupply, preJoinExitInvariant)
	}
	// Currently ignore givenOut case
	return nil, nil, nil, nil
}

// doExitSwap implements same functionality with the following url:
// https://github.com/balancer/balancer-v2-monorepo/blob/master/pkg/pool-stable/contracts/ComposableStablePool.sol#L470
func (c *ComposableStablePool) doExitSwap(
	isGivenIn bool,
	amount *big.Int,
	balances []*big.Int,
	indexOut int,
	currentAmp *big.Int,
	actualSupply *big.Int,
	preJoinExitInvariant *big.Int,
) (*big.Int, *big.Int, *big.Int, error) {

	if isGivenIn {
		return c.exitSwapExactBptInForTokenOut(amount, balances, indexOut, currentAmp, actualSupply, preJoinExitInvariant)
	}
	// Currently ignore givenOut case
	return nil, nil, nil, nil
}

// joinSwapExactTokenInForBptOut implements same functionality with the following url:
// https://github.com/balancer/balancer-v2-monorepo/blob/master/pkg/pool-stable/contracts/ComposableStablePool.sol#L409
func (c *ComposableStablePool) joinSwapExactTokenInForBptOut(
	amountIn *big.Int,
	balances []*big.Int,
	indexIn int,
	currentAmp *big.Int,
	actualSupply *big.Int,
	preJoinExitInvariant *big.Int,
) (*big.Int, *big.Int, *big.Int, error) {

	amountsIn := make([]*big.Int, len(balances))
	for i := range amountsIn {
		amountsIn[i] = new(big.Int)
	}
	amountsIn[indexIn] = amountIn
	bptOut, feeAmountIn, err := calcBptOutGivenExactTokensIn(
		currentAmp, balances, amountsIn, actualSupply, preJoinExitInvariant, c.config.SwapFeePercentage)
	if err != nil {
		return nil, nil, nil, err
	}
	balances[indexIn].Add(balances[indexIn], amountIn)
	postJoinExitSupply := new(big.Int).Add(actualSupply, bptOut)

	return bptOut, postJoinExitSupply, feeAmountIn, nil
}

// beforeJoinExit implements same functionality with the following url:
// https://github.com/balancer/balancer-v2-monorepo/blob/master/pkg/pool-stable/contracts/ComposableStablePool.sol#L701
func (c *ComposableStablePool) beforeJoinExit(registeredBalances []*big.Int) (*big.Int, []*big.Int, *big.Int, *big.Int, error) {
	preJoinExitSupply, balances, oldAmpPreJoinExitInvariant, err := c.payProtocolFeesBeforeJoinExit(registeredBalances)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	currentAmp := c.config.Extra.AmplificationParameter.Value

	var (
		preJoinExitInvariant *big.Int
	)

	if currentAmp.Cmp(c.config.Extra.LastJoinExit.LastJoinExitAmplification) == 0 {
		preJoinExitInvariant = oldAmpPreJoinExitInvariant
	} else {
		preJoinExitInvariant, err = CalculateInvariant(currentAmp, balances, false)
	}
	if err != nil {
		return nil, nil, nil, nil, err
	}

	return preJoinExitSupply, balances, currentAmp, preJoinExitInvariant, nil
}

// payProtocolFeesBeforeJoinExit implements same functionality with the following url:
// https://github.com/balancer/balancer-v2-monorepo/blob/master/pkg/pool-stable/contracts/ComposableStablePoolProtocolFees.sol#L64
func (c *ComposableStablePool) payProtocolFeesBeforeJoinExit(
	registeredBalances []*big.Int,
) (*big.Int, []*big.Int, *big.Int, error) {

	virtualSupply, droppedBalances := c.dropBptItemFromBalances(registeredBalances)
	expectedProtocolOwnershipPercentage, currentInvariantWithLastJoinExitAmp, err := c.getProtocolPoolOwnershipPercentage(droppedBalances)
	if err != nil {
		return nil, nil, nil, err
	}
	protocolFeeAmount := c.bptForPoolOwnershipPercentage(virtualSupply, expectedProtocolOwnershipPercentage)

	return new(big.Int).Add(virtualSupply, protocolFeeAmount), droppedBalances, currentInvariantWithLastJoinExitAmp, nil
}

// getProtocolPoolOwnershipPercentage implements same functionality with the following url:
// https://github.com/balancer/balancer-v2-monorepo/blob/master/pkg/pool-stable/contracts/ComposableStablePoolProtocolFees.sol#L102
func (c *ComposableStablePool) getProtocolPoolOwnershipPercentage(balances []*big.Int) (*big.Int, *big.Int, error) {
	swapFeeGrowthInvariant, totalNonExemptGrowthInvariant, totalGrowthInvariant, err := c.getGrowthInvariants(balances)
	if err != nil {
		return nil, nil, err
	}
	// Calculate the delta for swap fee growth invariant
	swapFeeGrowthInvariantDelta := new(big.Int).Sub(swapFeeGrowthInvariant, c.config.Extra.LastJoinExit.LastPostJoinExitInvariant)
	if swapFeeGrowthInvariantDelta.Cmp(bigIntZero) < 0 {
		swapFeeGrowthInvariantDelta.SetUint64(0)
	}

	// Calculate the delta for non-exempt yield growth invariant
	nonExemptYieldGrowthInvariantDelta := new(big.Int).Sub(totalNonExemptGrowthInvariant, swapFeeGrowthInvariant)
	if nonExemptYieldGrowthInvariantDelta.Cmp(bigIntZero) < 0 {
		nonExemptYieldGrowthInvariantDelta.SetUint64(0)
	}

	// swapFeeGrowthInvariantDelta/totalGrowthInvariant*getProtocolFeePercentageCache
	protocolSwapFeePercentage := MulDownFixed(
		DivDownFixed(swapFeeGrowthInvariantDelta, totalGrowthInvariant),
		c.config.Extra.ProtocolFeePercentageCacheSwapType)

	protocolYieldPercentage := MulDownFixed(
		DivDownFixed(nonExemptYieldGrowthInvariantDelta, totalGrowthInvariant),
		c.config.Extra.ProtocolFeePercentageCacheYieldType)

	// Calculate the total protocol ComposableStablePool ownership percentage
	protocolPoolOwnershipPercentage := new(big.Int).Add(protocolSwapFeePercentage, protocolYieldPercentage)

	return protocolPoolOwnershipPercentage, totalGrowthInvariant, nil
}

// getGrowthInvariants implements same functionality with the following url:
// https://github.com/balancer/balancer-v2-monorepo/blob/master/pkg/pool-stable/contracts/ComposableStablePoolProtocolFees.sol#L189
func (c *ComposableStablePool) getGrowthInvariants(balances []*big.Int) (*big.Int, *big.Int, *big.Int, error) {
	var (
		swapFeeGrowthInvariant        *big.Int
		totalNonExemptGrowthInvariant *big.Int
		totalGrowthInvariant          *big.Int
		err                           error
	)

	// This invariant result is calc by DivDown (round down)
	// https://github.com/balancer/balancer-v2-monorepo/blob/b46023f7c5deefaf58a0a42559a36df420e1639f/pkg/pool-stable/contracts/StableMath.sol#L96
	swapFeeGrowthInvariant, err = CalculateInvariant(
		c.config.Extra.LastJoinExit.LastJoinExitAmplification,
		c.getAdjustedBalances(balances, true), false)
	if err != nil {
		return nil, nil, nil, err
	}

	// For the other invariants, we can potentially skip some work. In the edge cases where none or all of the
	// tokens are exempt from yield, there's one fewer invariant to compute.
	switch {
	case c.areNoTokensExempt():
		// If there are no tokens with fee-exempt yield, then the total non-exempt growth will equal the total
		// growth: all yield growth is non-exempt. There's also no point in adjusting balances, since we
		// already know none are exempt.
		totalNonExemptGrowthInvariant, err = CalculateInvariant(c.config.Extra.LastJoinExit.LastJoinExitAmplification, balances, false)
		if err != nil {
			return nil, nil, nil, err
		}

		totalGrowthInvariant = totalNonExemptGrowthInvariant
	case c.areAllTokensExempt():
		// If no tokens are charged fees on yield, then the non-exempt growth is equal to the swap fee growth - no
		// yield fees will be collected.
		totalNonExemptGrowthInvariant = swapFeeGrowthInvariant
		totalGrowthInvariant, err = CalculateInvariant(c.config.Extra.LastJoinExit.LastJoinExitAmplification, balances, false)
		if err != nil {
			return nil, nil, nil, err
		}
	default:
		// In the general case, we need to calculate two invariants: one with some adjusted balances, and one with
		// the current balances.

		totalNonExemptGrowthInvariant, err = CalculateInvariant(
			c.config.Extra.LastJoinExit.LastJoinExitAmplification,
			c.getAdjustedBalances(balances, false), // Only adjust non-exempt balances
			false,
		)
		if err != nil {
			return nil, nil, nil, err
		}

		totalGrowthInvariant, err = CalculateInvariant(
			c.config.Extra.LastJoinExit.LastJoinExitAmplification,
			balances,
			false)
		if err != nil {
			return nil, nil, nil, err
		}
	}
	return swapFeeGrowthInvariant, totalNonExemptGrowthInvariant, totalGrowthInvariant, nil
}

// dropBptItemFromBalances implements same functionality with the following url:
// https://github.com/balancer/balancer-v2-monorepo/blob/master/pkg/pool-stable/contracts/ComposableStablePoolStorage.sol#L259
func (c *ComposableStablePool) dropBptItemFromBalances(balances []*big.Int) (*big.Int, []*big.Int) {
	return c.getVirtualSupply(balances[c.config.BptIndex]), c.dropBptItem(balances)
}

// getVirtualSupply implements same functionality with the following url:
// https://github.com/balancer/balancer-v2-monorepo/blob/master/pkg/pool-stable/contracts/ComposableStablePoolStorage.sol#L386
func (c *ComposableStablePool) getVirtualSupply(bptBalance *big.Int) *big.Int {
	return new(big.Int).Sub(c.config.TotalSupply, bptBalance)
}

// hasRateProvider implements same functionality with the following url:
// https://github.com/balancer/balancer-v2-monorepo/blob/master/pkg/pool-stable/contracts/ComposableStablePoolStorage.sol#L337
func (c *ComposableStablePool) hasRateProvider(tokenIndex int) bool {
	return c.config.RateProviders[tokenIndex] != types.ZeroAddress
}

// isTokenExemptFromYieldProtocolFee implements same functionality with the following url:
// https://github.com/balancer/balancer-v2-monorepo/blob/master/pkg/pool-stable/contracts/ComposableStablePoolStorage.sol#L362
func (c *ComposableStablePool) isTokenExemptFromYieldProtocolFee(tokenIndex int) bool {
	return c.config.Extra.TokensExemptFromYieldProtocolFee[tokenIndex]
}

// areNoTokensExempt implements same functionality with the following url:
// https://github.com/balancer/balancer-v2-monorepo/blob/master/pkg/pool-stable/contracts/ComposableStablePoolStorage.sol#L351
func (c *ComposableStablePool) areNoTokensExempt() bool {
	for _, exempt := range c.config.Extra.TokensExemptFromYieldProtocolFee {
		if exempt {
			return false
		}
	}
	return true
}

// areAllTokensExempt implements same functionality with the following url:
// https://github.com/balancer/balancer-v2-monorepo/blob/master/pkg/pool-stable/contracts/ComposableStablePoolStorage.sol#L344
func (c *ComposableStablePool) areAllTokensExempt() bool {
	for _, exempt := range c.config.Extra.TokensExemptFromYieldProtocolFee {
		if !exempt {
			return false
		}
	}
	return true
}

// getAdjustedBalances implements same functionality with the following url:
// https://github.com/balancer/balancer-v2-monorepo/blob/master/pkg/pool-stable/contracts/ComposableStablePoolRates.sol#L222
func (c *ComposableStablePool) getAdjustedBalances(balances []*big.Int, ignoreExemptFlags bool) []*big.Int {
	totalTokensWithoutBpt := len(balances)
	adjustedBalances := make([]*big.Int, totalTokensWithoutBpt)

	for i := 0; i < totalTokensWithoutBpt; i++ {
		skipBptIndex := i
		if i >= c.config.BptIndex {
			skipBptIndex++
		}

		if c.isTokenExemptFromYieldProtocolFee(skipBptIndex) || (ignoreExemptFlags && c.hasRateProvider(skipBptIndex)) {
			adjustedBalances[i] = c.adjustedBalance(balances[i], &c.config.Extra.TokenRateCaches[skipBptIndex])
		} else {
			adjustedBalances[i] = balances[i]
		}
	}

	return adjustedBalances
}

// adjustedBalance implements same functionality with the following url:
// https://github.com/balancer/balancer-v2-monorepo/blob/master/pkg/pool-stable/contracts/ComposableStablePoolRates.sol#L242
func (c *ComposableStablePool) adjustedBalance(balance *big.Int, cache *TokenRateCache) *big.Int {
	return DivDown(new(big.Int).Mul(balance, cache.OldRate), cache.Rate)
}

// dropBptItem implements same functionality with the following url:
// https://github.com/balancer/balancer-v2-monorepo/blob/master/pkg/pool-stable/contracts/ComposableStablePoolStorage.sol#L246
func (c *ComposableStablePool) dropBptItem(amounts []*big.Int) []*big.Int {
	amountsWithoutBpt := make([]*big.Int, len(amounts)-1)
	bptIndex := c.config.BptIndex

	for i := 0; i < len(amountsWithoutBpt); i++ {
		if i < bptIndex {
			amountsWithoutBpt[i] = new(big.Int).Set(amounts[i])
		} else {
			amountsWithoutBpt[i] = new(big.Int).Set(amounts[i+1])
		}
	}
	return amountsWithoutBpt
}

func (c *ComposableStablePool) bptForPoolOwnershipPercentage(totalSupply, poolOwnershipPercentage *big.Int) *big.Int {
	// If we mint some amount `bptAmount` of BPT then the percentage ownership of the pool this grants is given by:
	// `poolOwnershipPercentage = bptAmount / (totalSupply + bptAmount)`.
	// Solving for `bptAmount`, we arrive at:
	// `bptAmount = totalSupply * poolOwnershipPercentage / (1 - poolOwnershipPercentage)`.
	return DivDown(new(big.Int).Mul(totalSupply, poolOwnershipPercentage), ComplementFixed(poolOwnershipPercentage))
}

// skipBptIndex implements same functionality with the following url:
// https://github.com/balancer/balancer-v2-monorepo/blob/master/pkg/pool-stable/contracts/ComposableStablePoolStorage.sol#L235
func (c *ComposableStablePool) skipBptIndex(index int) int {
	if index < c.config.BptIndex {
		return index
	}
	return index - 1
}

// swapGivenIn simulates the functionality of `swapGivenIn` in `ComposableStablePool`
// https://github.com/balancer/balancer-v2-monorepo/blob/master/pkg/pool-stable/contracts/ComposableStablePool.sol#L187
func (c *ComposableStablePool) swapGivenIn(indexIn, indexOut int, amountIn *big.Int) (*big.Int, *big.Int, error) {
	// Fees are subtracted before scaling, to reduce the complexity of the rounding direction analysis.
	// swapRequest.amount = subtractSwapFeeAmount(swapRequest.amount);
	amountAfterFee, feeAmount := c.subtractSwapFeeAmount(amountIn, c.config.SwapFeePercentage)

	// upscaleArray(balances, scalingFactors);
	// swapRequest.amount = upscale(swapRequest.amount, scalingFactors[indexIn]);
	upscaledBalances := c.upscaleArray(c.config.Balances, c.config.Extra.ScalingFactors)
	amountUpScale := c.upscale(amountAfterFee, c.config.Extra.ScalingFactors[indexIn])

	// uint256 amountOut = onSwapGivenIn(swapRequest, balances, indexIn, indexOut);
	amountOut, err := c.onSwapGivenIn(amountUpScale, upscaledBalances, indexIn, indexOut)
	if err != nil {
		return nil, nil, err
	}

	return DivDownFixed(amountOut, c.config.Extra.ScalingFactors[indexOut]), feeAmount, nil
}

func (c *ComposableStablePool) subtractSwapFeeAmount(amount, swapFeePercentage *big.Int) (*big.Int, *big.Int) {
	feeAmount := MulUpFixed(amount, swapFeePercentage)
	return new(big.Int).Sub(amount, feeAmount), feeAmount
}

func (c *ComposableStablePool) upscaleArray(amounts, scalingFactors []*big.Int) []*big.Int {
	result := make([]*big.Int, len(amounts))
	for i, amount := range amounts {
		result[i] = MulUpFixed(amount, scalingFactors[i])
	}
	return result
}

func (c *ComposableStablePool) upscale(amount, scalingFactor *big.Int) *big.Int {
	return MulUpFixed(amount, scalingFactor)
}

// calcBptOutGivenExactTokensIn implements same functionality with the following url:
// https://github.com/balancer/balancer-v2-monorepo/blob/master/pkg/pool-stable/contracts/StableMath.sol#L201
func calcBptOutGivenExactTokensIn(
	amp *big.Int,
	balances []*big.Int,
	amountsIn []*big.Int,
	bptTotalSupply, invariant, swapFeePercentage *big.Int,
) (*big.Int, *big.Int, error) {

	feeAmountIn := big.NewInt(0)
	sumBalances := big.NewInt(0)
	for _, balance := range balances {
		sumBalances.Add(sumBalances, balance)
	}

	balanceRatiosWithFee := make([]*big.Int, len(amountsIn))
	invariantRatioWithFees := big.NewInt(0)
	for i, balance := range balances {
		currentWeight := DivDownFixed(balance, sumBalances)
		balanceRatiosWithFee[i] = DivDownFixed(new(big.Int).Add(balance, amountsIn[i]), balance)
		invariantRatioWithFees.Add(invariantRatioWithFees, MulDownFixed(balanceRatiosWithFee[i], currentWeight))
	}

	newBalances := make([]*big.Int, len(balances))
	for i, balance := range balances {
		var amountInWithoutFee *big.Int
		if balanceRatiosWithFee[i].Cmp(invariantRatioWithFees) > 0 {
			nonTaxableAmount := MulDownFixed(balance, new(big.Int).Sub(invariantRatioWithFees, big.NewInt(ether)))
			taxableAmount := new(big.Int).Sub(amountsIn[i], nonTaxableAmount)
			amountInWithoutFee = new(big.Int).Add(
				nonTaxableAmount,
				MulDownFixed(
					taxableAmount,
					new(big.Int).Sub(big.NewInt(ether), swapFeePercentage),
				),
			)
		} else {
			amountInWithoutFee = amountsIn[i]
		}
		feeAmountIn = feeAmountIn.Add(feeAmountIn, new(big.Int).Sub(amountsIn[i], amountInWithoutFee))
		newBalances[i] = new(big.Int).Add(balance, amountInWithoutFee)
	}

	newInvariant, err := CalculateInvariant(amp, newBalances, false)
	if err != nil {
		return nil, nil, err
	}

	invariantRatio := DivDownFixed(newInvariant, invariant)
	if invariantRatio.Cmp(big.NewInt(ether)) > 0 {
		return MulDownFixed(bptTotalSupply, new(big.Int).Sub(invariantRatio, big.NewInt(ether))), feeAmountIn, nil
	}
	return big.NewInt(0), feeAmountIn, nil
}

// calcTokenOutGivenExactBptIn implements same functionality with the following url:
// https://github.com/balancer/balancer-v2-monorepo/blob/master/pkg/pool-stable/contracts/StableMath.sol#L354
func calcTokenOutGivenExactBptIn(
	amp *big.Int,
	balances []*big.Int,
	tokenIndex int,
	bptAmountIn *big.Int,
	bptTotalSupply, invariant, swapFeePercentage *big.Int,
) (*big.Int, *big.Int, error) {

	newInvariant := MulUpFixed(DivUpFixed(new(big.Int).Sub(bptTotalSupply, bptAmountIn), bptTotalSupply), invariant)
	newBalanceTokenIndex, err := getTokenBalanceGivenInvariantAndAllOtherBalances(amp, balances, newInvariant, tokenIndex)
	if err != nil {
		return nil, nil, err
	}
	amountOutWithoutFee := new(big.Int).Sub(balances[tokenIndex], newBalanceTokenIndex)

	sumBalances := big.NewInt(0)
	for _, balance := range balances {
		sumBalances.Add(sumBalances, balance)
	}

	currentWeight := DivDownFixed(balances[tokenIndex], sumBalances)
	taxablePercentage := ComplementFixed(currentWeight)

	taxableAmount := MulUpFixed(amountOutWithoutFee, taxablePercentage)
	nonTaxableAmount := new(big.Int).Sub(amountOutWithoutFee, taxableAmount)

	feeOfTaxableAmount := MulDownFixed(
		taxableAmount,
		new(big.Int).Sub(big.NewInt(ether), swapFeePercentage),
	)

	feeAmount := new(big.Int).Sub(taxableAmount, feeOfTaxableAmount)
	return new(big.Int).Add(nonTaxableAmount, feeOfTaxableAmount), feeAmount, nil
}
