package syscoin

import (
	"blockbook/bchain"
	"blockbook/bchain/coins/btc"
	"blockbook/bchain/coins/utils"
	"bytes"
	"math/big"
	"github.com/martinboehm/btcd/wire"
	"github.com/martinboehm/btcutil/chaincfg"
	"github.com/martinboehm/btcutil/txscript"
	"github.com/martinboehm/btcutil"
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
	BaseParser *bchain.BaseParser
}

// NewSyscoinParser returns new SyscoinParser instance
func NewSyscoinParser(params *chaincfg.Params, c *btc.Configuration) *SyscoinParser {
	return &SyscoinParser{
		BitcoinParser: btc.NewBitcoinParser(params, c),
		BaseParser:    &bchain.BaseParser{},
	}
}
// matches max data carrier for systx
func (p *SyscoinParser) GetMaxAddrLength() int {
	return 8000
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

	txs := make([]*bchain.Tx, len(w.Transactions))
	for ti, t := range w.Transactions {
		txs[ti] = p.TxFromMsgTx(t, false)
	}

	return &bchain.Block{
		BlockHeader: &bchain.BlockHeader{
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
    return nVersion == SYSCOIN_TX_VERSION_ASSET_ACTIVATE || nVersion == SYSCOIN_TX_VERSION_ASSET_UPDATE || nVersion == SYSCOIN_TX_VERSION_ASSET_TRANSFER
}
// note assetsend in core is assettx but its deserialized as allocation, we just care about balances so we can do it in same code for allocations
func (p *SyscoinParser) IsAssetAllocationTx(nVersion int32) bool {
    return nVersion == SYSCOIN_TX_VERSION_ALLOCATION_BURN_TO_ETHEREUM || nVersion == SYSCOIN_TX_VERSION_ALLOCATION_BURN_TO_SYSCOIN || nVersion == SYSCOIN_TX_VERSION_SYSCOIN_BURN_TO_ALLOCATION ||
        nVersion == SYSCOIN_TX_VERSION_ALLOCATION_SEND || nVersion == SYSCOIN_TX_VERSION_ALLOCATION_LOCK || nVersion == SYSCOIN_TX_VERSION_ASSET_SEND
}
func (p *SyscoinParser) IsSyscoinTx(nVersion int32) bool {
    return p.IsAssetTx(nVersion) || p.IsAssetAllocationTx(nVersion) || p.IsSyscoinMintTx(nVersion)
}
// addressToOutputScript converts bitcoin address to ScriptPubKey
func (p *SyscoinParser) addressToOutputScript(address string) ([]byte, error) {
	if(address == "burn") {
		return []byte("burn"), nil
	}
	da, err := btcutil.DecodeAddress(address, p.Params)
	if err != nil {
		return nil, err
	}
	script, err := txscript.PayToAddrScript(da)
	if err != nil {
		return nil, err
	}
	return script, nil
}

// outputScriptToAddresses converts ScriptPubKey to bitcoin addresses
func (p *SyscoinParser) outputScriptToAddresses(script []byte) ([]string, bool, error) {
	if(string(script) == "burn"){
		return []string{"burn"}, true, nil
	}
	sc, addresses, _, err := txscript.ExtractPkScriptAddrs(script, p.Params)
	if err != nil {
		return nil, false, err
	}
	rv := make([]string, len(addresses))
	for i, a := range addresses {
		rv[i] = a.EncodeAddress()
	}
	var s bool
	if sc == txscript.PubKeyHashTy || sc == txscript.WitnessV0PubKeyHashTy || sc == txscript.ScriptHashTy || sc == txscript.WitnessV0ScriptHashTy {
		s = true
	} else if len(rv) == 0 {
		or := p.TryParseOPReturn(script)
		if or != "" {
			rv = []string{or}
		}
	}
	return rv, s, nil
}

// GetAddrDescFromAddress returns internal address representation (descriptor) of given address
func (p *SyscoinParser) GetAddrDescFromAddress(address string) (bchain.AddressDescriptor, error) {
	return p.addressToOutputScript(address)
}

// GetAddressesFromAddrDesc returns addresses for given address descriptor with flag if the addresses are searchable
func (p *SyscoinParser) GetAddressesFromAddrDesc(addrDesc bchain.AddressDescriptor) ([]string, bool, error) {
	return p.OutputScriptToAddressesFunc(addrDesc)
}
// TryGetOPReturn tries to process OP_RETURN script and return data
func (p *SyscoinParser) TryGetOPReturn(script *[]byte) *[]byte {
	if len(*script) > 1 && (*script)[0] == txscript.OP_RETURN {
		// trying 3 variants of OP_RETURN data
		// 1) OP_RETURN <datalen> <data>
		// 2) OP_RETURN OP_PUSHDATA1 <datalen in 1 byte> <data>
		// 3) OP_RETURN OP_PUSHDATA2 <datalen in 2 bytes> <data>
		
		var data []byte
		if len(*script) < txscript.OP_PUSHDATA1 {
			data = (*script)[2:]
		} else if (*script)[1] == txscript.OP_PUSHDATA1 && len(*script) <= 0xff {
			data = (*script)[3:]
		} else if (*script)[1] == txscript.OP_PUSHDATA2 && len(*script) <= 0xffff {
			data = (*script)[4:]
		}
		return &data
	}
	return nil
}

func (p *SyscoinParser) UnpackAddrBalance(buf []byte, txidUnpackedLen int, detail bchain.AddressBalanceDetail) (*bchain.AddrBalance, error) {
	txs, l := p.BaseParser.UnpackVaruint(buf)
	sentSat, sl := p.BaseParser.UnpackBigint(buf[l:])
	balanceSat, bl := p.BaseParser.UnpackBigint(buf[l+sl:])
	l = l + sl + bl
	ab := &bchain.AddrBalance{
		Txs:        uint32(txs),
		SentSat:    sentSat,
		BalanceSat: balanceSat,
	}
	// unpack asset balance information
	numAssetBalances, ll := p.BaseParser.UnpackVaruint(buf[l:])
	l += ll
	if numAssetBalances > 0 {
		ab.AssetBalances = make(map[uint32]*bchain.AssetBalance, numAssetBalances)
		for i := uint(0); i < numAssetBalances; i++ {
			asset, ll := p.BaseParser.UnpackVaruint(buf[l:])
			l += ll
			balancevalue, ll := p.BaseParser.UnpackBigint(buf[l:])
			l += ll
			sentvalue, ll := p.BaseParser.UnpackBigint(buf[l:])
			ab.AssetBalances[uint32(asset)] = &bchain.AssetBalance{SentAssetSat: sentvalue, BalanceAssetSat: balancevalue}
			l += ll
		}
	}
	if detail != bchain.AddressBalanceDetailNoUTXO {
		// estimate the size of utxos to avoid reallocation
		ab.Utxos = make([]*bchain.Utxo, 0, len(buf[l:])/txidUnpackedLen+3)
		// ab.utxosMap = make(map[string]int, cap(ab.Utxos))
		for len(buf[l:]) >= txidUnpackedLen+3 {
			btxID := append([]byte(nil), buf[l:l+txidUnpackedLen]...)
			l += txidUnpackedLen
			vout, ll := p.BaseParser.UnpackVaruint(buf[l:])
			l += ll
			height, ll := p.BaseParser.UnpackVaruint(buf[l:])
			l += ll
			valueSat, ll := p.BaseParser.UnpackBigint(buf[l:])
			l += ll
			u := &bchain.Utxo{
				BtxID:    btxID,
				Vout:     int32(vout),
				Height:   uint32(height),
				ValueSat: valueSat,
			}
			if detail == bchain.AddressBalanceDetailUTXO {
				ab.Utxos = append(ab.Utxos, u)
			} else {
				ab.AddUtxo(u)
			}
		}
	}
	return ab, nil
}

func (p *SyscoinParser) PackAddrBalance(ab *bchain.AddrBalance, buf, varBuf []byte) []byte {
	buf = buf[:0]
	l := p.BaseParser.PackVaruint(uint(ab.Txs), varBuf)
	buf = append(buf, varBuf[:l]...)
	l = p.BaseParser.PackBigint(&ab.SentSat, varBuf)
	buf = append(buf, varBuf[:l]...)
	l = p.BaseParser.PackBigint(&ab.BalanceSat, varBuf)
	buf = append(buf, varBuf[:l]...)
	
	// pack asset balance information
	l = p.BaseParser.PackVaruint(uint(len(ab.AssetBalances)), varBuf)
	buf = append(buf, varBuf[:l]...)
	for key, value := range ab.AssetBalances {
		l = p.BaseParser.PackVaruint(uint(key), varBuf)
		buf = append(buf, varBuf[:l]...)
		l = p.BaseParser.PackBigint(&value.BalanceAssetSat, varBuf)
		buf = append(buf, varBuf[:l]...)
		l = p.BaseParser.PackBigint(&value.SentAssetSat, varBuf)
		buf = append(buf, varBuf[:l]...)
	}
	for _, utxo := range ab.Utxos {
		// if Vout < 0, utxo is marked as spent
		if utxo.Vout >= 0 {
			buf = append(buf, utxo.BtxID...)
			l = p.BaseParser.PackVaruint(uint(utxo.Vout), varBuf)
			buf = append(buf, varBuf[:l]...)
			l = p.BaseParser.PackVaruint(uint(utxo.Height), varBuf)
			buf = append(buf, varBuf[:l]...)
			l = p.BaseParser.PackBigint(&utxo.ValueSat, varBuf)
			buf = append(buf, varBuf[:l]...)
		}
	}
	return buf
}

func (p *SyscoinParser) AppendTokenTransfer(tt *bchain.TokenTransfer, buf []byte, varBuf []byte) []byte {
	l := p.BaseParser.PackVaruint(uint(len(tt.Type)), varBuf)
	buf = append(buf, varBuf[:l]...)
	buf = append(buf, []byte(tt.Type)...)
	l = p.BaseParser.PackVaruint(uint(len(tt.From)), varBuf)
	buf = append(buf, varBuf[:l]...)
	buf = append(buf, []byte(tt.From)...)
	l = p.BaseParser.PackVaruint(uint(len(tt.To)), varBuf)
	buf = append(buf, varBuf[:l]...)
	buf = append(buf, []byte(tt.To)...)
	l = p.BaseParser.PackVaruint(uint(len(tt.Token)), varBuf)
	buf = append(buf, varBuf[:l]...)
	buf = append(buf, []byte(tt.Token)...)
	l = p.BaseParser.PackVaruint(uint(len(tt.Symbol)), varBuf)
	buf = append(buf, varBuf[:l]...)
	buf = append(buf, []byte(tt.Symbol)...)
	l = p.BaseParser.PackVaruint(uint(tt.Decimals), varBuf)
	buf = append(buf, varBuf[:l]...)
	l = p.BaseParser.PackBigint((*big.Int)(tt.Value), varBuf)
	buf = append(buf, varBuf[:l]...)
	return buf
}

func (p *SyscoinParser) UnpackTokenTransfer(tt *bchain.TokenTransfer, buf []byte) int {
	var Decimals uint
	var Value big.Int
	al, l := p.BaseParser.UnpackVaruint(buf)
	tt.Type = bchain.TokenType(append([]byte(nil), buf[l:l+int(al)]...))
	ll := uint(l)+al
	al, l = p.BaseParser.UnpackVaruint(buf[ll:])
	ll += uint(l)
	tt.From = string(append([]byte(nil), buf[l:l+int(al)]...))
	ll += al
	al, l = p.BaseParser.UnpackVaruint(buf[ll:])
	ll += uint(l)
	tt.To = string(append([]byte(nil), buf[l:l+int(al)]...))
	ll += al
	al, l = p.BaseParser.UnpackVaruint(buf[ll:])
	ll += uint(l)
	tt.Token = string(append([]byte(nil), buf[l:l+int(al)]...))
	ll += al
	al, l = p.BaseParser.UnpackVaruint(buf[ll:])
	ll += uint(l)
	tt.Symbol = string(append([]byte(nil), buf[l:l+int(al)]...))
	ll += al
	Decimals, l = p.BaseParser.UnpackVaruint(buf[ll:])
	tt.Decimals = int(Decimals)
	ll += uint(l)
	Value, l = p.BaseParser.UnpackBigint(buf[ll:])
	tt.Value = (*bchain.Amount)(&Value)
	return int(ll)+l
}

func (p *SyscoinParser) PackTxAddresses(ta *bchain.TxAddresses, buf []byte, varBuf []byte) []byte {
	buf = buf[:0]
	// pack version info for syscoin to detect sysx tx types
	l := p.BaseParser.PackVaruint(uint(ta.Version), varBuf)
	buf = append(buf, varBuf[:l]...)
	l = p.BaseParser.PackVaruint(uint(ta.Height), varBuf)
	buf = append(buf, varBuf[:l]...)
	l = p.BaseParser.PackVaruint(uint(len(ta.Inputs)), varBuf)
	buf = append(buf, varBuf[:l]...)
	for i := range ta.Inputs {
		buf = p.BitcoinParser.AppendTxInput(ta.Inputs[i], buf, varBuf)
	}
	l = p.BaseParser.PackVaruint(uint(len(ta.Outputs)), varBuf)
	buf = append(buf, varBuf[:l]...)
	for i := range ta.Outputs {
		buf = p.BitcoinParser.AppendTxOutput(ta.Outputs[i], buf, varBuf)
	}
	l = p.BaseParser.PackVaruint(uint(len(ta.TokenTransfers)), varBuf)
	buf = append(buf, varBuf[:l]...)
	for i := range ta.TokenTransfers {
		buf = p.AppendTokenTransfer(ta.TokenTransfers[i], buf, varBuf)
	}
	return buf
}

func (p *SyscoinParser) UnpackTxAddresses(buf []byte) (*bchain.TxAddresses, error) {
	ta := bchain.TxAddresses{}
	// unpack version info for syscoin to detect sysx tx types
	version, l := p.BaseParser.UnpackVaruint(buf)
	ta.Version = int32(version)
	height, ll := p.BaseParser.UnpackVaruint(buf[l:])
	ta.Height = uint32(height)
	l += ll
	inputs, ll := p.BaseParser.UnpackVaruint(buf[l:])
	l += ll
	ta.Inputs = make([]*bchain.TxInput, inputs)
	for i := uint(0); i < inputs; i++ {
		l += p.BitcoinParser.UnpackTxInput(ta.Inputs[i], buf[l:])
	}
	outputs, ll := p.BaseParser.UnpackVaruint(buf[l:])
	l += ll
	ta.Outputs = make([]*bchain.TxOutput, outputs)
	for i := uint(0); i < outputs; i++ {
		l += p.BitcoinParser.UnpackTxOutput(ta.Outputs[i], buf[l:])
	}
	tokenTransfers, ll := p.BaseParser.UnpackVaruint(buf[l:])
	l += ll
	ta.TokenTransfers = make([]*bchain.TokenTransfer, tokenTransfers)
	for i := uint(0); i < tokenTransfers; i++ {
		l += p.UnpackTokenTransfer(ta.TokenTransfers[i], buf[l:])
	}
	return &ta, nil
}