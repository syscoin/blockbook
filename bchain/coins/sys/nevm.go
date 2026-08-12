package syscoin

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"strings"
	"time"
	"unicode"

	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/golang/glog"
	"github.com/syscoin/syscoinwire/syscoin/wire"
	"github.com/trezor/blockbook/bchain"
	"github.com/trezor/blockbook/bchain/coins/btc"
)

type bridgeAssetMetadata struct {
	Description    string `json:"description"`
	AssetType      string `json:"assetType"`
	OriginDecimals *uint8 `json:"originDecimals,omitempty"`
	TokenID        string `json:"tokenId,omitempty"`
}

func encodeBridgeAssetMetadata(metadata bridgeAssetMetadata) ([]byte, error) {
	return json.Marshal(metadata)
}

const (
	// Syscoin 5 Bridge V2 registry. Legacy pre-cutover SPT metadata is not indexed.
	vaultManagerAddress       = "0x28bD37C0926575f2568ea8f297c0745EF16174Ab"
	defaultNEVMRPCTimeoutSec  = 15
	maxBridgeTokenSymbolRunes = 32
)

func sanitizeBridgeTokenSymbol(symbol string) string {
	sanitized := strings.Map(func(r rune) rune {
		if unicode.IsControl(r) || unicode.In(r, unicode.Cf) {
			return -1
		}
		return r
	}, symbol)
	sanitized = strings.TrimSpace(sanitized)
	runes := []rune(sanitized)
	if len(runes) > maxBridgeTokenSymbolRunes {
		runes = runes[:maxBridgeTokenSymbolRunes]
	}
	return string(runes)
}

type NEVMClient struct {
	rpcClient    *ethclient.Client
	backupClient *ethclient.Client
	vaultAddr    common.Address
	vaultABI     abi.ABI
	tokenABI     abi.ABI
	explorerURL  string
	timeout      time.Duration
}

// NewNEVMClient initializes the primary and backup RPC clients.
func NewNEVMClient(c *btc.Configuration) (*NEVMClient, error) {
	mainClient, err := ethclient.Dial(c.Web3RPCURL)
	if err != nil {
		return nil, err
	}

	var backupClient *ethclient.Client
	if c.Web3RPCURLBackup != "" {
		backupClient, err = ethclient.Dial(c.Web3RPCURLBackup)
		if err != nil {
			// Log backup client error but do NOT close main client or return err.
			glog.Warning("Backup RPC failed to connect: ", err)
			backupClient = nil // Explicitly set to nil for clarity.
		}
	}

	vaultABI, err := abi.JSON(strings.NewReader(vaultABIJSON))
	if err != nil {
		mainClient.Close()
		if backupClient != nil {
			backupClient.Close()
		}
		return nil, err
	}

	tokenABI, err := abi.JSON(strings.NewReader(`
		[{"constant":true,"inputs":[],"name":"symbol","outputs":[{"name":"","type":"string"}],"type":"function"}]
	`))
	if err != nil {
		mainClient.Close()
		if backupClient != nil {
			backupClient.Close()
		}
		return nil, err
	}
	timeout := time.Duration(c.RPCTimeout) * time.Second
	if timeout <= 0 {
		timeout = defaultNEVMRPCTimeoutSec * time.Second
	}

	return &NEVMClient{
		rpcClient:    mainClient,
		backupClient: backupClient,
		vaultAddr:    common.HexToAddress(vaultManagerAddress),
		vaultABI:     vaultABI,
		tokenABI:     tokenABI,
		explorerURL:  c.Web3Explorer,
		timeout:      timeout,
	}, nil
}

// Close closes both RPC connections.
func (c *NEVMClient) Close() {
	if c.rpcClient != nil {
		c.rpcClient.Close()
	}
	if c.backupClient != nil {
		c.backupClient.Close()
	}
}

// callContract attempts the call with primary RPC, falling back to backup if needed.
func (c *NEVMClient) callContract(msg ethereum.CallMsg) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
	res, err := c.rpcClient.CallContract(ctx, msg, nil)
	cancel()
	if err == nil {
		return res, nil
	}

	// If backup RPC exists, attempt fallback
	if c.backupClient != nil {
		ctx, cancel = context.WithTimeout(context.Background(), c.timeout)
		defer cancel()
		return c.backupClient.CallContract(ctx, msg, nil)
	}

	return nil, err
}

// Update existing methods to use callContract:
func (c *NEVMClient) getRealTokenId(assetId uint32, tokenIdx uint32) (*big.Int, error) {
	data, err := c.vaultABI.Pack("getRealTokenIdFromTokenIdx", assetId, tokenIdx)
	if err != nil {
		return nil, err
	}

	callMsg := ethereum.CallMsg{To: &c.vaultAddr, Data: data}
	res, err := c.callContract(callMsg)
	if err != nil {
		return nil, err
	}

	unpacked, err := c.vaultABI.Unpack("getRealTokenIdFromTokenIdx", res)
	if err != nil || len(unpacked) == 0 {
		return nil, err
	}

	return unpacked[0].(*big.Int), nil
}

