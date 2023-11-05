package origins

import (
	"fmt"
	"os"
	"testing"

	"github.com/chronicleprotocol/oracle-suite/pkg/util/query"

	"github.com/stretchr/testify/suite"
)

type UpshotSuite struct {
	suite.Suite
	origin *BaseExchangeHandler
}

func (suite *UpshotSuite) Origin() Handler {
	return suite.origin
}

func (suite *UpshotSuite) SetupSuite() {
	suite.origin = NewBaseExchangeHandler(
		IUpshot{WorkerPool: query.NewMockWorkerPool(), APIKey: "UP-0d9ed54694abdac60fd23b74"},
		nil,
	)
}

func (suite *UpshotSuite) TestLocalPair() {
	ex := suite.origin.ExchangeHandler.(IUpshot)
	suite.EqualValues("cryptopunks", ex.localPairName(Pair{Base: "cryptopunks", Quote: ""}))
	suite.EqualValues("BAYC", ex.localPairName(Pair{Base: "BAYC", Quote: ""}))
}

func (suite *UpshotSuite) TestFailOnWrongInput() {
	pair := Pair{Base: "BTC", Quote: "ETH"}

	// Wrong pair
	fr := suite.origin.Fetch([]Pair{{}})
	suite.Error(fr[0].Error)

	// Nil as a response
	fr = suite.origin.Fetch([]Pair{pair})
	suite.Equal(ErrEmptyOriginResponse, fr[0].Error)

	// Error in a response
	ourErr := fmt.Errorf("error")
	resp := &query.HTTPResponse{
		Error: ourErr,
	}

	suite.origin.ExchangeHandler.(IUpshot).Pool().(*query.MockWorkerPool).MockResp(resp)
	fr = suite.origin.Fetch([]Pair{pair})
	suite.Equal(ourErr, fr[0].Error)

	// Error during unmarshalling
	resp = &query.HTTPResponse{
		Body: []byte(""),
	}
	suite.origin.ExchangeHandler.(IUpshot).Pool().(*query.MockWorkerPool).MockResp(resp)
	fr = suite.origin.Fetch([]Pair{pair})
	suite.Error(fr[0].Error)

	// Error during converting price to a number
	resp = &query.HTTPResponse{
		Body: []byte(`
			{
				"code":"0",
				"msg":"",
				"data":[
				 {
						"instType":"SWAP",
						"instId":"BTC-ETH-SWAP",
						"last":"abcd",
						"askPx":"9999.99",
						"bidPx":"8888.88",
						"vol24h":"2222",
						"ts":"1597026383085"
					}
				]
			}
		`),
	}
	suite.origin.ExchangeHandler.(IUpshot).Pool().(*query.MockWorkerPool).MockResp(resp)
	fr = suite.origin.Fetch([]Pair{pair})
	suite.Error(fr[0].Error)
}

func (suite *UpshotSuite) TestSuccessResponse() {
	pairCryptopunks := Pair{Base: "cryptopunks", Quote: ""}

	resp := &query.HTTPResponse{
		Body: []byte(`
			{
			  "request_id": "a6105f33-21c3-447e-b55d-44d0defa2dd0",
			  "status": true,
			  "data": [
				{
				  "id": "0xb47e3cd837ddf8e4c57f05d70ab865de6e193bbb",
				  "slug": "cryptopunks",
				  "avg_appraisal": {
					"wei": "66202827864612675584",
					"timestamp": "1699061101",
					"change": {
					  "wei_1d": 0,
					  "wei_7d": -28.01,
					  "wei_30d": -33.22
					}
				  }
				}
			  ]
			}
		`),
	}
	suite.origin.ExchangeHandler.(IUpshot).Pool().(*query.MockWorkerPool).MockResp(resp)
	fr := suite.origin.Fetch([]Pair{pairCryptopunks})

	suite.Len(fr, 1)

	suite.NoError(fr[0].Error)
	suite.Equal(pairCryptopunks, fr[0].Price.Pair)
	suite.Equal(66202, fr[0].Price.Price)
	suite.Greater(fr[0].Price.Timestamp.Unix(), int64(0))
}

func (suite *UpshotSuite) TestRealAPICall() {
	os.Setenv("GOFER_TEST_API_CALLS", "1")
	testRealBatchAPICall(
		suite,
		NewBaseExchangeHandler(IUpshot{
			WorkerPool: query.NewHTTPWorkerPool(1),
			APIKey:     "UP-0d9ed54694abdac60fd23b74",
		}, nil),
		[]Pair{
			{Base: "cryptopunks", Quote: "xyz"},
		},
	)
}

func TestIUpshotSuite(t *testing.T) {
	suite.Run(t, new(UpshotSuite))
}
