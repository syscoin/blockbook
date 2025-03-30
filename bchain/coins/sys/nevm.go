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
)
const (
	vaultManagerAddress = "0x7904299b3D3dC1b03d1DdEb45E9fDF3576aCBd5f"
	
)

type NEVMClient struct {
    rpcClient *ethclient.Client
    vaultAddr common.Address
    vaultABI  abi.ABI
	tokenABI  abi.ABI
	explorerURL string
}

// NewClient initializes a new NEVM client.
func NewNEVMClient(rpcURL string, explorerURL string) (*NEVMClient, error) {
    ethClient, err := ethclient.Dial(rpcURL)
    if err != nil {
        return nil, err
    }

    parsedABI, err := abi.JSON(strings.NewReader(vaultABIJSON))
    if err != nil {
		ethClient.Close()
        return nil, err
    }

	tokenSymbolABI, err := abi.JSON(strings.NewReader(`
	[
	  {"constant":true,"inputs":[],"name":"symbol","outputs":[{"name":"","type":"string"}],"type":"function"}
	]`))
    if err != nil {
		ethClient.Close()
        return nil, err
    }
    return &NEVMClient{
        rpcClient: ethClient,
        vaultAddr: common.HexToAddress(vaultManagerAddress),
        vaultABI:  parsedABI,
		tokenABI: tokenSymbolABI,
		explorerURL: explorerURL,
    }, nil
}

func (c *NEVMClient) GetContractExplorerBaseURL() string {
	return c.explorerURL
}

func (c *NEVMClient) Close() {
	c.rpcClient.Close()
}

func (c *NEVMClient) getRealTokenId(assetId uint32, tokenIdx uint32) (*big.Int, error) {
	data, err := c.vaultABI.Pack("getRealTokenIdFromTokenIdx", assetId, tokenIdx)
	if err != nil {
		return nil, err
	}

	callMsg := ethereum.CallMsg{To: &c.vaultAddr, Data: data}
	res, err := c.rpcClient.CallContract(context.Background(), callMsg, nil)
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
	res, err := c.rpcClient.CallContract(context.Background(), callMsg, nil)
	if err != nil {
		return "", err
	}

	var unpacked []interface{}
	unpacked, err = c.tokenABI.Unpack("symbol", res)
	if err != nil || len(unpacked) == 0 {
		return "", err
	}
	return unpacked[0].(string), nil
}
func (c *NEVMClient) FetchNEVMAssetDetails(assetGuid uint64) (*bchain.Asset, error) {
	ctx := context.Background()

	assetId := uint32(assetGuid & 0xffffffff)
	tokenIdx := uint32(assetGuid >> 32)

	// Fetch basic registry details
	data, err := c.vaultABI.Pack("assetRegistry", assetId)
	if err != nil {
		return nil, err
	}

	callMsg := ethereum.CallMsg{To: &c.vaultAddr, Data: data}
	res, err := c.rpcClient.CallContract(ctx, callMsg, nil)
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

	assetType := registry.AssetType
	contractAddr := registry.AssetContract
	precision := registry.Precision

	var symbol string
	var metadata string

	switch assetType {
	case 1: // SYS native
		symbol = "SYS"
		metadata = "Native SYS asset"

	case 2: // ERC20
		symbol, err = c.getTokenSymbol(contractAddr)
		if err != nil || symbol == "" {
			symbol = fmt.Sprintf("ERC20-%d", assetId)
		}
		metadata = fmt.Sprintf("ERC20 Token (%s)", contractAddr.Hex())

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

	assetObj := wire.AssetType{
		Contract:    []byte(contractAddr.Hex()),
		Symbol:      []byte(symbol),
		Precision:   precision,
	}

	asset := &bchain.Asset{
		Transactions: 0,
		AssetObj:     assetObj,
		MetaData:     []byte(metadata),
	}

	return asset, nil
}


