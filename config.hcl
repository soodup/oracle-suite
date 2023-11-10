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
    listen_addrs    = try(split(env.CFG_LIBP2P_LISTEN_ADDRS, ","), ["/ip4/0.0.0.0/tcp/8101"])
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

ghost {
  ethereum_key = "default"
  interval     = 60
  pairs        = [
    "AAVE/USD",
    "AVAX/USD",
    "BAL/USD",
    "BAT/USD",
    "BTC/USD",
    "COMP/USD",
    "CRV/USD",
    "DOT/USD",
    "ETH/BTC",
    "ETH/USD",
    "FIL/USD",
    "GNO/USD",
    "IBTA/USD",
    "LINK/USD",
    "LRC/USD",
    "MANA/USD",
    "MKR/ETH",
    "MKR/USD",
    "PAXG/USD",
    "RETH/USD",
    "SNX/USD",
    "SOL/USD",
    "UNI/USD",
    "USDT/USD",
    "WNXM/USD",
    "XRP/USD",
    "XTZ/USD",
    "YFI/USD",
    "ZEC/USD",
    "ZRX/USD",
    "STETH/USD",
    "WSTETH/USD",
    "MATIC/USD"
  ]
}

gofer {
  origin "openexchangerates" {
    type   = "openexchangerates"
    params = {
      api_key = try(env.GOFER_OPENEXCHANGERATES_API_KEY, "")
    }
  }

  origin "upshot" {
    type   = "upshot"
    params = {
      api_key = try(env.GOFER_UPSHOT_API_KEY, "UP-0d9ed54694abdac60fd23b74")
    }
  }

  price_model "ETH/USD" "median" {
    source "ETH/USD" "origin" { origin = "openexchangerates" }
    min_sources = 1
  }

  price_model "cryptopunks/appraisal" "median" {
    source "cryptopunks/appraisal" "origin" { origin = "upshot" }
    min_sources = 1
  }

  }

leeloo {
  ethereum_key = "default"

  # Arbitrum
  # Enabled if CFG_TELEPORT_EVM_ARB_CONTRACT_ADDRS is set.
  dynamic "teleport_evm" {
    for_each = try(env.CFG_TELEPORT_EVM_ARB_CONTRACT_ADDRS, "") == "" ? [] : [1]
    content {
      ethereum_client     = "arbitrum"
      interval            = try(parseint(env.CFG_TELEPORT_EVM_ARB_INTERVAL, 10), 60)
      prefetch_period     = try(parseint(env.CFG_TELEPORT_EVM_ARB_PREFETCH_PERIOD, 10), 3600 * 24 * 7)
      block_confirmations = try(parseint(env.CFG_TELEPORT_EVM_ARB_BLOCK_CONFIRMATIONS, 10), 0)
      block_limit         = try(parseint(env.CFG_TELEPORT_EVM_ARB_BLOCK_LIMIT, 10), 1000)
      replay_after        = concat(
        [60, 300, 3600, 3600*2, 3600*4],
        [for i in range(3600 * 6, 3600 * 24 * 7, 3600 * 6) :i]
      )
      contract_addrs = try(split(",", env.CFG_TELEPORT_EVM_ARB_CONTRACT_ADDRS), [])
    }
  }

  # Optimism
  # Enabled if CFG_TELEPORT_EVM_OPT_CONTRACT_ADDRS is set.
  dynamic "teleport_evm" {
    for_each = try(env.CFG_TELEPORT_EVM_OPT_CONTRACT_ADDRS, "") == "" ? [] : [1]
    content {
      ethereum_client     = "optimism"
      interval            = try(parseint(env.CFG_TELEPORT_EVM_OPT_INTERVAL, 10), 60)
      prefetch_period     = try(parseint(env.CFG_TELEPORT_EVM_OPT_PREFETCH_PERIOD, 10), 3600 * 24 * 7)
      block_confirmations = try(parseint(env.CFG_TELEPORT_EVM_OPT_BLOCK_CONFIRMATIONS, 10), 0)
      block_limit         = try(parseint(env.CFG_TELEPORT_EVM_OPT_BLOCK_LIMIT, 10), 1000)
      replay_after        = concat(
        [60, 300, 3600, 3600*2, 3600*4],
        [for i in range(3600 * 6, 3600 * 24 * 7, 3600 * 6) :i]
      )
      contract_addrs = try(split(",", env.CFG_TELEPORT_EVM_OPT_CONTRACT_ADDRS), [])
    }
  }

  # Starknet
  # Enabled if CFG_TELEPORT_STARKNET_CONTRACT_ADDRS is set.
  dynamic "teleport_starknet" {
    for_each = try(env.CFG_TELEPORT_STARKNET_CONTRACT_ADDRS, "") == "" ? [] : [1]
    content {
      sequencer       = try(env.CFG_TELEPORT_STARKNET_SEQUENCER, "https://alpha-mainnet.starknet.io")
      interval        = try(parseint(env.CFG_TELEPORT_STARKNET_INTERVAL, 10), 60)
      prefetch_period = try(parseint(env.CFG_TELEPORT_STARKNET_PREFETCH_PERIOD, 10), 3600 * 24 * 7)
      replay_after    = concat(
        [60, 300, 3600, 3600*2, 3600*4],
        [for i in range(3600 * 6, 3600 * 24 * 7, 3600 * 6) :i]
      )
      contract_addrs = try(split(",", env.CFG_TELEPORT_STARKNET_CONTRACT_ADDRS), [])
    }
  }
}

lair {
  listen_addr = try(env.CFG_LAIR_LISTEN_ADDR, "0.0.0.0:8082")

  # Configuration for memory storage. Enabled if CFG_LAIR_STORAGE is "memory" or unset.
  dynamic "storage_memory" {
    for_each = try(env.CFG_LAIR_STORAGE, "memory") == "memory" ? [1] : []
    content {}
  }

  # Configuration for redis storage. Enabled if CFG_LAIR_STORAGE is "redis".
  dynamic "storage_redis" {
    for_each = try(env.CFG_LAIR_STORAGE, "") == "redis" ? [1] : []
    content {
      addr                     = try(env.CFG_LAIR_REDIS_ADDR, "127.0.0.1:6379")
      user                     = try(env.CFG_LAIR_REDIS_USER, "")
      pass                     = try(env.CFG_LAIR_REDIS_PASS, "")
      db                       = try(parseint(env.CFG_LAIR_REDIS_DB, 10), 0)
      memory_limit             = try(parseint(env.CFG_LAIR_REDIS_MEMORY_LIMIT, 10), 0)
      tls                      = try(env.CFG_LAIR_REDIS_TLS == "true", false)
      tls_server_name          = try(env.CFG_LAIR_REDIS_TLS_SERVER_NAME, "")
      tls_cert_file            = try(env.CFG_LAIR_REDIS_TLS_CERT_FILE, "")
      tls_key_file             = try(env.CFG_LAIR_REDIS_TLS_KEY_FILE, "")
      tls_root_ca_file         = try(env.CFG_LAIR_REDIS_TLS_ROOT_CA_FILE, "")
      tls_insecure_skip_verify = try(env.CFG_LAIR_REDIS_TLS_INSECURE == "true", false)
      cluster                  = try(env.CFG_LAIR_REDIS_CLUSTER == "true", false)
      cluster_addrs            = try(split(env.CFG_LAIR_REDIS_CLUSTER_ADDRS, ","), [])
    }
  }
}
