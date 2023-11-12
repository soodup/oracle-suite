# Gofer Dev Readme

An Origin is defined as a source in gofer where it fetches a price from. 

A price model is a feed which can have multiple origins to fetch its price and the result price will be the median of these.

For example, an appraisal of cryptopunks would be a feed and the origin would be upshot, reservoir etc.

## How to add a new origin/feed?

1.) Add a new “origin” and “price_model” block config in `config.hcl`

For example, the below block adds a new origin "upshot" and a feed (cryptopunks appraisal) which will fetch from only upshot origin.
```hcl
gofer {
  origin "upshot" {
    type   = "upshot"
    params = {
      api_key = try(env.GOFER_UPSHOT_API_KEY, "UP-0d9ed54694abdac60fd23b74")
    }
  }

  price_model "cryptopunks/appraisal" "median" {
    source "cryptopunks/appraisal" "origin" { origin = "upshot" }
    min_sources = 1
  }
}
```


2.) Add logic to fetch from the new config block in `pkg/config/priceprovider/origin.go`

```go
case "upshot":
	baseURL := parseSingleParam(params, "url")
	apiKey := parseSingleParam(params, "api_key")
	return origins.NewBaseExchangeHandler(
		origins.IUpshot{WorkerPool: wp, BaseURL: baseURL, APIKey: apiKey},
		aliases,
	), nil
```

3.) Add valid path to the handler file in `pkg/price/provider/origins/origin.go`

```go
func DefaultOriginSet(pool query.WorkerPool) *Set {
	return NewSet(map[string]Handler{
		"upshot":        NewBaseExchangeHandler(IUpshot{WorkerPool: pool}, nil),
	})
}
```


4.) Create a handler file for the logic to fetch the api from the new origin

    For example, `pkg/price/provider/origins/upshot.go`


5.) Release a new tag to use it in omnia-feed and omnia-relay service


Sample PR -> https://github.com/soodup/oracle-suite/compare/e4314aeaec5686f040aebf9e5b1a02de44fc092f...soodup:oracle-suite:upshot



Gofer Supported origins:
- `upshot` - [Upshot](https://docs.upshot.xyz/reference/)

