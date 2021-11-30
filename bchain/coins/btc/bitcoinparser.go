package btc

import (
	"encoding/json"
	"math/big"
	"github.com/golang/glog"
	vlq "github.com/bsm/go-vlq"
	"github.com/juju/errors"

	"github.com/martinboehm/btcd/wire"
	"github.com/martinboehm/btcutil/chaincfg"
	"github.com/syscoin/blockbook/bchain"
	"github.com/syscoin/blockbook/common"
)

// temp params for signet(wait btcd commit)
// magic numbers
const (
	SignetMagic wire.BitcoinNet = 0x6a70c7f0
)

// chain parameters
var (
	SigNetParams chaincfg.Params
)

func init() {
	SigNetParams = chaincfg.TestNet3Params
	SigNetParams.Net = SignetMagic
}

// BitcoinParser handle
type BitcoinParser struct {
	*BitcoinLikeParser
}

// NewBitcoinParser returns new BitcoinParser instance
func NewBitcoinParser(params *chaincfg.Params, c *Configuration) *BitcoinParser {
	return &BitcoinParser{
		BitcoinLikeParser: NewBitcoinLikeParser(params, c),
	}
}

// GetChainParams contains network parameters for the main Bitcoin network,
// the regression test Bitcoin network, the test Bitcoin network and
// the simulation test Bitcoin network, in this order
func GetChainParams(chain string) *chaincfg.Params {
	if !chaincfg.IsRegistered(&chaincfg.MainNetParams) {
		chaincfg.RegisterBitcoinParams()
	}
	switch chain {
	case "test":
		return &chaincfg.TestNet3Params
	case "regtest":
		return &chaincfg.RegressionNetParams
	}
	return &chaincfg.MainNetParams
}

// ScriptPubKey contains data about output script
type ScriptPubKey struct {
	// Asm       string   `json:"asm"`
	Hex string `json:"hex,omitempty"`
	// Type      string   `json:"type"`
	Addresses []string `json:"addresses"` // removed from Bitcoind 22.0.0
	Address   string   `json:"address"`   // used in Bitcoind 22.0.0
}

// Vout contains data about tx output
type Vout struct {
	ValueSat     big.Int
	JsonValue    common.JSONNumber `json:"value"`
	N            uint32            `json:"n"`
	ScriptPubKey ScriptPubKey      `json:"scriptPubKey"`
}

// Tx is blockchain transaction
// unnecessary fields are commented out to avoid overhead
type Tx struct {
	Hex         string       `json:"hex"`
	Txid        string       `json:"txid"`
	Version     int32        `json:"version"`
	LockTime    uint32       `json:"locktime"`
	Vin         []bchain.Vin `json:"vin"`
	Vout        []Vout       `json:"vout"`
	BlockHeight uint32       `json:"blockHeight,omitempty"`
	// BlockHash     string `json:"blockhash,omitempty"`
	Confirmations    uint32      `json:"confirmations,omitempty"`
	Time             int64       `json:"time,omitempty"`
	Blocktime        int64       `json:"blocktime,omitempty"`
	CoinSpecificData interface{} `json:"-"`
}

