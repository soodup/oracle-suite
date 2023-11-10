package origins

import (
	"encoding/json"
	"fmt"
	"github.com/chronicleprotocol/oracle-suite/pkg/util/query"
	"strconv"
	"time"
)

const upshotBaseURL = "https://api.upshot.xyz"
const collectionAppraisalURL = "%s/v2/appraisals/collections?collection_id_or_slugs=%s" //nolint:lll

type upshotResponseData struct {
	Collection   string       `json:"slug"`
	AvgAppraisal AvgAppraisal `json:"avg_appraisal"`
}

type AvgAppraisal struct {
	Wei       string `json:"wei"`
	Timestamp string `json:"timestamp"`
}

type upshotResponse struct {
	Data []upshotResponseData `json:"data"`
}

// Origin handler
type IUpshot struct {
	WorkerPool query.WorkerPool
	BaseURL    string
	APIKey     string
}

func (o IUpshot) localPairName(pair Pair) string {
	return pair.Base + pair.Quote
}

func (o IUpshot) getURL(pair Pair) string {
	if pair.Quote == "appraisal" {
		return buildOriginURL(collectionAppraisalURL, o.BaseURL, upshotBaseURL, pair.Base)
	}
	return buildOriginURL(collectionAppraisalURL, o.BaseURL, upshotBaseURL, o.localPairName(pair))
}

func (o IUpshot) Pool() query.WorkerPool {
	return o.WorkerPool
}

func (o IUpshot) PullPrices(pairs []Pair) []FetchResult {
	return callSinglePairOrigin(&o, pairs)
}

func (o *IUpshot) callOne(pair Pair) (*Price, error) {
	var err error
	req := &query.HTTPRequest{
		URL: o.getURL(pair),
		Headers: map[string]string{
			"x-api-key": o.APIKey,
			"Accept":    "application/json",
		},
	}

	// make query
	res := o.Pool().Query(req)
	if res == nil {
		return nil, ErrEmptyOriginResponse
	}
	if res.Error != nil {
		return nil, res.Error
	}
	// parsing JSON
	var resp upshotResponse
	err = json.Unmarshal(res.Body, &resp)
	if err != nil {
		return nil, fmt.Errorf("failed to parse upshot response: %w", err)
	}

	// building Price
	if len(resp.Data) != 1 {
		return nil, ErrMissingResponseForPair
	}

	data := resp.Data[0]

	wei, err := strconv.ParseFloat(data.AvgAppraisal.Wei, 64)
	if err != nil {
		return nil, fmt.Errorf("failed to parse price from upshot origin %s", res.Body)
	}

	//timestamp, err := strconv.ParseInt(data.AvgAppraisal.Timestamp, 10, 64)
	//if err != nil {
	//	return nil, fmt.Errorf("failed to parse timestamp from upshot origin %s", res.Body)
	//}
	return &Price{
		Pair:      pair,
		Price:     wei,
		Timestamp: time.Now(),
	}, nil
}
