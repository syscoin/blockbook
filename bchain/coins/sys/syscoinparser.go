package syscoin

import (
	"blockbook/bchain"
	"blockbook/bchain/coins/btc"
	"blockbook/bchain/coins/utils"
	"bytes"

	"github.com/martinboehm/btcd/wire"
	"github.com/martinboehm/btcutil/chaincfg"
)

// magic numbers
const (
	MainnetMagic wire.BitcoinNet = 0xffcae2ce
	RegtestMagic wire.BitcoinNet = 0xdab5bffa
	SYSCOIN_TX_VERSION_ALLOCATION_BURN_TO_SYSCOIN uint32 = 0x7400
	SYSCOIN_TX_VERSION_SYSCOIN_BURN_TO_ALLOCATION uint32 = 0x7401
	SYSCOIN_TX_VERSION_ASSET_ACTIVATE uint32 = 0x7402
	SYSCOIN_TX_VERSION_ASSET_UPDATE uint32 = 0x7403
	SYSCOIN_TX_VERSION_ASSET_TRANSFER uint32 = 0x7404
	SYSCOIN_TX_VERSION_ASSET_SEND uint32 = 0x7405
	SYSCOIN_TX_VERSION_ALLOCATION_MINT uint32 = 0x7406
	SYSCOIN_TX_VERSION_ALLOCATION_BURN_TO_ETHEREUM uint32 = 0x7407
	SYSCOIN_TX_VERSION_ALLOCATION_SEND uint32 = 0x7408
	SYSCOIN_TX_VERSION_ALLOCATION_LOCK uint32 = 0x7409
)

// chain parameters
var (
	MainNetParams chaincfg.Params
	RegtestParams chaincfg.Params
)

func init() {
	MainNetParams = chaincfg.MainNetParams
	MainNetParams.Net = MainnetMagic

	// Mainnet address encoding magics
	MainNetParams.PubKeyHashAddrID = []byte{63} // base58 prefix: s
	MainNetParams.ScriptHashAddrID = []byte{5} // base68 prefix: 3
	MainNetParams.Bech32HRPSegwit = "sys"

	RegtestParams = chaincfg.RegressionNetParams
	RegtestParams.Net = RegtestMagic

	// Regtest address encoding magics
	RegtestParams.PubKeyHashAddrID = []byte{65} // base58 prefix: t
	RegtestParams.ScriptHashAddrID = []byte{196} // base58 prefix: 2
	RegtestParams.Bech32HRPSegwit = "tsys"
}

// SyscoinParser handle
type SyscoinParser struct {
	*btc.BitcoinParser
}

// NewSyscoinParser returns new SyscoinParser instance
func NewSyscoinParser(params *chaincfg.Params, c *btc.Configuration) *SyscoinParser {
	return &SyscoinParser{BitcoinParser: btc.NewBitcoinParser(params, c)}
}

// GetChainParams returns network parameters
func GetChainParams(chain string) *chaincfg.Params {
	if !chaincfg.IsRegistered(&MainNetParams) {
		err := chaincfg.Register(&MainNetParams)
		if err == nil {
			err = chaincfg.Register(&RegtestParams)
		}
		if err != nil {
			panic(err)
		}
	}

	switch chain {
	case "regtest":
		return &RegtestParams
	default:
		return &MainNetParams
	}
}
// ParseBlock parses raw block to our Block struct
// it has special handling for Auxpow blocks that cannot be parsed by standard btc wire parse
func (p *SyscoinParser) ParseBlock(b []byte) (*bchain.Block, error) {
	r := bytes.NewReader(b)
	w := wire.MsgBlock{}
	h := wire.BlockHeader{}
	err := h.Deserialize(r)
	if err != nil {
		return nil, err
	}

	if (h.Version & utils.VersionAuxpow) != 0 {
		if err = utils.SkipAuxpow(r); err != nil {
			return nil, err
		}
	}

	err = utils.DecodeTransactions(r, 0, wire.WitnessEncoding, &w)
	if err != nil {
		return nil, err
	}

	txs := make([]bchain.Tx, len(w.Transactions))
	for ti, t := range w.Transactions {
		txs[ti] = p.TxFromMsgTx(t, false)
	}

	return &bchain.Block{
		BlockHeader: bchain.BlockHeader{
			Size: len(b),
			Time: h.Timestamp.Unix(),
		},
		Txs: txs,
	}, nil
}

