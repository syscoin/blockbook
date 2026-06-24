package bchain

import (
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"math/big"
	"strconv"
	"strings"

	vlq "github.com/bsm/go-vlq"
	"github.com/golang/glog"
	"github.com/juju/errors"
	"github.com/trezor/blockbook/common"
	"google.golang.org/protobuf/proto"
)

// BaseParser implements data parsing/handling functionality base for all other parsers
type BaseParser struct {
	BlockAddressesToKeep int
	AmountDecimalPoint   int
	AddressAliases       bool
}

// ParseBlock parses raw block to our Block struct - currently not implemented
func (p *BaseParser) ParseBlock(b []byte) (*Block, error) {
	return nil, errors.New("ParseBlock: not implemented")
}

// ParseTx parses byte array containing transaction and returns Tx struct - currently not implemented
func (p *BaseParser) ParseTx(b []byte) (*Tx, error) {
	return nil, errors.New("ParseTx: not implemented")
}

// GetAddrDescForUnknownInput returns nil AddressDescriptor
func (p *BaseParser) GetAddrDescForUnknownInput(tx *Tx, input int) AddressDescriptor {
	var iTxid string
	if len(tx.Vin) > input {
		iTxid = tx.Vin[input].Txid
	}
	glog.Warningf("tx %v, input tx %v not found in txAddresses", tx.Txid, iTxid)
	return nil
}

const (
	zeros                   = "0000000000000000000000000000000000000000"
	maxAmountExpandedDigits = 1024
)

// AmountToBigInt converts amount in common.JSONNumber (string) to big.Int
// it uses string operations to avoid problems with rounding
func (p *BaseParser) AmountToBigInt(n common.JSONNumber) (big.Int, error) {
	var r big.Int
	d := min(p.AmountDecimalPoint, len(zeros))
	if d < 0 {
		d = 0
	}
	s := string(n)
	if strings.IndexAny(s, "eE") == -1 {
		s = normalizePlainAmountToIntString(s, d)
	} else {
		var err error
		s, err = normalizeScientificAmountToIntString(s, d)
		if err != nil {
			return r, errors.New("AmountToBigInt: failed to convert")
		}
	}
	if _, ok := r.SetString(s, 10); !ok {
		return r, errors.New("AmountToBigInt: failed to convert")
	}
	return r, nil
}

func normalizePlainAmountToIntString(s string, decimalPoint int) string {
	i := strings.IndexByte(s, '.')
	if i == -1 {
		return s + zeros[:decimalPoint]
	}
	z := decimalPoint - len(s) + i + 1
	if z > 0 {
		return s[:i] + s[i+1:] + zeros[:z]
	}
	return s[:i] + s[i+1:len(s)+z]
}

func normalizeScientificAmountToIntString(s string, decimalPoint int) (string, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		s = "0"
	}

	sign := ""
	if strings.HasPrefix(s, "-") {
		sign = "-"
		s = s[1:]
	} else if strings.HasPrefix(s, "+") {
		s = s[1:]
	}
	if s == "" {
		return "", errors.New("empty mantissa")
	}

	exponent := 0
	if i := strings.IndexAny(s, "eE"); i != -1 {
		if strings.IndexAny(s[i+1:], "eE") != -1 {
			return "", errors.New("invalid scientific notation")
		}
		var err error
		exponent, err = strconv.Atoi(s[i+1:])
		if err != nil {
			return "", err
		}
		s = s[:i]
		if s == "" {
			return "", errors.New("empty mantissa")
		}
	}

	fractionDigits := 0
	if i := strings.IndexByte(s, '.'); i != -1 {
		if strings.IndexByte(s[i+1:], '.') != -1 {
			return "", errors.New("invalid decimal notation")
		}
		fractionDigits = len(s) - i - 1
		s = s[:i] + s[i+1:]
	}
	if s == "" {
		return "", errors.New("empty value")
	}
	for _, c := range s {
		if c < '0' || c > '9' {
			return "", errors.New("invalid value")
		}
	}

	s = strings.TrimLeft(s, "0")
	if s == "" {
		return "0", nil
	}

	shift := exponent - fractionDigits + decimalPoint
	if shift >= 0 {
		if shift > maxAmountExpandedDigits || len(s) > maxAmountExpandedDigits-shift {
			return "", errors.New("expanded value too large")
		}
		s = s + strings.Repeat("0", shift)
	} else {
		keep := len(s) + shift
		if keep > 0 {
			s = s[:keep]
		} else {
			s = "0"
		}
	}

	if sign == "-" && s != "0" {
		s = sign + s
	}
	return s, nil
}

