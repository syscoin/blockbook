# Syscoin API Notes

Syscoin uses the standard Blockbook API V2 shape for base coin transactions and
extends it only where Syscoin Platform Token (SPT), NEVM bridge, or Syscoin Core
RPC behavior differs from ordinary Bitcoin-like chains.

## Transactions and UTXOs

SPT-aware inputs, outputs, and UTXOs include `assetInfo` when the indexed value
belongs to an asset allocation:

```json
{
  "assetInfo": {
    "assetGuid": "123456",
    "value": "100000000",
    "valueStr": "1.00000000"
  }
}
```

`assetGuid` is the decimal Syscoin asset GUID. `value` is the raw integer asset
amount in the asset base unit. `valueStr` is formatted using the chain parser
decimal rules.

Transaction responses may also include:

- `tokenType`: the Syscoin SPT transaction class derived from the transaction
  version, such as `SPT`, `SPTAssetAllocationSend`,
  `SPTAssetAllocationMint`, `SPTSyscoinBurnToAllocation`,
  `SPTAssetAllocationBurnToSyscoin`, or `SPTAssetAllocationBurnToNEVM`.
- `memo`: the raw Syscoin memo bytes when present.

These fields appear on existing transaction endpoints, including
`/api/v2/tx/{txid}` and address transaction lists. They are omitted for ordinary
base coin transactions.

## Address Responses

Address responses retain the standard Blockbook `tokens` array for regular
token-like metadata and add Syscoin SPT fields where applicable:

- `usedAssetTokens`: number of SPT assets historically used by the address.
- `tokensAsset`: SPT asset balances and metadata associated with the address.
- `assetGuid`: the SPT asset GUID on token entries where the token represents a
  Syscoin asset.
- `unconfirmedBalanceSat`: unconfirmed SPT balance delta when mempool data is
  available.

The internal address filter supports a Syscoin asset transaction bitmask:

- `0`: all Syscoin/base coin transaction classes.
- `1`: base coin only.
- `2`: asset allocation send.
- `4`: SYS burn to asset allocation.
- `8`: asset allocation burn to SYS.
- `16`: asset allocation burn to NEVM.
- `32`: asset allocation mint.

## Sending Raw Transactions

Syscoin accepts the standard raw transaction body and also JSON payloads with
Syscoin Core's optional burn and fee controls:

```json
{
  "hex": "<raw transaction hex>",
  "maxfeerate": "0.10",
  "maxburnamount": "150"
}
```

The same options are accepted as query/form fields on send endpoints and as
WebSocket fields on `sendTransaction`:

```json
{
  "id": "1",
  "method": "sendTransaction",
  "params": {
    "hex": "<raw transaction hex>",
    "maxfeerate": "0.10",
    "maxburnamount": "150"
  }
}
```

`maxburnamount` is required for transactions that intentionally burn SYS, such
as governance proposal collateral. If omitted, Syscoin Core's default burn
allowance is used.

## Validation

The Syscoin integration test matrix enables RPC, sync, and HTTP connectivity
checks for `syscoin` and `syscoin_testnet`. Live validation should be run
against Syscoin Core 5.1+ with `txindex=1`, ZMQ hashblock/hashtx enabled, and
the generated `blockchaincfg.json` for the matching network.
