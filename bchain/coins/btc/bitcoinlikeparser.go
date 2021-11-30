package btc

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"math/big"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"

	vlq "github.com/bsm/go-vlq"
	"github.com/golang/glog"
	"github.com/juju/errors"
	"github.com/martinboehm/btcd/blockchain"
	"github.com/martinboehm/btcd/btcec"
	"github.com/martinboehm/btcd/wire"
	"github.com/martinboehm/btcutil"
	"github.com/martinboehm/btcutil/chaincfg"
	"github.com/martinboehm/btcutil/hdkeychain"
	"github.com/martinboehm/btcutil/txscript"
	"github.com/syscoin/blockbook/bchain"
)

// OutputScriptToAddressesFunc converts ScriptPubKey to bitcoin addresses
type OutputScriptToAddressesFunc func(script []byte) ([]string, bool, error)

// BitcoinLikeParser handle
type BitcoinLikeParser struct {
	*bchain.BaseParser
	Params                       *chaincfg.Params
	OutputScriptToAddressesFunc  OutputScriptToAddressesFunc
	XPubMagic                    uint32
	XPubMagicSegwitP2sh          uint32
	XPubMagicSegwitNative        uint32
	Slip44                       uint32
	minimumCoinbaseConfirmations int
}

// NewBitcoinLikeParser returns new BitcoinLikeParser instance
func NewBitcoinLikeParser(params *chaincfg.Params, c *Configuration) *BitcoinLikeParser {
	p := &BitcoinLikeParser{
		BaseParser: &bchain.BaseParser{
			BlockAddressesToKeep: c.BlockAddressesToKeep,
			AmountDecimalPoint:   8,
		},
		Params:                       params,
		XPubMagic:                    c.XPubMagic,
		XPubMagicSegwitP2sh:          c.XPubMagicSegwitP2sh,
		XPubMagicSegwitNative:        c.XPubMagicSegwitNative,
		Slip44:                       c.Slip44,
		minimumCoinbaseConfirmations: c.MinimumCoinbaseConfirmations,
	}
	p.OutputScriptToAddressesFunc = p.outputScriptToAddresses
	return p
}

// GetAddrDescFromVout returns internal address representation (descriptor) of given transaction output
func (p *BitcoinLikeParser) GetAddrDescFromVout(output *bchain.Vout) (bchain.AddressDescriptor, error) {
	ad, err := hex.DecodeString(output.ScriptPubKey.Hex)
	if err != nil {
		return ad, err
	}
	// convert possible P2PK script to P2PKH
	// so that all transactions by given public key are indexed together
	return txscript.ConvertP2PKtoP2PKH(p.Params.Base58CksumHasher, ad)
}

// GetAddrDescFromAddress returns internal address representation (descriptor) of given address
func (p *BitcoinLikeParser) GetAddrDescFromAddress(address string) (bchain.AddressDescriptor, error) {
	return p.addressToOutputScript(address)
}

// GetAddressesFromAddrDesc returns addresses for given address descriptor with flag if the addresses are searchable
func (p *BitcoinLikeParser) GetAddressesFromAddrDesc(addrDesc bchain.AddressDescriptor) ([]string, bool, error) {
	return p.OutputScriptToAddressesFunc(addrDesc)
}

// GetScriptFromAddrDesc returns output script for given address descriptor
func (p *BitcoinLikeParser) GetScriptFromAddrDesc(addrDesc bchain.AddressDescriptor) ([]byte, error) {
	return addrDesc, nil
}

// IsAddrDescIndexable returns true if AddressDescriptor should be added to index
// empty or OP_RETURN scripts are not indexed
func (p *BitcoinLikeParser) IsAddrDescIndexable(addrDesc bchain.AddressDescriptor) bool {
	if len(addrDesc) == 0 || addrDesc[0] == txscript.OP_RETURN {
		return false
	}
	return true
}