// AmountToDecimalString converts amount in big.Int to string with decimal point in the place defined by the parameter d
func AmountToDecimalString(a *big.Int, d int) string {
	if a == nil {
		return ""
	}
	n := a.String()
	var s string
	if n[0] == '-' {
		n = n[1:]
		s = "-"
	}
	if d > len(zeros) {
		d = len(zeros)
	}
	if len(n) <= d {
		n = zeros[:d-len(n)+1] + n
	}
	i := len(n) - d
	ad := strings.TrimRight(n[i:], "0")
	if len(ad) > 0 {
		n = n[:i] + "." + ad
	} else {
		n = n[:i]
	}
	return s + n
}

// AmountToDecimalString converts amount in big.Int to string with decimal point in the correct place
func (p *BaseParser) AmountToDecimalString(a *big.Int) string {
	return AmountToDecimalString(a, p.AmountDecimalPoint)
}

// AmountDecimals returns number of decimal places in amounts
func (p *BaseParser) AmountDecimals() int {
	return p.AmountDecimalPoint
}

// UseAddressAliases returns true if address aliases are enabled
func (p *BaseParser) UseAddressAliases() bool {
	return p.AddressAliases
}

// ParseTxFromJson parses JSON message containing transaction and returns Tx struct
func (p *BaseParser) ParseTxFromJson(msg json.RawMessage) (*Tx, error) {
	var tx Tx
	err := json.Unmarshal(msg, &tx)
	if err != nil {
		return nil, err
	}

	for i := range tx.Vout {
		vout := &tx.Vout[i]
		// convert vout.JsonValue to big.Int and clear it, it is only temporary value used for unmarshal
		vout.ValueSat, err = p.AmountToBigInt(vout.JsonValue)
		if err != nil {
			return nil, err
		}
		vout.JsonValue = ""
	}

	return &tx, nil
}

// PackedTxidLen returns length in bytes of packed txid
func (p *BaseParser) PackedTxidLen() int {
	return 32
}

// KeepBlockAddresses returns number of blocks which are to be kept in blockaddresses column
func (p *BaseParser) KeepBlockAddresses() int {
	return p.BlockAddressesToKeep
}

// PackTxid packs txid to byte array
func (p *BaseParser) PackTxid(txid string) ([]byte, error) {
	if txid == "" {
		return nil, ErrTxidMissing
	}
	return hex.DecodeString(txid)
}

// UnpackTxid unpacks byte array to txid
func (p *BaseParser) UnpackTxid(buf []byte) (string, error) {
	return hex.EncodeToString(buf), nil
}

// PackBlockHash packs block hash to byte array
func (p *BaseParser) PackBlockHash(hash string) ([]byte, error) {
	return hex.DecodeString(hash)
}

// UnpackBlockHash unpacks byte array to block hash
func (p *BaseParser) UnpackBlockHash(buf []byte) (string, error) {
	return hex.EncodeToString(buf), nil
}

// GetChainType is type of the blockchain, default is ChainBitcoinType
func (p *BaseParser) GetChainType() ChainType {
	return ChainBitcoinType
}

// MinimumCoinbaseConfirmations returns minimum number of confirmations a coinbase transaction must have before it can be spent
func (p *BaseParser) MinimumCoinbaseConfirmations() int {
	return 0
}

// SupportsVSize returns true if vsize of a transaction should be computed and returned by API
func (p *BaseParser) SupportsVSize() bool {
	return false
}

// PackTx packs transaction to byte array using protobuf
func (p *BaseParser) PackTx(tx *Tx, height uint32, blockTime int64) ([]byte, error) {
	var err error
	pti := make([]*ProtoTransaction_VinType, len(tx.Vin))
	for i, vi := range tx.Vin {
		hex, err := hex.DecodeString(vi.ScriptSig.Hex)
		if err != nil {
			return nil, errors.Annotatef(err, "Vin %v Hex %v", i, vi.ScriptSig.Hex)
		}
		// coinbase txs do not have Vin.txid
		itxid, err := p.PackTxid(vi.Txid)
		if err != nil && err != ErrTxidMissing {
			return nil, errors.Annotatef(err, "Vin %v Txid %v", i, vi.Txid)
		}
		pti[i] = &ProtoTransaction_VinType{
			Addresses:    vi.Addresses,
			Coinbase:     vi.Coinbase,
			ScriptSigHex: hex,
			Sequence:     vi.Sequence,
			Txid:         itxid,
			Vout:         vi.Vout,
		}
	}
	pto := make([]*ProtoTransaction_VoutType, len(tx.Vout))
	for i, vo := range tx.Vout {
		hex, err := hex.DecodeString(vo.ScriptPubKey.Hex)
		if err != nil {
			return nil, errors.Annotatef(err, "Vout %v Hex %v", i, vo.ScriptPubKey.Hex)
		}
		pto[i] = &ProtoTransaction_VoutType{
			Addresses:       vo.ScriptPubKey.Addresses,
			N:               vo.N,
			ScriptPubKeyHex: hex,
			ValueSat:        vo.ValueSat.Bytes(),
		}
	}
	pt := &ProtoTransaction{
		Blocktime: uint64(blockTime),
		Height:    height,
		Locktime:  tx.LockTime,
		Vin:       pti,
		Vout:      pto,
		Version:   tx.Version,
		VSize:     tx.VSize,
	}
	if pt.Hex, err = hex.DecodeString(tx.Hex); err != nil {
		return nil, errors.Annotatef(err, "Hex %v", tx.Hex)
	}
	if pt.Txid, err = p.PackTxid(tx.Txid); err != nil {
		return nil, errors.Annotatef(err, "Txid %v", tx.Txid)
	}
	return proto.Marshal(pt)
}