func (c *NEVMClient) getTokenSymbol(contractAddr common.Address) (string, error) {
	data, err := c.tokenABI.Pack("symbol")
	if err != nil {
		return "", err
	}

	callMsg := ethereum.CallMsg{To: &contractAddr, Data: data}
	res, err := c.callContract(callMsg)
	if err != nil {
		return "", err
	}

	unpacked, err := c.tokenABI.Unpack("symbol", res)
	if err != nil || len(unpacked) == 0 {
		return "", err
	}

	return sanitizeBridgeTokenSymbol(unpacked[0].(string)), nil
}

func (c *NEVMClient) FetchNEVMAssetDetails(assetGuid uint64) (*bchain.Asset, error) {
	if assetGuid == 123456 {
		metadata, err := encodeBridgeAssetMetadata(bridgeAssetMetadata{
			Description: "Syscoin Native Asset",
			AssetType:   "SYSX",
		})
		if err != nil {
			return nil, err
		}
		return &bchain.Asset{
			Transactions: 0,
			AssetObj: wire.AssetType{
				Contract:    []byte{},
				Symbol:      []byte("SYSX"),
				Precision:   8,
				TotalSupply: 0,
				MaxSupply:   0,
			},
			MetaData: metadata,
		}, nil
	}

	assetId := uint32(assetGuid & 0xffffffff)
	tokenIdx := uint32(assetGuid >> 32)

	data, err := c.vaultABI.Pack("assetRegistry", assetId)
	if err != nil {
		return nil, err
	}
	glog.Infof("Calling vaultManager for assetId: %d with data: %x", assetId, data)
	callMsg := ethereum.CallMsg{To: &c.vaultAddr, Data: data}
	res, err := c.callContract(callMsg)
	if err != nil {
		return nil, err
	}

	var registry struct {
		AssetType     uint8
		AssetContract common.Address
		Precision     uint8
		TokenIdCount  uint32
	}

	err = c.vaultABI.UnpackIntoInterface(&registry, "assetRegistry", res)
	if err != nil {
		return nil, err
	}

	var symbol string
	metadata := bridgeAssetMetadata{}
	contractAddr := registry.AssetContract
	precision := registry.Precision
	switch registry.AssetType {
	case 2: // ERC20
		symbol, err = c.getTokenSymbol(contractAddr)
		if err != nil || symbol == "" {
			symbol = fmt.Sprintf("ERC20-%d", assetId)
		}
		metadata.Description = "ERC20 Token"
		metadata.AssetType = "ERC20"
		metadata.OriginDecimals = &registry.Precision
		// SYSCOIN: UTXO-side SPT values are stored and exposed as CAmount/COIN
		// units, matching Core's 8-decimal asset_value formatting.
		precision = 8

	case 3: // ERC721 (NFT)
		realTokenId, err := c.getRealTokenId(assetId, tokenIdx)
		if err != nil {
			return nil, err
		}
		symbol, err = c.getTokenSymbol(contractAddr)
		if err != nil || symbol == "" {
			symbol = fmt.Sprintf("ERC721-%d", assetId)
		}
		metadata.Description = fmt.Sprintf("ERC721 NFT Token ID %s", realTokenId.String())
		metadata.AssetType = "ERC721"
		metadata.TokenID = realTokenId.String()
		originDecimals := uint8(0)
		metadata.OriginDecimals = &originDecimals
		precision = 0

	case 4: // ERC1155
		realTokenId, err := c.getRealTokenId(assetId, tokenIdx)
		if err != nil {
			return nil, err
		}
		symbol = fmt.Sprintf("ERC1155-%d", assetId)
		metadata.Description = fmt.Sprintf("ERC1155 Token ID %s", realTokenId.String())
		metadata.AssetType = "ERC1155"
		metadata.TokenID = realTokenId.String()
		originDecimals := uint8(0)
		metadata.OriginDecimals = &originDecimals
		precision = 0

	default:
		return nil, fmt.Errorf("unsupported NEVM asset type %d for asset %d", registry.AssetType, assetId)
	}

	metadataJSON, err := encodeBridgeAssetMetadata(metadata)
	if err != nil {
		return nil, err
	}

	return &bchain.Asset{
		Transactions: 0,
		AssetObj: wire.AssetType{
			Contract:    contractAddr.Bytes(),
			Symbol:      []byte(symbol),
			Precision:   precision,
			TotalSupply: 0,
			MaxSupply:   0,
		},
		MetaData: metadataJSON,
	}, nil
}
