# Syscoin 5 and SPT API Documentation Updates

## Summary of Changes Made to docs/api.md

### 1. Added Introduction for Syscoin 5.0 Features
- Added note about SPT (Syscoin Platform Tokens) support
- Added NEVM (Network-Enhanced Virtual Machine) integration information
- Mentioned cross-chain asset management between UTXO and EVM layers

### 2. New API Endpoints Added

#### Get Asset
- **Endpoint**: `GET /api/v2/asset/<asset GUID>`
- **Purpose**: Returns asset details and transactions for a specific SPT
- **Response**: Includes asset metadata (symbol, decimals, total supply, contract address, etc.)

#### Get Assets
- **Endpoint**: `GET /api/v2/assets/<search filter>`
- **Purpose**: Returns filtered list of assets matching search criteria
- **Response**: Paginated list of assets with basic information

### 3. Enhanced Transaction Responses

#### SPT Transaction Support
- Added example of SPT transaction with `assetInfo` fields in vin/vout
- Included `tokenType` field for SPT transaction types
- Added `memo` field for SPT-specific data
- Transaction version 142 example for SPT allocation send

#### Asset Information in Transactions
- `assetInfo` objects in vin/vout containing:
  - `assetGuid`: Unique identifier for the SPT
  - `value`: Amount of the asset being transferred

### 4. Updated Address Endpoint
- Added SPT token information in address responses
- Example shows `tokens` array with SPT assets
- Token type: "SPTAllocated" 
- Includes asset GUID, symbol, decimals, balances, and transfer counts

### 5. Enhanced Query Parameters

#### AssetMask Parameter
- Updated documentation for Syscoin-specific asset filtering
- Bitmask values for different SPT transaction types:
  - `basecoin`: 1
  - `assetallocationsend`: 2 (SPT allocation send)
  - `syscoinburntoallocation`: 4 (Syscoin burn to SPT allocation)
  - `assetallocationburntosyscoin`: 8 (SPT allocation burn to Syscoin)
  - `assetallocationburntonevm`: 16 (SPT allocation burn to NEVM)
  - `assetallocationmint`: 32 (SPT allocation mint)

### 6. Updated UTXO Endpoint
- Added support for SPT assets in UTXO responses
- `assetInfo` objects in UTXO outputs
- Example showing SPT asset information in unspent outputs

### 7. WebSocket API Updates
- Added `getAsset` and `getAssets` methods (Syscoin only)
- Updated list of available websocket requests

## Key Syscoin 5 Features Documented

### SPT (Syscoin Platform Tokens)
- Native token support on Syscoin UTXO layer
- Cross-chain compatibility with NEVM
- Support for ERC20, ERC721, and ERC1155 token standards
- Comprehensive transaction filtering and querying

### NEVM Integration
- Network-Enhanced Virtual Machine for Ethereum compatibility
- Contract address mapping for SPT assets
- Cross-chain asset transfers between UTXO and EVM layers

### Asset Management
- Comprehensive asset metadata storage
- Transaction history tracking per asset
- Balance and transfer counting
- Search and filtering capabilities

## Implementation Consistency

The API documentation now accurately reflects:
- The RocksDB storage implementation for SPT assets
- NEVM client integration for ERC token details
- Asset cache management for performance
- Transaction type filtering and asset mask support
- Cross-chain asset allocation and burning mechanisms

## Future Considerations

- Monitor syscoinjs-lib updates for client-side compatibility
- Ensure API responses match the latest Syscoin Core 5.0 specifications
- Consider adding more detailed error responses for asset-related operations
- Document any additional NEVM-specific endpoints that may be added