// UnpackTx unpacks transaction from protobuf byte array
func (p *BaseParser) UnpackTx(buf []byte) (*Tx, uint32, error) {
	var pt ProtoTransaction
	err := proto.Unmarshal(buf, &pt)
	if err != nil {
		return nil, 0, err
	}
	txid, err := p.UnpackTxid(pt.Txid)
	if err != nil {
		return nil, 0, err
	}
	vin := make([]Vin, len(pt.Vin))
	for i, pti := range pt.Vin {
		itxid, err := p.UnpackTxid(pti.Txid)
		if err != nil {
			return nil, 0, err
		}
		vin[i] = Vin{
			Addresses: pti.Addresses,
			Coinbase:  pti.Coinbase,
			ScriptSig: ScriptSig{
				Hex: hex.EncodeToString(pti.ScriptSigHex),
			},
			Sequence: pti.Sequence,
			Txid:     itxid,
			Vout:     pti.Vout,
		}
	}
	vout := make([]Vout, len(pt.Vout))
	for i, pto := range pt.Vout {
		var vs big.Int
		vs.SetBytes(pto.ValueSat)
		vout[i] = Vout{
			N: pto.N,
			ScriptPubKey: ScriptPubKey{
				Addresses: pto.Addresses,
				Hex:       hex.EncodeToString(pto.ScriptPubKeyHex),
			},
			ValueSat: vs,
		}
	}
	tx := Tx{
		Blocktime: int64(pt.Blocktime),
		Hex:       hex.EncodeToString(pt.Hex),
		LockTime:  pt.Locktime,
		Time:      int64(pt.Blocktime),
		Txid:      txid,
		Vin:       vin,
		Vout:      vout,
		Version:   pt.Version,
		VSize:     pt.VSize,
	}
	return &tx, pt.Height, nil
}

// IsAddrDescIndexable returns true if AddressDescriptor should be added to index
// by default all AddressDescriptors are indexable
func (p *BaseParser) IsAddrDescIndexable(addrDesc AddressDescriptor) bool {
	return true
}

// ParseXpub is unsupported
func (p *BaseParser) ParseXpub(xpub string) (*XpubDescriptor, error) {
	return nil, errors.New("Not supported")
}

// DerivationBasePath is unsupported
func (p *BaseParser) DerivationBasePath(descriptor *XpubDescriptor) (string, error) {
	return "", errors.New("Not supported")
}

// DeriveAddressDescriptors is unsupported
func (p *BaseParser) DeriveAddressDescriptors(descriptor *XpubDescriptor, change uint32, indexes []uint32) ([]AddressDescriptor, error) {
	return nil, errors.New("Not supported")
}

// DeriveAddressDescriptorsFromTo is unsupported
func (p *BaseParser) DeriveAddressDescriptorsFromTo(descriptor *XpubDescriptor, change uint32, fromIndex uint32, toIndex uint32) ([]AddressDescriptor, error) {
	return nil, errors.New("Not supported")
}

// EthereumTypeGetTokenTransfersFromTx is unsupported
func (p *BaseParser) EthereumTypeGetTokenTransfersFromTx(tx *Tx) (TokenTransfers, error) {
	return nil, errors.New("Not supported")
}

// GetEthereumTxData returns default pending status for non-Ethereum-like chains.
func (p *BaseParser) GetEthereumTxData(tx *Tx) *EthereumTxData {
	return &EthereumTxData{Status: TxStatusPending}
}

// GetChainExtraData returns optional normalized chain-specific transaction data.
func (p *BaseParser) GetChainExtraData(tx *Tx) (json.RawMessage, error) {
	return nil, nil
}

// GetChainExtraPayloadType identifies the shape of normalized chain-specific transaction data.
func (p *BaseParser) GetChainExtraPayloadType() ChainExtraPayloadType {
	return ChainExtraPayloadTypeUnknown
}

// FormatAddressAlias makes possible to do coin specific formatting to an address alias
func (p *BaseParser) FormatAddressAlias(address string, name string) string {
	return name
}

