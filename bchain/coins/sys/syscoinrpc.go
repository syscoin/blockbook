package syscoin

import (
	"context"
	"encoding/json"

	"github.com/golang/glog"
	"github.com/trezor/blockbook/bchain"
	"github.com/trezor/blockbook/bchain/coins/btc"
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
	if height > 0 {
		return b.GetBlockWithoutHeader(hash, height)
	}
	return b.BitcoinRPC.GetBlock(hash, height)
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
		return "", res.Error
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

// SYSCOIN: Syscoin Core sendrawtransaction default maxburnamount is 0.0.
// Governance proposal collateral needs 150 SYS burn allowance.
const (
	defaultSyscoinMaxFeeRate    = "0.10"
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

// SendRawTransaction overrides BitcoinRPC to apply Syscoin default maxfeerate /
// maxburnamount while preserving upstream's disableAlternativeRPC argument.
func (b *SyscoinRPC) SendRawTransaction(tx string, disableAlternativeRPC bool) (string, error) {
	return b.SendRawTransactionWithOpts(bchain.SendRawTransactionParams{Hex: tx, DisableAlternativeRPC: disableAlternativeRPC})
}

// Forwards maxfeerate / maxburnamount to Syscoin Core sendrawtransaction.
func (b *SyscoinRPC) SendRawTransactionWithOpts(p bchain.SendRawTransactionParams) (string, error) {
	p = syscoinSendRawParams(p)
	return btc.SendRawTransactionWithParams(b.BitcoinRPC, p)
}

var _ bchain.SendRawTransactionOpts = (*SyscoinRPC)(nil)
