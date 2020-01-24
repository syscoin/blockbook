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
	SYSCOIN_TX_VERSION_ALLOCATION_BURN_TO_SYSCOIN int32 = 0x7400
	SYSCOIN_TX_VERSION_SYSCOIN_BURN_TO_ALLOCATION int32 = 0x7401
	SYSCOIN_TX_VERSION_ASSET_ACTIVATE int32 = 0x7402
	SYSCOIN_TX_VERSION_ASSET_UPDATE int32 = 0x7403
	SYSCOIN_TX_VERSION_ASSET_TRANSFER int32 = 0x7404
	SYSCOIN_TX_VERSION_ASSET_SEND int32 = 0x7405
	SYSCOIN_TX_VERSION_ALLOCATION_MINT int32 = 0x7406
	SYSCOIN_TX_VERSION_ALLOCATION_BURN_TO_ETHEREUM int32 = 0x7407
	SYSCOIN_TX_VERSION_ALLOCATION_SEND int32 = 0x7408
	SYSCOIN_TX_VERSION_ALLOCATION_LOCK int32 = 0x7409
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

func (p *SyscoinParser) IsSyscoinMintTx(nVersion int32) bool {
    return nVersion == SYSCOIN_TX_VERSION_ALLOCATION_MINT
}
func (p *SyscoinParser) IsAssetTx(nVersion int32) bool {
    return nVersion == SYSCOIN_TX_VERSION_ASSET_ACTIVATE || nVersion == SYSCOIN_TX_VERSION_ASSET_UPDATE || nVersion == SYSCOIN_TX_VERSION_ASSET_TRANSFER || nVersion == SYSCOIN_TX_VERSION_ASSET_SEND
}
func (p *SyscoinParser) IsAssetAllocationTx(nVersion int32) bool {
    return nVersion == SYSCOIN_TX_VERSION_ALLOCATION_BURN_TO_ETHEREUM || nVersion == SYSCOIN_TX_VERSION_ALLOCATION_BURN_TO_SYSCOIN || nVersion == SYSCOIN_TX_VERSION_SYSCOIN_BURN_TO_ALLOCATION ||
        nVersion == SYSCOIN_TX_VERSION_ALLOCATION_SEND || nVersion == SYSCOIN_TX_VERSION_ALLOCATION_LOCK
}
func (p *SyscoinParser) IsSyscoinTx(nVersion int32) bool {
    return p.IsAssetTx(nVersion) || p.IsAssetAllocationTx(nVersion) || p.IsSyscoinMintTx(nVersion)
}
// IsAddrDescIndexable returns true if AddressDescriptor should be added to index
// empty or OP_RETURN scripts are not indexed if not sys tx
func (p *SyscoinParser) IsAddrDescIndexable(addrDesc bchain.AddressDescriptor, version int32) bool {
	if !p.IsSyscoinTx(version) && (len(addrDesc) == 0 || addrDesc[0] == txscript.OP_RETURN) {
		return false
	}
	return true
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
func (p *SyscoinParser) GetChainType() bchain.ChainType {
	return bchain.ChainSyscoinType
}