func (b *BaseParser) ParseInputData(signatures *[]FourByteSignature, data string) *EthereumParsedInputData {
	return nil
}

// SYSCOIN: default no-op hooks for Syscoin SPT parsing/indexing. Non-Syscoin
// parsers keep upstream behavior and never enter these code paths.
func (p *BaseParser) IsSyscoinTx(nVersion int32, nHeight uint32) bool { return false }
func (p *BaseParser) IsSyscoinMintTx(nVersion int32) bool             { return false }
func (p *BaseParser) IsAssetAllocationTx(nVersion int32) bool         { return false }
func (p *BaseParser) GetAssetsMaskFromVersion(nVersion int32) AssetsMask {
	return BaseCoinMask
}
func (p *BaseParser) GetAssetTypeFromVersion(nVersion int32) *TokenType { return nil }
func (p *BaseParser) TryGetOPReturn(script []byte) []byte               { return nil }
func (p *BaseParser) GetMaxAddrLength() int                             { return 1024 }
func (p *BaseParser) PackAssetKey(assetGuid uint64, height uint32) []byte {
	return nil
}
func (p *BaseParser) UnpackAssetKey(buf []byte) (uint64, uint32) { return 0, 0 }
func (p *BaseParser) PackAssetTxIndex(txAsset *TxAsset) []byte   { return nil }
func (p *BaseParser) UnpackAssetTxIndex(buf []byte) []*TxAssetIndex {
	return nil
}
func (p *BaseParser) GetAssetAllocationFromData(sptData []byte, txVersion int32) (*AssetAllocation, []byte, error) {
	return nil, nil, errors.New("Not supported")
}
func (p *BaseParser) GetAssetAllocationFromDesc(addrDesc *AddressDescriptor, txVersion int32) (*AssetAllocation, []byte, error) {
	return nil, nil, errors.New("Not supported")
}
func (p *BaseParser) GetAllocationFromTx(tx *Tx) (*AssetAllocation, []byte, error) {
	return nil, nil, errors.New("Not supported")
}
func (p *BaseParser) LoadAssets(tx *Tx) error { return errors.New("Not supported") }
func (p *BaseParser) WitnessPubKeyHashFromKeyID(keyId []byte) (string, error) {
	return "", errors.New("Not supported")
}
func (p *BaseParser) AppendAssetInfo(assetInfo *AssetInfo, buf []byte, varBuf []byte) []byte {
	return nil
}
func (p *BaseParser) UnpackAssetInfo(assetInfo *AssetInfo, buf []byte) int { return 0 }

// SYSCOIN: legacy packing helpers used by the Syscoin SPT serializers.
func (p *BaseParser) PackUint(i uint32) []byte {
	buf := make([]byte, 4)
	binary.BigEndian.PutUint32(buf, i)
	return buf
}

func (p *BaseParser) UnpackUint(buf []byte) uint32 {
	return binary.BigEndian.Uint32(buf)
}

func (p *BaseParser) PackVarint32(i int32, buf []byte) int {
	return vlq.PutInt(buf, int64(i))
}

func (p *BaseParser) PackVaruint(i uint, buf []byte) int {
	return vlq.PutUint(buf, uint64(i))
}

func (p *BaseParser) PackVaruint64(i uint64, buf []byte) int {
	return vlq.PutUint(buf, i)
}

func (p *BaseParser) UnpackVaruint(buf []byte) (uint, int) {
	i, l := vlq.Uint(buf)
	return uint(i), l
}

func (p *BaseParser) UnpackVaruint64(buf []byte) (uint64, int) {
	return vlq.Uint(buf)
}

func (p *BaseParser) PackBigint(bi *big.Int, buf []byte) int {
	if bi == nil {
		return p.PackVaruint(0, buf)
	}
	if bi.Sign() < 0 {
		return 0
	}
	b := bi.Bytes()
	l := p.PackVaruint(uint(len(b)), buf)
	copy(buf[l:], b)
	return l + len(b)
}

func (p *BaseParser) UnpackBigint(buf []byte) (big.Int, int) {
	var bi big.Int
	l, ll := p.UnpackVaruint(buf)
	if l == 0 {
		return bi, ll
	}
	bi.SetBytes(buf[ll : ll+int(l)])
	return bi, ll + int(l)
}

func (p *BaseParser) PackVarBytes(bufValue []byte) []byte {
	buf := make([]byte, vlq.MaxLen64+len(bufValue))
	l := p.PackVaruint(uint(len(bufValue)), buf)
	buf = append(buf[:l], bufValue...)
	return buf
}

func (p *BaseParser) UnpackVarBytes(buf []byte) ([]byte, int) {
	l, ll := p.UnpackVaruint(buf)
	return append([]byte(nil), buf[ll:ll+int(l)]...), ll + int(l)
}
