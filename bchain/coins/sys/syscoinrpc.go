package syscoin

import (
	"encoding/json"
	"context"
	
	"github.com/golang/glog"
	"github.com/syscoin/blockbook/bchain"
	"github.com/syscoin/blockbook/bchain/coins/btc"
)

// SyscoinRPC is an interface to JSON-RPC bitcoind service
type SyscoinRPC struct {
	*btc.BitcoinRPC
	NEVMClient *NEVMClient
}


// NewSyscoinRPC returns new SyscoinRPC instance
func NewSyscoinRPC(config json.RawMessage, pushHandler func(notificationType bchain.NotificationType)) (bchain.BlockChain, error) {
	b, err := btc.NewBitcoinRPC(config, pushHandler)
	if err != nil {
		return nil, err
	}

	s := &SyscoinRPC{
		b.(*btc.BitcoinRPC),
		nil,
	}
	s.RPCMarshaler = btc.JSONMarshalerV2{}
	s.ChainConfig.SupportsEstimateFee = false

	return s, nil
}

func (b *SyscoinRPC) Shutdown(ctx context.Context) error {
	// Call BitcoinRPC's shutdown first
	if err := b.BitcoinRPC.Shutdown(ctx); err != nil {
		glog.Error("BitcoinRPC.Shutdown error: ", err)
		return err
	}

	// Then shutdown NEVMClient if it exists
	if b.NEVMClient != nil {
		b.NEVMClient.Close()
	}

	return nil
}


// Initialize initializes SyscoinRPC instance.
func (b *SyscoinRPC) Initialize() error {
	ci, err := b.GetChainInfo()
	if err != nil {
		return err
	}
	chainName := ci.Chain

	glog.Info("Chain name ", chainName)
	params := GetChainParams(chainName)

	// always create parser
	b.Parser = NewSyscoinParser(params, b.ChainConfig)
	// parameters for getInfo request
	if params.Net == MainnetMagic {
		b.Testnet = false
		b.Network = "livenet"
	} else {
		b.Testnet = true
		b.Network = "testnet"
	}
	b.NEVMClient, err = NewNEVMClient(b.ChainConfig)
	if err != nil {
		return err
	}
	glog.Info("rpc: block chain ", params.Name)

	return nil
}
func (b *SyscoinRPC) FetchNEVMAssetDetails(assetGuid uint64) (*bchain.Asset, error) {
	return b.NEVMClient.FetchNEVMAssetDetails(assetGuid)
}
func (b *SyscoinRPC) GetContractExplorerBaseURL() string {
	return b.ChainConfig.Web3Explorer
}
// GetBlock returns block with given hash
func (b *SyscoinRPC) GetBlock(hash string, height uint32) (*bchain.Block, error) {
	var err error
	if hash == "" {
		hash, err = b.GetBlockHash(height)
		if err != nil {
			return nil, err
		}
	}
	if !b.ParseBlocks {
		return b.GetBlockFull(hash)
	}
	return b.GetBlockWithoutHeader(hash, height)
}

func (b *SyscoinRPC) GetChainTips() (string, error) {
	glog.V(1).Info("rpc: getchaintips")

	res := btc.ResGetChainTips{}
	req := btc.CmdGetChainTips{Method: "getchaintips"}
	err := b.Call(&req, &res)

	if err != nil {
		return "", err
	}
	if res.Error != nil {
		return "", err
	}
	rawMarshal, err := json.Marshal(&res.Result)
    if err != nil {
        return "", err
    }
	decodedRawString := string(rawMarshal)
	return decodedRawString, nil
}

func (b *SyscoinRPC) GetSPVProof(hash string) (string, error) {
	glog.V(1).Info("rpc: getspvproof", hash)

	res := btc.ResGetSPVProof{}
	req := btc.CmdGetSPVProof{Method: "syscoingetspvproof"}
	req.Params.Txid = hash
	err := b.Call(&req, &res)

	if err != nil {
		return "", err
	}
	if res.Error != nil {
		return "", err
	}
	rawMarshal, err := json.Marshal(&res.Result)
    if err != nil {
        return "", err
    }
	decodedRawString := string(rawMarshal)
	return decodedRawString, nil
}

// GetTransactionForMempool returns a transaction by the transaction ID.
// It could be optimized for mempool, i.e. without block time and confirmations
func (b *SyscoinRPC) GetTransactionForMempool(txid string) (*bchain.Tx, error) {
	return b.GetTransaction(txid)
}

// Syscoin Core sendrawtransaction default maxfeerate is 0.10.
// Syscoin Core sendrawtransaction default maxburnamount is 0.0.
// Governance Proposal needs maxburnamount of 150.
// This change allows sending governance proposals without explicitly setting both parameters.
const (
	defaultSyscoinMaxFeeRate = "0.10"
	defaultSyscoinMaxBurnAmount = "150"
)

func syscoinSendRawParams(p bchain.SendRawTransactionParams) bchain.SendRawTransactionParams {
	if p.MaxFeeRate == nil || *p.MaxFeeRate == "" {
		s := defaultSyscoinMaxFeeRate
		p.MaxFeeRate = &s
	}
	if p.MaxBurnAmount == nil || *p.MaxBurnAmount == "" {
		s := defaultSyscoinMaxBurnAmount
		p.MaxBurnAmount = &s
	}
	return p
}

// Override BitcoinRPC to apply Syscoin default maxfeerate / maxburnamount.
func (b *SyscoinRPC) SendRawTransaction(tx string) (string, error) {
	return b.SendRawTransactionWithOpts(bchain.SendRawTransactionParams{Hex: tx})
}

// Forwards maxfeerate / maxburnamount to Syscoin Core sendrawtransaction.
func (b *SyscoinRPC) SendRawTransactionWithOpts(p bchain.SendRawTransactionParams) (string, error) {
	p = syscoinSendRawParams(p)
	return btc.SendRawTransactionWithParams(b.BitcoinRPC, p)
}

var _ bchain.SendRawTransactionOpts = (*SyscoinRPC)(nil)