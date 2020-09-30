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

//TODO: check if it's possible to merge coinbase and coinbasepro
//nolint:dupl
package origins

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/makerdao/gofer/internal/query"
)

// Coinbase URL
const coinbaseURL = "https://api.gdax.com/products/%s/ticker"

type coinbaseResponse struct {
	Price  string `json:"price"`
	Ask    string `json:"ask"`
	Bid    string `json:"bid"`
	Volume string `json:"volume"`
}

// Coinbase origin handler
type Coinbase struct {
	Pool query.WorkerPool
}

// TODO: We should find better solution for this.
func (c *Coinbase) renameSymbol(symbol string) string {
	if symbol == "USD" {
		return "USDC"
	}
	return symbol
}

func (c *Coinbase) localPairName(pair Pair) string {
	return fmt.Sprintf("%s-%s", c.renameSymbol(pair.Base), c.renameSymbol(pair.Quote))
}

func (c *Coinbase) getURL(pair Pair) string {
	return fmt.Sprintf(coinbaseURL, c.localPairName(pair))
}

func (c *Coinbase) Fetch(pairs []Pair) []FetchResult {
	return callSinglePairOrigin(c, pairs)
}

func (c *Coinbase) callOne(pair Pair) (*Tick, error) {
	var err error
	req := &query.HTTPRequest{
		URL: c.getURL(pair),
	}

	// make query
	res := c.Pool.Query(req)
	if res == nil {
		return nil, errEmptyOriginResponse
	}
	if res.Error != nil {
		return nil, res.Error
	}
	// parsing JSON
	var resp coinbaseResponse
	err = json.Unmarshal(res.Body, &resp)
	if err != nil {
		return nil, fmt.Errorf("failed to parse coinbase response: %w", err)
	}
	// Parsing price from string
	price, err := strconv.ParseFloat(resp.Price, 64)
	if err != nil {
		return nil, fmt.Errorf("failed to parse price from coinbase origin %s", res.Body)
	}
	// Parsing ask from string
	ask, err := strconv.ParseFloat(resp.Ask, 64)
	if err != nil {
		return nil, fmt.Errorf("failed to parse ask from coinbase origin %s", res.Body)
	}
	// Parsing volume from string
	volume, err := strconv.ParseFloat(resp.Volume, 64)
	if err != nil {
		return nil, fmt.Errorf("failed to parse volume from coinbase origin %s", res.Body)
	}
	// Parsing bid from string
	bid, err := strconv.ParseFloat(resp.Bid, 64)
	if err != nil {
		return nil, fmt.Errorf("failed to parse bid from coinbase origin %s", res.Body)
	}
	// building Tick
	return &Tick{
		Pair:      pair,
		Price:     price,
		Volume24h: volume,
		Ask:       ask,
		Bid:       bid,
		Timestamp: time.Now(),
	}, nil
}
