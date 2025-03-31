package syscoin

import (
    "context"
    "math/big"
    "strings"
	"fmt"

    "github.com/ethereum/go-ethereum/accounts/abi"
    "github.com/ethereum/go-ethereum/common"
    "github.com/ethereum/go-ethereum/ethclient"
	"github.com/syscoin/blockbook/bchain"
	ethereum "github.com/ethereum/go-ethereum"
	"github.com/syscoin/syscoinwire/syscoin/wire"
	"github.com/syscoin/blockbook/bchain/coins/btc"
	"github.com/golang/glog"
)
const (
	vaultManagerAddress = "0x7904299b3D3dC1b03d1DdEb45E9fDF3576aCBd5f"
	
)
type NEVMClient struct {
	rpcClient   *ethclient.Client
	backupClient *ethclient.Client
	vaultAddr   common.Address
	vaultABI    abi.ABI
	tokenABI    abi.ABI
	explorerURL string
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

	return &NEVMClient{
		rpcClient:    mainClient,
		backupClient: backupClient,
		vaultAddr:    common.HexToAddress(vaultManagerAddress),
		vaultABI:     vaultABI,
		tokenABI:     tokenABI,
		explorerURL:  c.Web3Explorer,
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
func (c *NEVMClient) callContract(ctx context.Context, msg ethereum.CallMsg) ([]byte, error) {
	res, err := c.rpcClient.CallContract(ctx, msg, nil)
	if err == nil {
		return res, nil
	}

	// If backup RPC exists, attempt fallback
	if c.backupClient != nil {
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
	res, err := c.callContract(context.Background(), callMsg)
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
	res, err := c.callContract(context.Background(), callMsg)
	if err != nil {
		return "", err
	}

	unpacked, err := c.tokenABI.Unpack("symbol", res)
	if err != nil || len(unpacked) == 0 {
		return "", err
	}

	return unpacked[0].(string), nil
}

func (c *NEVMClient) FetchNEVMAssetDetails(assetGuid uint64) (*bchain.Asset, error) {
	if assetGuid == 123456 {
		return &bchain.Asset{
			Transactions: 0,
			AssetObj: wire.AssetType{
				Contract:    []byte{},
				Symbol:      []byte("SYSX"),
				Precision:   8,
				TotalSupply: 0,
				MaxSupply:   0,
			},
			MetaData: []byte("Syscoin Native Asset"),
		}, nil
	}
	
	ctx := context.Background()

	assetId := uint32(assetGuid & 0xffffffff)
	tokenIdx := uint32(assetGuid >> 32)

	data, err := c.vaultABI.Pack("assetRegistry", assetId)
	if err != nil {
		return nil, err
	}
	glog.Infof("Calling vaultManager for assetId: %d with data: %x", assetId, data)
	callMsg := ethereum.CallMsg{To: &c.vaultAddr, Data: data}
	res, err := c.callContract(ctx, callMsg)
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

	var symbol, metadata string
	contractAddr := registry.AssetContract
	precision := registry.Precision
	if precision > 8 {
		precision = 8
	}
	switch registry.AssetType {
	case 2: // ERC20
		symbol, err = c.getTokenSymbol(contractAddr)
		if err != nil || symbol == "" {
			symbol = fmt.Sprintf("ERC20-%d", assetId)
		}
		metadata = "ERC20 Token"

	case 3: // ERC721 (NFT)
		realTokenId, err := c.getRealTokenId(assetId, tokenIdx)
		if err != nil {
			return nil, err
		}
		symbol, err = c.getTokenSymbol(contractAddr)
		if err != nil || symbol == "" {
			symbol = fmt.Sprintf("ERC721-%d", assetId)
		}
		metadata = fmt.Sprintf("ERC721 NFT Token ID %s", realTokenId.String())

	case 4: // ERC1155
		realTokenId, err := c.getRealTokenId(assetId, tokenIdx)
		if err != nil {
			return nil, err
		}
		symbol = fmt.Sprintf("ERC1155-%d", assetId)
		metadata = fmt.Sprintf("ERC1155 Token ID %s", realTokenId.String())

	default:
		symbol = fmt.Sprintf("UNKNOWN-%d", assetId)
		metadata = "Unknown Asset Type"
	}

	return &bchain.Asset{
		Transactions: 0,
		AssetObj: wire.AssetType{
			Contract:    []byte(contractAddr.Hex()),
			Symbol:      []byte(symbol),
			Precision:   precision,
			TotalSupply: 0,
			MaxSupply:   0,
		},
		MetaData: []byte(metadata),
	}, nil
}