// ParseTxFromJson parses JSON message containing transaction and returns Tx struct
// Bitcoind version 22.0.0 removed ScriptPubKey.Addresses from the API and replaced it by a single Address
func (p *BitcoinParser) ParseTxFromJson(msg json.RawMessage) (*bchain.Tx, error) {
	var bitcoinTx Tx
	var tx bchain.Tx
	err := json.Unmarshal(msg, &bitcoinTx)
	if err != nil {
		return nil, err
	}

	// it is necessary to copy bitcoinTx to Tx to make it compatible
	tx.Hex = bitcoinTx.Hex
	tx.Txid = bitcoinTx.Txid
	tx.Version = bitcoinTx.Version
	tx.LockTime = bitcoinTx.LockTime
	tx.Vin = bitcoinTx.Vin
	tx.BlockHeight = bitcoinTx.BlockHeight
	tx.Confirmations = bitcoinTx.Confirmations
	tx.Time = bitcoinTx.Time
	tx.Blocktime = bitcoinTx.Blocktime
	tx.CoinSpecificData = bitcoinTx.CoinSpecificData
	tx.Vout = make([]bchain.Vout, len(bitcoinTx.Vout))

	for i := range bitcoinTx.Vout {
		bitcoinVout := &bitcoinTx.Vout[i]
		vout := &tx.Vout[i]
		// convert vout.JsonValue to big.Int and clear it, it is only temporary value used for unmarshal
		vout.ValueSat, err = p.AmountToBigInt(bitcoinVout.JsonValue)
		if err != nil {
			return nil, err
		}
		vout.N = bitcoinVout.N
		vout.ScriptPubKey.Hex = bitcoinVout.ScriptPubKey.Hex
		// convert single Address to Addresses if Addresses are empty
		if len(bitcoinVout.ScriptPubKey.Addresses) == 0 {
			vout.ScriptPubKey.Addresses = []string{bitcoinVout.ScriptPubKey.Address}
		} else {
			vout.ScriptPubKey.Addresses = bitcoinVout.ScriptPubKey.Addresses
		}
	}

	return &tx, nil
}
func (p *BitcoinParser) PackAddrBalance(ab *bchain.AddrBalance, buf, varBuf []byte) []byte {
	buf = buf[:0]
	l := p.BaseParser.PackVaruint(uint(ab.Txs), varBuf)
	buf = append(buf, varBuf[:l]...)
	l = p.BaseParser.PackBigint(&ab.SentSat, varBuf)
	buf = append(buf, varBuf[:l]...)
	l = p.BaseParser.PackBigint(&ab.BalanceSat, varBuf)
	buf = append(buf, varBuf[:l]...)
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

func (p *BitcoinParser) UnpackAddrBalance(buf []byte, txidUnpackedLen int, detail bchain.AddressBalanceDetail) (*bchain.AddrBalance, error) {
	txs, l := p.BaseParser.UnpackVaruint(buf)
	sentSat, sl := p.BaseParser.UnpackBigint(buf[l:])
	balanceSat, bl := p.BaseParser.UnpackBigint(buf[l+sl:])
	l = l + sl + bl
	ab := &bchain.AddrBalance{
		Txs:        uint32(txs),
		SentSat:    sentSat,
		BalanceSat: balanceSat,
	}

	if detail != bchain.AddressBalanceDetailNoUTXO {
		// estimate the size of utxos to avoid reallocation
		ab.Utxos = make([]bchain.Utxo, 0, len(buf[l:])/txidUnpackedLen+3)
		// ab.UtxosMap = make(map[string]int, cap(ab.Utxos))
		for len(buf[l:]) >= txidUnpackedLen+3 {
			btxID := append([]byte(nil), buf[l:l+txidUnpackedLen]...)
			l += txidUnpackedLen
			vout, ll := p.BaseParser.UnpackVaruint(buf[l:])
			l += ll
			height, ll := p.BaseParser.UnpackVaruint(buf[l:])
			l += ll
			valueSat, ll := p.BaseParser.UnpackBigint(buf[l:])
			l += ll
			u := bchain.Utxo{
				BtxID:    btxID,
				Vout:     int32(vout),
				Height:   uint32(height),
				ValueSat: valueSat,
			}
			if detail == bchain.AddressBalanceDetailUTXO {
				ab.Utxos = append(ab.Utxos, u)
			} else {
				ab.AddUtxo(&u)
			}
		}
	}
	return ab, nil
}

func (p *BitcoinParser) PackTxAddresses(ta *bchain.TxAddresses, buf []byte, varBuf []byte) []byte {
	buf = buf[:0]
	l := p.BaseParser.PackVaruint(uint(ta.Height), varBuf)
	buf = append(buf, varBuf[:l]...)
	l = p.BaseParser.PackVaruint(uint(len(ta.Inputs)), varBuf)
	buf = append(buf, varBuf[:l]...)
	for i := range ta.Inputs {
		buf = p.AppendTxInput(&ta.Inputs[i], buf, varBuf)
	}
	l = p.BaseParser.PackVaruint(uint(len(ta.Outputs)), varBuf)
	buf = append(buf, varBuf[:l]...)
	for i := range ta.Outputs {
		buf = p.AppendTxOutput(&ta.Outputs[i], buf, varBuf)
	}
	return buf
}

func (p *BitcoinParser) UnpackTxAddresses(buf []byte) (*bchain.TxAddresses, error) {
	ta := bchain.TxAddresses{}
	height, l := p.BaseParser.UnpackVaruint(buf)
	ta.Height = uint32(height)
	inputs, ll := p.BaseParser.UnpackVaruint(buf[l:])
	l += ll
	ta.Inputs = make([]bchain.TxInput, inputs)
	for i := uint(0); i < inputs; i++ {
		l += p.UnpackTxInput(&ta.Inputs[i], buf[l:])
	}
	outputs, ll := p.BaseParser.UnpackVaruint(buf[l:])
	l += ll
	ta.Outputs = make([]bchain.TxOutput, outputs)
	for i := uint(0); i < outputs; i++ {
		l += p.UnpackTxOutput(&ta.Outputs[i], buf[l:])
	}
	return &ta, nil
}

func (p *BitcoinParser) AppendTxInput(txi *bchain.TxInput, buf []byte, varBuf []byte) []byte {
	la := len(txi.AddrDesc)
	l := p.BaseParser.PackVaruint(uint(la), varBuf)
	buf = append(buf, varBuf[:l]...)
	buf = append(buf, txi.AddrDesc...)
	l = p.BaseParser.PackBigint(&txi.ValueSat, varBuf)
	buf = append(buf, varBuf[:l]...)
	return buf
}

func (p *BitcoinParser) AppendTxOutput(txo *bchain.TxOutput, buf []byte, varBuf []byte) []byte {
	la := len(txo.AddrDesc)
	if txo.Spent {
		la = ^la
	}
	l := p.BaseParser.PackVarint(la, varBuf)
	buf = append(buf, varBuf[:l]...)
	buf = append(buf, txo.AddrDesc...)
	l = p.BaseParser.PackBigint(&txo.ValueSat, varBuf)
	buf = append(buf, varBuf[:l]...)
	return buf
}


func (p *BitcoinParser) UnpackTxInput(ti *bchain.TxInput, buf []byte) int {
	al, l := p.BaseParser.UnpackVaruint(buf)
	ti.AddrDesc = append([]byte(nil), buf[l:l+int(al)]...)
	al += uint(l)
	ti.ValueSat, l = p.BaseParser.UnpackBigint(buf[al:])
	return l + int(al)
}

func (p *BitcoinParser) UnpackTxOutput(to *bchain.TxOutput, buf []byte) int {
	al, l := p.BaseParser.UnpackVarint(buf)
	if al < 0 {
		to.Spent = true
		al = ^al
	}
	to.AddrDesc = append([]byte(nil), buf[l:l+al]...)
	al += l
	to.ValueSat, l = p.BaseParser.UnpackBigint(buf[al:])
	return l + al
}

func (p *BitcoinParser) PackOutpoints(outpoints []bchain.DbOutpoint) []byte {
	buf := make([]byte, 0, 32)
	bvout := make([]byte, vlq.MaxLen32)
	for _, o := range outpoints {
		l := p.BaseParser.PackVarint32(o.Index, bvout)
		buf = append(buf, []byte(o.BtxID)...)
		buf = append(buf, bvout[:l]...)
	}
	return buf
}

func (p *BitcoinParser) UnpackNOutpoints(buf []byte) ([]bchain.DbOutpoint, int, error) {
	txidUnpackedLen := p.BaseParser.PackedTxidLen()
	n, m := p.BaseParser.UnpackVaruint(buf)
	outpoints := make([]bchain.DbOutpoint, n)
	for i := uint(0); i < n; i++ {
		if m+txidUnpackedLen >= len(buf) {
			return nil, 0, errors.New("Inconsistent data in UnpackNOutpoints")
		}
		btxID := append([]byte(nil), buf[m:m+txidUnpackedLen]...)
		m += txidUnpackedLen
		vout, voutLen := p.BaseParser.UnpackVarint32(buf[m:])
		m += voutLen
		outpoints[i] = bchain.DbOutpoint{
			BtxID: btxID,
			Index: vout,
		}
	}
	return outpoints, m, nil
}

// Block index

func (p *BitcoinParser) PackBlockInfo(block *bchain.DbBlockInfo) ([]byte, error) {
	packed := make([]byte, 0, 64)
	varBuf := make([]byte, vlq.MaxLen64)
	b, err := p.BaseParser.PackBlockHash(block.Hash)
	if err != nil {
		return nil, err
	}
	pl := p.BaseParser.PackedTxidLen()
	if len(b) != pl {
		glog.Warning("Non standard block hash for height ", block.Height, ", hash [", block.Hash, "]")
		if len(b) > pl {
			b = b[:pl]
		} else {
			b = append(b, make([]byte, pl-len(b))...)
		}
	}
	packed = append(packed, b...)
	packed = append(packed, p.BaseParser.PackUint(uint32(block.Time))...)
	l := p.BaseParser.PackVaruint(uint(block.Txs), varBuf)
	packed = append(packed, varBuf[:l]...)
	l = p.BaseParser.PackVaruint(uint(block.Size), varBuf)
	packed = append(packed, varBuf[:l]...)
	return packed, nil
}

func (p *BitcoinParser) UnpackBlockInfo(buf []byte) (*bchain.DbBlockInfo, error) {
	pl := p.BaseParser.PackedTxidLen()
	// minimum length is PackedTxidLen + 4 bytes time + 1 byte txs + 1 byte size
	if len(buf) < pl+4+2 {
		return nil, nil
	}
	txid, err := p.BaseParser.UnpackBlockHash(buf[:pl])
	if err != nil {
		return nil, err
	}
	t := p.BaseParser.UnpackUint(buf[pl:])
	txs, l := p.BaseParser.UnpackVaruint(buf[pl+4:])
	size, _ := p.BaseParser.UnpackVaruint(buf[pl+4+l:])
	return &bchain.DbBlockInfo{
		Hash: txid,
		Time: int64(t),
		Txs:  uint32(txs),
		Size: uint32(size),
	}, nil
}