// addressToOutputScript converts bitcoin address to ScriptPubKey
func (p *BitcoinLikeParser) addressToOutputScript(address string) ([]byte, error) {
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

// TryParseOPReturn tries to process OP_RETURN script and return its string representation
func (p *BitcoinLikeParser) TryParseOPReturn(script []byte) string {
	if len(script) > 1 && script[0] == txscript.OP_RETURN {
		// trying 2 variants of OP_RETURN data
		// 1) OP_RETURN OP_PUSHDATA1 <datalen> <data>
		// 2) OP_RETURN <datalen> <data>
		// 3) OP_RETURN OP_PUSHDATA2 <datalenlow> <datalenhigh> <data>
		var data []byte
		var l int
		if script[1] == txscript.OP_PUSHDATA1 && len(script) > 2 {
			l = int(script[2])
			data = script[3:]
			if l != len(data) {
				l = int(script[1])
				data = script[2:]
			}
		} else if script[1] == txscript.OP_PUSHDATA2 && len(script) > 3 {
			l = int(script[2]) + int(script[3])<<8
			data = script[4:]
		} else {
			l = int(script[1])
			data = script[2:]
		}
		if l == len(data) {
			var ed string

			ed = p.tryParseOmni(data)
			if ed != "" {
				return ed
			}

			if utf8.Valid(data) {
				ed = "(" + string(data) + ")"
			} else {
				ed = hex.EncodeToString(data)
			}
			return "OP_RETURN " + ed
		}
	}
	return ""
}

var omniCurrencyMap = map[uint32]string{
	1:  "Omni",
	2:  "Test Omni",
	31: "TetherUS",
}

// tryParseOmni tries to extract Omni simple send transaction from script
func (p *BitcoinLikeParser) tryParseOmni(data []byte) string {

	// currently only simple send transaction version 0 is supported, see
	// https://github.com/OmniLayer/spec#transfer-coins-simple-send
	if len(data) != 20 || data[0] != 'o' {
		return ""
	}
	// omni (4) <tx_version> (2) <tx_type> (2)
	omniHeader := []byte{'o', 'm', 'n', 'i', 0, 0, 0, 0}
	if !bytes.Equal(data[0:8], omniHeader) {
		return ""
	}

	currencyID := binary.BigEndian.Uint32(data[8:12])
	currency, ok := omniCurrencyMap[currencyID]
	if !ok {
		return ""
	}
	amount := new(big.Int)
	amount.SetBytes(data[12:])
	amountStr := p.AmountToDecimalString(amount)

	ed := "OMNI Simple Send: " + amountStr + " " + currency + " (#" + strconv.Itoa(int(currencyID)) + ")"
	return ed
}

// outputScriptToAddresses converts ScriptPubKey to addresses with a flag that the addresses are searchable
func (p *BitcoinLikeParser) outputScriptToAddresses(script []byte) ([]string, bool, error) {
	sc, addresses, _, err := txscript.ExtractPkScriptAddrs(script, p.Params)
	if err != nil {
		return nil, false, err
	}
	rv := make([]string, len(addresses))
	for i, a := range addresses {
		rv[i] = a.EncodeAddress()
	}
	var s bool
	if sc == txscript.PubKeyHashTy || sc == txscript.WitnessV0PubKeyHashTy || sc == txscript.ScriptHashTy || sc == txscript.WitnessV0ScriptHashTy || sc == txscript.WitnessV1TaprootTy {
		s = true
	} else if len(rv) == 0 {
		or := p.TryParseOPReturn(script)
		if or != "" {
			rv = []string{or}
		}
	}
	return rv, s, nil
}

// TxFromMsgTx converts bitcoin wire Tx to bchain.Tx
func (p *BitcoinLikeParser) TxFromMsgTx(t *wire.MsgTx, parseAddresses bool) bchain.Tx {
	vin := make([]bchain.Vin, len(t.TxIn))
	for i, in := range t.TxIn {
		if blockchain.IsCoinBaseTx(t) {
			vin[i] = bchain.Vin{
				Coinbase: hex.EncodeToString(in.SignatureScript),
				Sequence: in.Sequence,
			}
			break
		}
		s := bchain.ScriptSig{
			Hex: hex.EncodeToString(in.SignatureScript),
			// missing: Asm,
		}
		vin[i] = bchain.Vin{
			Txid:      in.PreviousOutPoint.Hash.String(),
			Vout:      in.PreviousOutPoint.Index,
			Sequence:  in.Sequence,
			ScriptSig: s,
		}
	}
	vout := make([]bchain.Vout, len(t.TxOut))
	for i, out := range t.TxOut {
		addrs := []string{}
		if parseAddresses {
			addrs, _, _ = p.OutputScriptToAddressesFunc(out.PkScript)
		}
		s := bchain.ScriptPubKey{
			Hex:       hex.EncodeToString(out.PkScript),
			Addresses: addrs,
			// missing: Asm,
			// missing: Type,
		}
		var vs big.Int
		vs.SetInt64(out.Value)
		vout[i] = bchain.Vout{
			ValueSat:     vs,
			N:            uint32(i),
			ScriptPubKey: s,
		}
	}
	tx := bchain.Tx{
		Txid:     t.TxHash().String(),
		Version:  t.Version,
		LockTime: t.LockTime,
		Vin:      vin,
		Vout:     vout,
		// skip: BlockHash,
		// skip: Confirmations,
		// skip: Time,
		// skip: Blocktime,
	}
	return tx
}

// ParseTx parses byte array containing transaction and returns Tx struct
func (p *BitcoinLikeParser) ParseTx(b []byte) (*bchain.Tx, error) {
	t := wire.MsgTx{}
	r := bytes.NewReader(b)
	if err := t.Deserialize(r); err != nil {
		return nil, err
	}
	tx := p.TxFromMsgTx(&t, true)
	tx.Hex = hex.EncodeToString(b)
	return &tx, nil
}

// ParseBlock parses raw block to our Block struct
func (p *BitcoinLikeParser) ParseBlock(b []byte) (*bchain.Block, error) {
	w := wire.MsgBlock{}
	r := bytes.NewReader(b)

	if err := w.Deserialize(r); err != nil {
		return nil, err
	}

	txs := make([]bchain.Tx, len(w.Transactions))
	for ti, t := range w.Transactions {
		txs[ti] = p.TxFromMsgTx(t, false)
	}

	return &bchain.Block{
		BlockHeader: bchain.BlockHeader{
			Size: len(b),
			Time: w.Header.Timestamp.Unix(),
		},
		Txs: txs,
	}, nil
}

// PackTx packs transaction to byte array
func (p *BitcoinLikeParser) PackTx(tx *bchain.Tx, height uint32, blockTime int64) ([]byte, error) {
	buf := make([]byte, 4+vlq.MaxLen64+len(tx.Hex)/2)
	binary.BigEndian.PutUint32(buf[0:4], height)
	vl := vlq.PutInt(buf[4:4+vlq.MaxLen64], blockTime)
	hl, err := hex.Decode(buf[4+vl:], []byte(tx.Hex))
	return buf[0 : 4+vl+hl], err
}

// UnpackTx unpacks transaction from byte array
func (p *BitcoinLikeParser) UnpackTx(buf []byte) (*bchain.Tx, uint32, error) {
	height := binary.BigEndian.Uint32(buf)
	bt, l := vlq.Int(buf[4:])
	tx, err := p.ParseTx(buf[4+l:])
	if err != nil {
		return nil, 0, err
	}
	tx.Blocktime = bt

	return tx, height, nil
}

// MinimumCoinbaseConfirmations returns minimum number of confirmations a coinbase transaction must have before it can be spent
func (p *BitcoinLikeParser) MinimumCoinbaseConfirmations() int {
	return p.minimumCoinbaseConfirmations
}

var tapTweakTagHash = sha256.Sum256([]byte("TapTweak"))

func tapTweakHash(msg []byte) []byte {
	tagLen := len(tapTweakTagHash)
	m := make([]byte, tagLen*2+len(msg))
	copy(m[:tagLen], tapTweakTagHash[:])
	copy(m[tagLen:tagLen*2], tapTweakTagHash[:])
	copy(m[tagLen*2:], msg)
	h := sha256.Sum256(m)
	return h[:]
}

func (p *BitcoinLikeParser) taprootAddrFromExtKey(extKey *hdkeychain.ExtendedKey) (*btcutil.AddressWitnessTaproot, error) {
	curve := btcec.S256()
	t := new(big.Int)

	// tweak the derived pubkey to the output pub key according to https://en.bitcoin.it/wiki/BIP_0341
	// and https://github.com/bitcoin/bips/blob/master/bip-0086.mediawiki
	derived_key := extKey.PubKeyBytes()[1:]

	t.SetBytes(tapTweakHash(derived_key))
	// Fail if t >=order of the base point
	if t.Cmp(curve.N) >= 0 {
		return nil, errors.New("greater than or equal to curve order")
	}
	// Q = point_add(lift_x(int_from_bytes(pubkey)), point_mul(G, t))
	ipx, ipy, err := btcec.LiftX(derived_key)
	if err != nil {
		return nil, err
	}
	tGx, tGy := curve.ScalarBaseMult(t.Bytes())
	output_pubkey, _ := curve.Add(ipx, ipy, tGx, tGy)
	//
	b := output_pubkey.Bytes()
	// the x coordinate on the curve can be a number small enough that it does not need 32 bytes required for the output script
	if len(b) < 32 {
		b = make([]byte, 32)
		output_pubkey.FillBytes(b)
	}
	return btcutil.NewAddressWitnessTaproot(b, p.Params)
}

func (p *BitcoinLikeParser) addrDescFromExtKey(extKey *hdkeychain.ExtendedKey, descriptor *bchain.XpubDescriptor) (bchain.AddressDescriptor, error) {
	var a btcutil.Address
	var err error
	switch descriptor.Type {
	case bchain.P2PKH:
		a, err = extKey.Address(p.Params)
	case bchain.P2SHWPKH:
		// redeemScript <witness version: OP_0><len pubKeyHash: 20><20-byte-pubKeyHash>
		pubKeyHash := btcutil.Hash160(extKey.PubKeyBytes())
		redeemScript := make([]byte, len(pubKeyHash)+2)
		redeemScript[0] = 0
		redeemScript[1] = byte(len(pubKeyHash))
		copy(redeemScript[2:], pubKeyHash)
		hash := btcutil.Hash160(redeemScript)
		a, err = btcutil.NewAddressScriptHashFromHash(hash, p.Params)
	case bchain.P2WPKH:
		a, err = btcutil.NewAddressWitnessPubKeyHash(btcutil.Hash160(extKey.PubKeyBytes()), p.Params)
	case bchain.P2TR:
		a, err = p.taprootAddrFromExtKey(extKey)
	default:
		return nil, errors.New("Unsupported xpub descriptor type")
	}
	if err != nil {
		return nil, err
	}
	return txscript.PayToAddrScript(a)
}

func (p *BitcoinLikeParser) xpubDescriptorFromXpub(xpub string) (*bchain.XpubDescriptor, error) {
	var descriptor bchain.XpubDescriptor
	extKey, err := hdkeychain.NewKeyFromString(xpub, p.Params.Base58CksumHasher)
	if err != nil {
		return nil, err
	}
	descriptor.Xpub = xpub
	descriptor.XpubDescriptor = xpub
	if extKey.Version() == p.XPubMagicSegwitP2sh {
		descriptor.Type = bchain.P2SHWPKH
		descriptor.Bip = "49"
	} else if extKey.Version() == p.XPubMagicSegwitNative {
		descriptor.Type = bchain.P2WPKH
		descriptor.Bip = "84"
	} else {
		descriptor.Type = bchain.P2PKH
		descriptor.Bip = "44"
	}
	descriptor.ChangeIndexes = []uint32{0, 1}
	descriptor.ExtKey = extKey
	return &descriptor, nil
}

var (
	xpubDesriptorRegex     *regexp.Regexp
	typeSubexpIndex        int
	bipSubexpIndex         int
	xpubSubexpIndex        int
	changeSubexpIndex      int
	changeList1SubexpIndex int
	changeList2SubexpIndex int
)

func init() {
	xpubDesriptorRegex, _ = regexp.Compile(`^(?P<type>(sh\(wpkh|wpkh|pk|pkh|wpkh|wsh|tr))\((\[\w+/(?P<bip>\d+)'/\d+'?/\d+'?\])?(?P<xpub>\w+)(/(({(?P<changelist1>\d+(,\d+)*)})|(<(?P<changelist2>\d+(;\d+)*)>)|(?P<change>\d+))/\*)?\)+`)
	typeSubexpIndex = xpubDesriptorRegex.SubexpIndex("type")
	bipSubexpIndex = xpubDesriptorRegex.SubexpIndex("bip")
	xpubSubexpIndex = xpubDesriptorRegex.SubexpIndex("xpub")
	changeList1SubexpIndex = xpubDesriptorRegex.SubexpIndex("changelist1")
	changeList2SubexpIndex = xpubDesriptorRegex.SubexpIndex("changelist2")
	changeSubexpIndex = xpubDesriptorRegex.SubexpIndex("change")
	if changeSubexpIndex < 0 {
		panic("Invalid bitcoinparser xpubDesriptorRegex")
	}
}

// ParseXpub parses xpub (or xpub descriptor) and returns XpubDescriptor
func (p *BitcoinLikeParser) ParseXpub(xpub string) (*bchain.XpubDescriptor, error) {
	match := xpubDesriptorRegex.FindStringSubmatch(xpub)
	if len(match) > changeSubexpIndex {
		var descriptor bchain.XpubDescriptor
		descriptor.XpubDescriptor = xpub
		m := match[typeSubexpIndex]
		switch m {
		case "pkh":
			descriptor.Type = bchain.P2PKH
			descriptor.Bip = "44"
		case "sh(wpkh":
			descriptor.Type = bchain.P2SHWPKH
			descriptor.Bip = "49"
		case "wpkh":
			descriptor.Type = bchain.P2WPKH
			descriptor.Bip = "84"
		case "tr":
			descriptor.Type = bchain.P2TR
			descriptor.Bip = "86"
		default:
			return nil, errors.Errorf("Xpub descriptor %s is not supported", m)
		}
		if len(match[bipSubexpIndex]) > 0 {
			descriptor.Bip = match[bipSubexpIndex]
		}
		descriptor.Xpub = match[xpubSubexpIndex]
		extKey, err := hdkeychain.NewKeyFromString(descriptor.Xpub, p.Params.Base58CksumHasher)
		if err != nil {
			return nil, err
		}
		descriptor.ExtKey = extKey
		if len(match[changeSubexpIndex]) > 0 {
			change, err := strconv.ParseUint(match[changeSubexpIndex], 10, 32)
			if err != nil {
				return nil, err
			}
			descriptor.ChangeIndexes = []uint32{uint32(change)}
		} else {
			if len(match[changeList1SubexpIndex]) > 0 || len(match[changeList2SubexpIndex]) > 0 {
				var changes []string
				if len(match[changeList1SubexpIndex]) > 0 {
					changes = strings.Split(match[changeList1SubexpIndex], ",")
				} else {
					changes = strings.Split(match[changeList2SubexpIndex], ";")
				}
				if len(changes) == 0 {
					return nil, errors.New("Invalid xpub descriptor, cannot parse change")
				}
				descriptor.ChangeIndexes = make([]uint32, len(changes))
				for i, ch := range changes {
					change, err := strconv.ParseUint(ch, 10, 32)
					if err != nil {
						return nil, err
					}
					descriptor.ChangeIndexes[i] = uint32(change)

				}
			} else {
				// default to {0,1}
				descriptor.ChangeIndexes = []uint32{0, 1}
			}

		}
		return &descriptor, nil
	}
	return p.xpubDescriptorFromXpub(xpub)

}

// DeriveAddressDescriptors derives address descriptors from given xpub for listed indexes
func (p *BitcoinLikeParser) DeriveAddressDescriptors(descriptor *bchain.XpubDescriptor, change uint32, indexes []uint32) ([]bchain.AddressDescriptor, error) {
	ad := make([]bchain.AddressDescriptor, len(indexes))
	changeExtKey, err := descriptor.ExtKey.(*hdkeychain.ExtendedKey).Derive(change)
	if err != nil {
		return nil, err
	}
	for i, index := range indexes {
		indexExtKey, err := changeExtKey.Derive(index)
		if err != nil {
			return nil, err
		}
		ad[i], err = p.addrDescFromExtKey(indexExtKey, descriptor)
		if err != nil {
			return nil, err
		}
	}
	return ad, nil
}

// DeriveAddressDescriptorsFromTo derives address descriptors from given xpub for addresses in index range
func (p *BitcoinLikeParser) DeriveAddressDescriptorsFromTo(descriptor *bchain.XpubDescriptor, change uint32, fromIndex uint32, toIndex uint32) ([]bchain.AddressDescriptor, error) {
	if toIndex <= fromIndex {
		return nil, errors.New("toIndex<=fromIndex")
	}
	changeExtKey, err := descriptor.ExtKey.(*hdkeychain.ExtendedKey).Derive(change)
	if err != nil {
		return nil, err
	}
	ad := make([]bchain.AddressDescriptor, toIndex-fromIndex)
	for index := fromIndex; index < toIndex; index++ {
		indexExtKey, err := changeExtKey.Derive(index)
		if err != nil {
			return nil, err
		}
		ad[index-fromIndex], err = p.addrDescFromExtKey(indexExtKey, descriptor)
		if err != nil {
			return nil, err
		}
	}
	return ad, nil
}

// DerivationBasePath returns base path of xpub
func (p *BitcoinLikeParser) DerivationBasePath(descriptor *bchain.XpubDescriptor) (string, error) {
	var c string
	extKey := descriptor.ExtKey.(*hdkeychain.ExtendedKey)
	cn := extKey.ChildNum()
	if cn >= 0x80000000 {
		cn -= 0x80000000
		c = "'"
	}
	c = strconv.Itoa(int(cn)) + c
	if extKey.Depth() != 3 {
		return "unknown/" + c, nil
	}
	return "m/" + descriptor.Bip + "'/" + strconv.Itoa(int(p.Slip44)) + "'/" + c, nil
}
func (p *BitcoinLikeParser) PackAddrBalance(ab *bchain.AddrBalance, buf, varBuf []byte) []byte {
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

func (p *BitcoinLikeParser) UnpackAddrBalance(buf []byte, txidUnpackedLen int, detail bchain.AddressBalanceDetail) (*bchain.AddrBalance, error) {
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

func (p *BitcoinLikeParser) PackTxAddresses(ta *bchain.TxAddresses, buf []byte, varBuf []byte) []byte {
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

func (p *BitcoinLikeParser) UnpackTxAddresses(buf []byte) (*bchain.TxAddresses, error) {
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

func (p *BitcoinLikeParser) AppendTxInput(txi *bchain.TxInput, buf []byte, varBuf []byte) []byte {
	la := len(txi.AddrDesc)
	l := p.BaseParser.PackVaruint(uint(la), varBuf)
	buf = append(buf, varBuf[:l]...)
	buf = append(buf, txi.AddrDesc...)
	l = p.BaseParser.PackBigint(&txi.ValueSat, varBuf)
	buf = append(buf, varBuf[:l]...)
	return buf
}

func (p *BitcoinLikeParser) AppendTxOutput(txo *bchain.TxOutput, buf []byte, varBuf []byte) []byte {
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


func (p *BitcoinLikeParser) UnpackTxInput(ti *bchain.TxInput, buf []byte) int {
	al, l := p.BaseParser.UnpackVaruint(buf)
	ti.AddrDesc = append([]byte(nil), buf[l:l+int(al)]...)
	al += uint(l)
	ti.ValueSat, l = p.BaseParser.UnpackBigint(buf[al:])
	return l + int(al)
}

func (p *BitcoinLikeParser) UnpackTxOutput(to *bchain.TxOutput, buf []byte) int {
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

func (p *BitcoinLikeParser) PackOutpoints(outpoints []bchain.DbOutpoint) []byte {
	buf := make([]byte, 0, 32)
	bvout := make([]byte, vlq.MaxLen32)
	for _, o := range outpoints {
		l := p.BaseParser.PackVarint32(o.Index, bvout)
		buf = append(buf, []byte(o.BtxID)...)
		buf = append(buf, bvout[:l]...)
	}
	return buf
}

func (p *BitcoinLikeParser) UnpackNOutpoints(buf []byte) ([]bchain.DbOutpoint, int, error) {
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

func (p *BitcoinLikeParser) PackBlockInfo(block *bchain.DbBlockInfo) ([]byte, error) {
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

func (p *BitcoinLikeParser) UnpackBlockInfo(buf []byte) (*bchain.DbBlockInfo, error) {
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
