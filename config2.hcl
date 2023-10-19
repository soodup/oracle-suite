variables {
  # List of feeds that are allowed to send price updates and event attestations.
  feeds = [
#    "0x16655369eb59f3e1cafbcfac6d3dd4001328f747",
#    "0xfadad77b3a7e5a84a1f7ded081e785585d4ffaf3",
#    "0x3980aa37f838bec2e457445d943feb3af98ff036",
    "0x2d800d93b065ce011af83f316cef9f0d005b0aa4",
    "0xe3ced0f62f7eb2856d37bed128d2b195712d2644"
  ]
}

ethereum {
  rand_keys = try(env.CFG_ETH_FROM, "") == "" ? ["default"] : []

  dynamic "key" {
    for_each = try(env.CFG_ETH_FROM, "") == "" ? [] : [1]
    labels   = ["default"]
    content {
      address         = try(env.CFG_ETH_FROM, "")
      keystore_path   = try(env.CFG_ETH_KEYS, "")
      passphrase_file = try(env.CFG_ETH_PASS, "")
    }
  }

  client "default" {
    rpc_urls     = try(split(env.CFG_ETH_RPC_URLS, ","), ["https://eth.public-rpc.com"])
    chain_id     = try(parseint(env.CFG_ETH_CHAIN_ID, 10), 1)
    ethereum_key = "default"
  }

  client "arbitrum" {
    rpc_urls     = try(split(env.CFG_ETH_ARB_RPC_URLS, ","), ["https://arbitrum.public-rpc.com"])
    chain_id     = try(parseint(env.CFG_ETH_ARB_CHAIN_ID, 10), 42161)
    ethereum_key = "default"
  }

  client "optimism" {
    rpc_urls     = try(split(env.CFG_ETH_OPT_RPC_URLS, ","), ["https://mainnet.optimism.io"])
    chain_id     = try(parseint(env.CFG_ETH_OPT_CHAIN_ID, 10), 10)
    ethereum_key = "default"
  }
}

transport {
  # LibP2P transport configuration. Always enabled.
  libp2p {
    feeds           = var.feeds
    priv_key_seed   = try(env.CFG_LIBP2P_PK_SEED, "")
    listen_addrs    = try(split(env.CFG_LIBP2P_LISTEN_ADDRS, ","), ["/ip4/0.0.0.0/tcp/8102"])
    bootstrap_addrs = try(split(env.CFG_LIBP2P_BOOTSTRAP_ADDRS, ","), [
      "/dns/spire-bootstrap1.makerops.services/tcp/8000/p2p/12D3KooWRfYU5FaY9SmJcRD5Ku7c1XMBRqV6oM4nsnGQ1QRakSJi",
      "/dns/spire-bootstrap2.makerops.services/tcp/8000/p2p/12D3KooWBGqjW4LuHUoYZUhbWW1PnDVRUvUEpc4qgWE3Yg9z1MoR"
    ])
    direct_peers_addrs = try(split(env.CFG_LIBP2P_DIRECT_PEERS_ADDRS, ","), [])
    blocked_addrs      = try(split(env.CFG_LIBP2P_BLOCKED_ADDRS, ","), [])
    disable_discovery  = try(env.CFG_LIBP2P_DISABLE_DISCOVERY, "false") == "true"
    ethereum_key       = try(env.CFG_ETH_FROM, "") == "" ? "" : "default"
  }

  # WebAPI transport configuration. Enabled if CFG_WEBAPI_LISTEN_ADDR is set to a listen address.
  dynamic "webapi" {
    for_each = try(env.CFG_WEBAPI_LISTEN_ADDR, "") == "" ? [] : [1]
    content {
      feeds             = var.feeds
      listen_addr       = try(env.CFG_WEBAPI_LISTEN_ADDR, "0.0.0.0.8080")
      socks5_proxy_addr = try(env.CFG_WEBAPI_SOCKS5_PROXY_ADDR, "127.0.0.1:9050")

      # Ethereum based address book. Enabled if CFG_WEBAPI_ETH_ADDR_BOOK is set to a contract address.
      dynamic "ethereum_address_book" {
        for_each = try(env.CFG_WEBAPI_ETH_ADDR_BOOK, "") == "" ? [] : [1]
        content {
          contract_addr   = try(env.CFG_WEBAPI_ETH_ADDR_BOOK, "")
          ethereum_client = "default"
        }
      }

      # Static address book. Enabled if CFG_WEBAPI_STATIC_ADDR_BOOK is set to a comma separated list of addresses.
      dynamic "static_address_book" {
        for_each = try(env.CFG_WEBAPI_STATIC_ADDR_BOOK, "") == "" ? [] : [1]
        content {
          addresses = try(split(env.CFG_WEBAPI_STATIC_ADDR_BOOK, ","), "")
        }
      }
    }
  }
}

spire {
  rpc_listen_addr = try(env.CFG_SPIRE_RPC_ADDR, "127.0.0.1:9100")
  rpc_agent_addr  = try(env.CFG_SPIRE_RPC_ADDR, "127.0.0.1:9100")

  # List of pairs that are collected by the spire node. Other pairs are ignored.
  pairs = [
    "BTCUSD",
    "ETHBTC",
    "ETHUSD"
  ]
}