func (p *SyscoinParser) IsSyscoinMintTx(uint32 nVersion) bool {
    return nVersion == SYSCOIN_TX_VERSION_ALLOCATION_MINT
}
func (p *SyscoinParser) IsAssetTx(uint32 nVersion) bool {
    return nVersion == SYSCOIN_TX_VERSION_ASSET_ACTIVATE || nVersion == SYSCOIN_TX_VERSION_ASSET_UPDATE || nVersion == SYSCOIN_TX_VERSION_ASSET_TRANSFER || nVersion == SYSCOIN_TX_VERSION_ASSET_SEND
}
func (p *SyscoinParser) IsAssetAllocationTx(uint32 nVersion) bool {
    return nVersion == SYSCOIN_TX_VERSION_ALLOCATION_BURN_TO_ETHEREUM || nVersion == SYSCOIN_TX_VERSION_ALLOCATION_BURN_TO_SYSCOIN || nVersion == SYSCOIN_TX_VERSION_SYSCOIN_BURN_TO_ALLOCATION ||
        nVersion == SYSCOIN_TX_VERSION_ALLOCATION_SEND || nVersion == SYSCOIN_TX_VERSION_ALLOCATION_LOCK
}
func (p *SyscoinParser) IsSyscoinTx(uint32 nVersion) bool {
    return p.IsAssetTx(nVersion) || p.IsAssetAllocationTx(nVersion) || p.IsSyscoinMintTx(nVersion)
}
// TryGetOPReturn tries to process OP_RETURN script and return data
func (p *SyscoinParser) TryGetOPReturn(script []byte) []byte {
	if len(script) > 1 && script[0] == txscript.OP_RETURN {
		// trying 3 variants of OP_RETURN data
		// 1) OP_RETURN <datalen> <data>
		// 2) OP_RETURN OP_PUSHDATA1 <datalen in 1 byte> <data>
		// 3) OP_RETURN OP_PUSHDATA2 <datalen in 2 bytes> <data>
		
		var data []byte
		if len(script) < txscript.OP_PUSHDATA1 {
			data = script[2:]
		} else if script[1] == txscript.OP_PUSHDATA1 && len(script) <= 0xff {
			data = script[3:]
		} else if script[1] == txscript.OP_PUSHDATA2 && len(script) <= 0xffff {
			data = script[4:]
		}
		return data
	}
	return nil
}
// GetChainType is type of the blockchain, default is ChainBitcoinType
func (p *SyscoinParser) GetChainType() ChainType {
	return ChainSyscoinType
}
func (p *SyscoinParser) ConnectAssetAllocationOutput(d *RocksDB, sptData []bytes, balances map[string]*AddrBalance, version uint32) (*SyscoinOutputPackage, error) {
	var pt ProtoTransaction_AssetAllocationType
	err := proto.Unmarshal(sptData, &pt)
	if err != nil {
		return nil, err
	}
	totalValue := big.NewInt(0)
	assetSenderAddrDesc, err := p.GetAddrDescFromAddress(pt.assetAllocationTuple.witnessProgram.ToString())
	if err != nil || len(assetSenderAddrDesc) == 0 || len(assetSenderAddrDesc) > maxAddrDescLen {
		if err != nil {
			// do not log ErrAddressMissing, transactions can be without to address (for example eth contracts)
			if err != bchain.ErrAddressMissing {
				glog.Warningf("rocksdb: asset sender addrDesc: %v - height %d, tx %v, output %v, error %v", err, block.Height, tx.Txid, output, err)
			}
		} else {
			glog.V(1).Infof("rocksdb: height %d, tx %v, vout %v, skipping asset sender addrDesc of length %d", block.Height, tx.Txid, i, len(assetSenderAddrDesc))
		}
		continue
	}
	strAddrDescriptors := make([]string, 0, len(pt.listSendingAllocationAmounts))
	for allocationIndex, allocation := range pt.listSendingAllocationAmounts {
		addrDesc, err := p.GetAddrDescFromAddress(allocation.witnessProgram.ToString())
		if err != nil || len(addrDesc) == 0 || len(addrDesc) > maxAddrDescLen {
			if err != nil {
				// do not log ErrAddressMissing, transactions can be without to address (for example eth contracts)
				if err != bchain.ErrAddressMissing {
					glog.Warningf("rocksdb: asset addrDesc: %v - height %d, tx %v, output %v, error %v", err, block.Height, tx.Txid, output, err)
				}
			} else {
				glog.V(1).Infof("rocksdb: height %d, tx %v, vout %v, skipping asset addrDesc of length %d", block.Height, tx.Txid, i, len(addrDesc))
			}
			continue
		}
		strAddrDesc := string(addrDesc)
		balance, e := balances[strAddrDesc]
		if !e {
			balance, err = d.GetAddrDescBalance(addrDesc, addressBalanceDetailUTXOIndexed)
			if err != nil {
				return nil, err
			}
			if balance == nil {
				balance = &AddrBalance{}
			}
			balances[strAddrDesc] = balance
		}
		if balance.BalanceAssetAllocatedSat == nil{
			balance.BalanceAssetAllocatedSat = map[uint32]big.Int{}
		}
		balanceAssetAllocatedSat, ok := balance.BalanceAssetAllocatedSat[pt.assetAllocationTuple.Asset]
		if !ok {
			balanceAssetAllocatedSat = big.NewInt(0) 
		}
		strAddrDescriptors = append(strAddrDescriptors, strAddrDesc)
		balanceAssetAllocatedSat.Add(&balanceAssetAllocatedSat, &allocation.Amount)
		totalAssetSentValue.Add(&totalAssetSentValue, &allocation.Amount)
		balance.BalanceAssetAllocatedSat[pt.assetAllocationTuple.Asset] = balanceAssetAllocatedSat
	}
	return &SyscoinOutputPackage{
		Version: version,
		AssetGuid: pt.assetAllocationTuple.Asset,
		TotalAssetSentValue: totalAssetSentValue,
		AssetSenderAddrDesc: assetSenderAddrDesc,
		AssetReceiverStrAddrDesc: strAddrDescriptors,
	}, nil
}
func (p *SyscoinParser) ConnectAssetAllocationInput(outputPackage SyscoinOutputPackage, balance *AddrBalance) bool {
	
	if balance.SentAssetAllocatedSat == nil{
		balance.SentAssetAllocatedSat = map[uint32]big.Int{}
	}
	sentAssetAllocatedSat := balance.SentAssetAllocatedSat[outputPackage.assetGuid]
	balanceAssetAllocatedSat, ok := balance.BalanceAssetAllocatedSat[outputPackage.assetGuid]
	if !ok {
		balanceAssetAllocatedSat = big.NewInt(0) 
	}
	balanceAssetAllocatedSat.Sub(&balanceAssetAllocatedSat, &outputPackage.totalAssetSentValue)
	sentAssetAllocatedSat.Add(&sentAssetAllocatedSat, &outputPackage.totalAssetSentValue)
	if balanceAssetAllocatedSat.Sign() < 0 {
		d.resetValueSatToZero(&balanceAssetAllocatedSat, outputPackage.assetSenderAddrDesc, "balance")
	}
	balance.SentAssetAllocatedSat[outputPackage.assetGuid] = sentAssetAllocatedSat
	balance.BalanceAssetAllocatedSat[outputPackage.assetGuid] = balanceAssetAllocatedSat
	return true

}
func (p *SyscoinParser) ConnectOutputs(d *RocksDB, script []byte, balances map[string]*AddrBalance, version uint32) (*SyscoinOutputPackage, error) {
	sptData := p.TryGetOPReturn(script)
	if sptData == nil {
		return nil, nil
	}
	if p.IsAssetAllocationTx(version) {
		return p.ConnectAssetAllocationOutput(d, sptData, balances, version)
	}
}
func (p *SyscoinParser) ConnectInputs(outputPackage SyscoinOutputPackage, balance *AddrBalance) bool {
	if p.IsAssetAllocationTx(outputPackage.Version) {
		return p.ConnectAssetAllocationInput(outputPackage, balance)
	}
	return false
}

