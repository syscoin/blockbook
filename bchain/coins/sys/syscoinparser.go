package syscoin

import (
	"encoding/json"
	"bytes"
	"math/big"
	"github.com/martinboehm/btcd/wire"
	"github.com/martinboehm/btcutil/chaincfg"
	"github.com/martinboehm/btcutil/txscript"
	vlq "github.com/bsm/go-vlq"
	"github.com/juju/errors"
	"github.com/martinboehm/btcutil"
	
	"github.com/syscoin/blockbook/bchain"
	"github.com/syscoin/blockbook/bchain/coins/btc"
	"github.com/syscoin/blockbook/bchain/coins/utils"
)

// magic numbers
const (
	MainnetMagic wire.BitcoinNet = 0xffcae2ce
	TestnetMagic wire.BitcoinNet = 0xcee2cafe

	SYSCOIN_TX_VERSION_ALLOCATION_BURN_TO_SYSCOIN int32 = 128
	SYSCOIN_TX_VERSION_SYSCOIN_BURN_TO_ALLOCATION int32 = 129
	SYSCOIN_TX_VERSION_ASSET_ACTIVATE int32 = 130
	SYSCOIN_TX_VERSION_ASSET_UPDATE int32 = 131
	SYSCOIN_TX_VERSION_ASSET_SEND int32 = 132
	SYSCOIN_TX_VERSION_ALLOCATION_MINT int32 = 133
	SYSCOIN_TX_VERSION_ALLOCATION_BURN_TO_NEVM int32 = 134
	SYSCOIN_TX_VERSION_ALLOCATION_SEND int32 = 135
	maxAddrDescLen = 10000
	maxMemoLen = 256
	fNexusBlock = 0
)
// chain parameters
var (
	MainNetParams chaincfg.Params
	TestnetParams chaincfg.Params
)

func init() {
	MainNetParams = chaincfg.MainNetParams
	MainNetParams.Net = MainnetMagic

	// Mainnet address encoding magics
	MainNetParams.PubKeyHashAddrID = []byte{63} // base58 prefix: s
	MainNetParams.ScriptHashAddrID = []byte{5} // base68 prefix: 3
	MainNetParams.Bech32HRPSegwit = "sys"

	TestnetParams = chaincfg.TestNet3Params
	TestnetParams.Net = TestnetMagic

	// Testnet address encoding magics
	TestnetParams.PubKeyHashAddrID = []byte{65} // base58 prefix: t
	TestnetParams.ScriptHashAddrID = []byte{196} // base58 prefix: 2
	TestnetParams.Bech32HRPSegwit = "tsys"
}

// SyscoinParser handle
type SyscoinParser struct {
	*btc.BitcoinLikeParser
	BaseParser *bchain.BaseParser
}

// NewSyscoinParser returns new SyscoinParser instance
func NewSyscoinParser(params *chaincfg.Params, c *btc.Configuration) *SyscoinParser {
	parser := &SyscoinParser{
		BitcoinLikeParser: btc.NewBitcoinLikeParser(params, c),
	}
	parser.BaseParser = parser.BitcoinLikeParser.BaseParser
	return parser
}

// matches max data carrier for systx
func (p *SyscoinParser) GetMaxAddrLength() int {
	return maxAddrDescLen
}

// GetChainParams returns network parameters
func GetChainParams(chain string) *chaincfg.Params {
	if !chaincfg.IsRegistered(&chaincfg.MainNetParams) {
		chaincfg.RegisterBitcoinParams()
	}
	if !chaincfg.IsRegistered(&MainNetParams) {
		err := chaincfg.Register(&MainNetParams)
		if err == nil {
			err = chaincfg.Register(&TestnetParams)
		}
		if err != nil {
			panic(err)
		}
	}

	switch chain {
	case "test":
		return &TestnetParams
	case "regtest":
		return &chaincfg.RegressionNetParams
	default:
		return &MainNetParams
	}
}

// UnpackTx unpacks transaction from protobuf byte array
func (p *SyscoinParser) UnpackTx(buf []byte) (*bchain.Tx, uint32, error) {
	tx, height, err := p.BitcoinLikeParser.UnpackTx(buf)
	if err != nil {
		return nil, 0, err
	}
	p.LoadAssets(tx)
	return tx, height, nil
}
// TxFromMsgTx converts syscoin wire Tx to bchain.Tx
func (p *SyscoinParser) TxFromMsgTx(t *wire.MsgTx, parseAddresses bool) bchain.Tx {
	tx := p.BitcoinLikeParser.TxFromMsgTx(t, parseAddresses)
	p.LoadAssets(&tx)
	return tx
}
// ParseTxFromJson parses JSON message containing transaction and returns Tx struct
func (p *SyscoinParser) ParseTxFromJson(msg json.RawMessage) (*bchain.Tx, error) {
	tx, err := p.BaseParser.ParseTxFromJson(msg)
	if err != nil {
		return nil, err
	}
	p.LoadAssets(tx)
	return tx, nil
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
func (p *SyscoinParser) GetAssetTypeFromVersion(nVersion int32) *bchain.TokenType {
	var ttype bchain.TokenType
	switch nVersion {
	case SYSCOIN_TX_VERSION_ASSET_ACTIVATE:
		ttype = bchain.SPTAssetActivateType
	case SYSCOIN_TX_VERSION_ASSET_UPDATE:
		ttype = bchain.SPTAssetUpdateType
	case SYSCOIN_TX_VERSION_ASSET_SEND:
		ttype = bchain.SPTAssetSendType
	case SYSCOIN_TX_VERSION_ALLOCATION_MINT:
		ttype = bchain.SPTAssetAllocationMintType
	case SYSCOIN_TX_VERSION_ALLOCATION_BURN_TO_NEVM:
		ttype = bchain.SPTAssetAllocationBurnToNEVMType
	case SYSCOIN_TX_VERSION_ALLOCATION_BURN_TO_SYSCOIN:
		ttype = bchain.SPTAssetAllocationBurnToSyscoinType
	case SYSCOIN_TX_VERSION_SYSCOIN_BURN_TO_ALLOCATION:
		ttype = bchain.SPTAssetSyscoinBurnToAllocationType
	case SYSCOIN_TX_VERSION_ALLOCATION_SEND:
		ttype = bchain.SPTAssetAllocationSendType
	default:
		return nil
	}
	return &ttype
}

func (p *SyscoinParser) GetAssetsMaskFromVersion(nVersion int32) bchain.AssetsMask {
	switch nVersion {
	case SYSCOIN_TX_VERSION_ASSET_ACTIVATE:
		return bchain.AssetActivateMask
	case SYSCOIN_TX_VERSION_ASSET_UPDATE:
		return bchain.AssetUpdateMask
	case SYSCOIN_TX_VERSION_ASSET_SEND:
		return bchain.AssetSendMask
	case SYSCOIN_TX_VERSION_ALLOCATION_MINT:
		return bchain.AssetAllocationMintMask
	case SYSCOIN_TX_VERSION_ALLOCATION_BURN_TO_NEVM:
		return bchain.AssetAllocationBurnToNEVMMask
	case SYSCOIN_TX_VERSION_ALLOCATION_BURN_TO_SYSCOIN:
		return bchain.AssetAllocationBurnToSyscoinMask
	case SYSCOIN_TX_VERSION_SYSCOIN_BURN_TO_ALLOCATION:
		return bchain.AssetSyscoinBurnToAllocationMask
	case SYSCOIN_TX_VERSION_ALLOCATION_SEND:
		return bchain.AssetAllocationSendMask
	default:
		return bchain.BaseCoinMask
	}
}

func (p *SyscoinParser) IsSyscoinMintTx(nVersion int32) bool {
    return nVersion == SYSCOIN_TX_VERSION_ALLOCATION_MINT
}

func (p *SyscoinParser) IsAssetTx(nVersion int32) bool {
    return nVersion == SYSCOIN_TX_VERSION_ASSET_ACTIVATE || nVersion == SYSCOIN_TX_VERSION_ASSET_UPDATE
}

// note assetsend in core is assettx but its deserialized as allocation, we just care about balances so we can do it in same code for allocations
func (p *SyscoinParser) IsAssetAllocationTx(nVersion int32) bool {
	return nVersion == SYSCOIN_TX_VERSION_ALLOCATION_BURN_TO_NEVM || nVersion == SYSCOIN_TX_VERSION_ALLOCATION_BURN_TO_SYSCOIN || nVersion == SYSCOIN_TX_VERSION_SYSCOIN_BURN_TO_ALLOCATION ||
	nVersion == SYSCOIN_TX_VERSION_ALLOCATION_SEND
}

func (p *SyscoinParser) IsAssetSendTx(nVersion int32) bool {
    return nVersion == SYSCOIN_TX_VERSION_ASSET_SEND
}

func (p *SyscoinParser) IsAssetActivateTx(nVersion int32) bool {
    return nVersion == SYSCOIN_TX_VERSION_ASSET_ACTIVATE
}

func (p *SyscoinParser) IsSyscoinTx(nVersion int32, nHeight uint32) bool {
    return nHeight > fNexusBlock && (p.IsAssetAllocationTx(nVersion) || p.IsSyscoinMintTx(nVersion))
}

	
// TryGetOPReturn tries to process OP_RETURN script and return data
func (p *SyscoinParser) TryGetOPReturn(script []byte) []byte {
	if len(script) > 1 && script[0] == txscript.OP_RETURN {
		// trying 3 variants of OP_RETURN data
		// 1) OP_RETURN <datalen> <data>
		// 2) OP_RETURN OP_PUSHDATA1 <datalen in 1 byte> <data>
		// 3) OP_RETURN OP_PUSHDATA2 <datalen in 2 bytes> <data>
		op := script[1]
		var data []byte
		if op < txscript.OP_PUSHDATA1 {
			data = script[2:]
		} else if op == txscript.OP_PUSHDATA1 {
			data = script[3:]
		} else if op == txscript.OP_PUSHDATA2 {
			data = script[4:]
		}
		return data
	}
	return nil
}

func (p *SyscoinParser) GetAllocationFromTx(tx *bchain.Tx) (*bchain.AssetAllocation, []byte, error) {
	var addrDesc bchain.AddressDescriptor
	var err error
	for _, output := range tx.Vout {
		addrDesc, err = p.GetAddrDescFromVout(&output)
		if err != nil || len(addrDesc) == 0 || len(addrDesc) > maxAddrDescLen {
			continue
		}
		if addrDesc[0] == txscript.OP_RETURN {
			break
		}
	}
	return p.GetAssetAllocationFromDesc(&addrDesc, tx.Version)
}
func (p *SyscoinParser) GetSPTDataFromDesc(addrDesc *bchain.AddressDescriptor) ([]byte, error) {
	script, err := p.GetScriptFromAddrDesc(*addrDesc)
	if err != nil {
		return nil, err
	}
	sptData := p.TryGetOPReturn(script)
	if sptData == nil {
		return nil, errors.New("OP_RETURN empty")
	}
	return sptData, nil
}



func (p *SyscoinParser) GetAssetAllocationFromDesc(addrDesc *bchain.AddressDescriptor, txVersion int32) (*bchain.AssetAllocation, []byte, error) {
	sptData, err := p.GetSPTDataFromDesc(addrDesc)
	if err != nil {
		return nil, nil, err
	}
	return p.GetAssetAllocationFromData(sptData, txVersion)
}

func (p *SyscoinParser) GetAssetAllocationFromData(sptData []byte, txVersion int32) (*bchain.AssetAllocation, []byte, error) {
	var assetAllocation bchain.AssetAllocation
	r := bytes.NewReader(sptData)
	err := assetAllocation.AssetObj.Deserialize(r)
	if err != nil {
		return nil, nil, err
	}
	var memo []byte
	if (p.IsAssetAllocationTx(txVersion) && txVersion != SYSCOIN_TX_VERSION_ALLOCATION_BURN_TO_NEVM && txVersion != SYSCOIN_TX_VERSION_ALLOCATION_BURN_TO_SYSCOIN) {
		memo = make([]byte, maxMemoLen)
		n, _ := r.Read(memo)
		memo = memo[:n]
	}
	return &assetAllocation, memo, nil
}
func (p *SyscoinParser) LoadAssets(tx *bchain.Tx) error {
    if p.IsSyscoinTx(tx.Version, tx.BlockHeight) {
        allocation, memo, err := p.GetAllocationFromTx(tx)
		if err != nil {
			return err
		}
		tx.Memo = memo
        for _, v := range allocation.AssetObj.VoutAssets {
            for _,voutAsset := range v.Values {
				// store in vout
				tx.Vout[voutAsset.N].AssetInfo = &bchain.AssetInfo{AssetGuid: v.AssetGuid, ValueSat: big.NewInt(voutAsset.ValueSat)}
            }
        }       
	}
	return nil
}

func (p *SyscoinParser) WitnessPubKeyHashFromKeyID(keyId []byte) (string, error) {
	addr, err := btcutil.NewAddressWitnessPubKeyHash(keyId, p.BitcoinLikeParser.Params)
	if err != nil {
		return "", err
	}
	return addr.EncodeAddress(), nil
}


func (p *SyscoinParser) PackAssetKey(assetGuid uint64, height uint32) []byte {
	var buf []byte
	varBuf := make([]byte, vlq.MaxLen64)
	l := p.BaseParser.PackVaruint64(assetGuid, varBuf)
	buf = append(buf, varBuf[:l]...)
	// pack height as binary complement to achieve ordering from newest to oldest block
	varBuf = p.BaseParser.PackUint(^height)
	buf = append(buf, varBuf...)
	return buf
}

func (p *SyscoinParser) UnpackAssetKey(buf []byte) (uint64, uint32) {
	assetGuid, l := p.BaseParser.UnpackVaruint64(buf)
	height := p.BaseParser.UnpackUint(buf[l:])
	// height is packed in binary complement, convert it
	return assetGuid, ^height
}

func (p *SyscoinParser) PackAssetTxIndex(txAsset *bchain.TxAsset) []byte {
	var buf []byte
	varBuf := make([]byte, vlq.MaxLen64)
	l := p.BaseParser.PackVaruint(uint(len(txAsset.Txs)), varBuf)
	buf = append(buf, varBuf[:l]...)
	for _, txAssetIndex := range txAsset.Txs {
		varBuf = p.BaseParser.PackUint(uint32(txAssetIndex.Type))
		buf = append(buf, varBuf...)
		buf = append(buf, txAssetIndex.BtxID...)
	}
	return buf
}

func (p *SyscoinParser) UnpackAssetTxIndex(buf []byte) []*bchain.TxAssetIndex {
	var txAssetIndexes []*bchain.TxAssetIndex
	len := p.BaseParser.PackedTxidLen()
	numTxIndexes, l := p.BaseParser.UnpackVaruint(buf)
	if numTxIndexes > 0 {
		txAssetIndexes = make([]*bchain.TxAssetIndex, numTxIndexes)
		for i := uint(0); i < numTxIndexes; i++ {
			var txIndex bchain.TxAssetIndex
			txIndex.Type = bchain.AssetsMask(p.BaseParser.UnpackUint(buf[l:]))
			l += 4
			txIndex.BtxID = append([]byte(nil), buf[l:l+len]...)
			l += len
			txAssetIndexes[i] = &txIndex
		}
	}
	return txAssetIndexes
}

func (p *SyscoinParser) AppendAssetInfo(assetInfo *bchain.AssetInfo, buf []byte, varBuf []byte) []byte {
	l := p.BaseParser.PackVaruint64(assetInfo.AssetGuid, varBuf)
	buf = append(buf, varBuf[:l]...)
	l = p.BaseParser.PackBigint(assetInfo.ValueSat, varBuf)
	buf = append(buf, varBuf[:l]...)
	return buf
}

func (p *SyscoinParser) UnpackAssetInfo(assetInfo *bchain.AssetInfo, buf []byte) int {
	var l int
	assetInfo.AssetGuid, l = p.BaseParser.UnpackVaruint64(buf)
	valueSat, al := p.BaseParser.UnpackBigint(buf[l:])
	assetInfo.ValueSat = &valueSat
	l += al
	return l
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
		ti := &ta.Inputs[i]
		buf = p.BitcoinLikeParser.AppendTxInput(ti, buf, varBuf)
		if ti.AssetInfo != nil {
			l = p.BaseParser.PackVaruint(1, varBuf)
			buf = append(buf, varBuf[:l]...)
			buf = p.AppendAssetInfo(ti.AssetInfo, buf, varBuf)
		} else {
			l = p.BaseParser.PackVaruint(0, varBuf)
			buf = append(buf, varBuf[:l]...)
		}
	}
	l = p.BaseParser.PackVaruint(uint(len(ta.Outputs)), varBuf)
	buf = append(buf, varBuf[:l]...)
	for i := range ta.Outputs {
		to := &ta.Outputs[i]
		buf = p.BitcoinLikeParser.AppendTxOutput(to, buf, varBuf)
		if to.AssetInfo != nil {
			l = p.BaseParser.PackVaruint(1, varBuf)
			buf = append(buf, varBuf[:l]...)
			buf = p.AppendAssetInfo(to.AssetInfo, buf, varBuf)
		} else {
			l = p.BaseParser.PackVaruint(0, varBuf)
			buf = append(buf, varBuf[:l]...)
		}
	}
	buf = append(buf, p.BaseParser.PackVarBytes(ta.Memo)...)
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
	ta.Inputs = make([]bchain.TxInput, inputs)
	for i := uint(0); i < inputs; i++ {
		ti := &ta.Inputs[i]
		l += p.BitcoinLikeParser.UnpackTxInput(ti, buf[l:])
		assetInfoFlag, ll := p.BaseParser.UnpackVaruint(buf[l:])
		l += ll
		if assetInfoFlag == 1 {
			ti.AssetInfo = &bchain.AssetInfo{}
			l += p.UnpackAssetInfo(ti.AssetInfo, buf[l:])
		}
	}
	outputs, ll := p.BaseParser.UnpackVaruint(buf[l:])
	l += ll
	ta.Outputs = make([]bchain.TxOutput, outputs)
	for i := uint(0); i < outputs; i++ {
		to := &ta.Outputs[i]
		l += p.BitcoinLikeParser.UnpackTxOutput(to, buf[l:])
		assetInfoFlag, ll := p.BaseParser.UnpackVaruint(buf[l:])
		l += ll
		if assetInfoFlag == 1 {
			to.AssetInfo = &bchain.AssetInfo{}
			l += p.UnpackAssetInfo(to.AssetInfo, buf[l:])
		}
	}
	ta.Memo, _ = p.BaseParser.UnpackVarBytes(buf[l:])
	return &ta, nil
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
		ab.AssetBalances = make(map[uint64]*bchain.AssetBalance, numAssetBalances)
		for i := uint(0); i < numAssetBalances; i++ {
			asset, ll := p.BaseParser.UnpackVaruint64(buf[l:])
			l += ll
			balancevalue, ll := p.BaseParser.UnpackBigint(buf[l:])
			l += ll
			sentvalue, ll := p.BaseParser.UnpackBigint(buf[l:])
			l += ll
			transfers, ll := p.BaseParser.UnpackVaruint(buf[l:])
			l += ll
			ab.AssetBalances[asset] = &bchain.AssetBalance{Transfers: uint32(transfers), SentSat: &sentvalue, BalanceSat: &balancevalue}
		}
	}
	if detail != bchain.AddressBalanceDetailNoUTXO {
		// estimate the size of utxos to avoid reallocation
		ab.Utxos = make([]bchain.Utxo, 0, len(buf[l:])/txidUnpackedLen+4)
		// ab.UtxosMap = make(map[string]int, cap(ab.Utxos))
		for len(buf[l:]) >= txidUnpackedLen+4 {
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
			assetInfoFlag, ll := p.BaseParser.UnpackVaruint(buf[l:])
			l += ll
			if assetInfoFlag == 1 {
				u.AssetInfo = &bchain.AssetInfo{}
				l += p.UnpackAssetInfo(u.AssetInfo, buf[l:])
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
		l = p.BaseParser.PackVaruint64(key, varBuf)
		buf = append(buf, varBuf[:l]...)
		l = p.BaseParser.PackBigint(value.BalanceSat, varBuf)
		buf = append(buf, varBuf[:l]...)
		l = p.BaseParser.PackBigint(value.SentSat, varBuf)
		buf = append(buf, varBuf[:l]...)
		l = p.BaseParser.PackVaruint(uint(value.Transfers), varBuf)
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
			if utxo.AssetInfo != nil {
				l = p.BaseParser.PackVaruint(1, varBuf)
				buf = append(buf, varBuf[:l]...)
				buf = p.AppendAssetInfo(utxo.AssetInfo, buf, varBuf)
			} else {
				l = p.BaseParser.PackVaruint(0, varBuf)
				buf = append(buf, varBuf[:l]...)
			}
		}
	}
	return buf
}

func (p *SyscoinParser) PackedTxIndexLen() int {
	return p.BaseParser.PackedTxidLen() + 1
}

func (p *SyscoinParser) UnpackTxIndexType(buf []byte) (bchain.AssetsMask, int) {
	maskUint, l := p.BaseParser.UnpackVaruint(buf)
	return bchain.AssetsMask(maskUint), l
}

func (p *SyscoinParser) UnpackTxIndexAssets(assetGuids *[]uint64, buf *[]byte) uint {
	numAssets, l := p.BaseParser.UnpackVaruint(*buf)
	*buf = (*buf)[l:]
	for k := uint(0); k < numAssets; k++ {
		assetGuidUint, l := p.BaseParser.UnpackVaruint64(*buf)
		*assetGuids = append(*assetGuids, assetGuidUint)
		*buf = (*buf)[l:]
	}
	return numAssets
}

func (p *SyscoinParser) PackTxIndexes(txi []bchain.TxIndexes) []byte {
	buf := make([]byte, 0, 34)
	bvout := make([]byte, vlq.MaxLen32)
	varBuf := make([]byte, vlq.MaxLen64)
	// store the txs in reverse order for ordering from newest to oldest
	for j := len(txi) - 1; j >= 0; j-- {
		t := &txi[j]
		l := p.BaseParser.PackVaruint(uint(t.Type), bvout)
		buf = append(buf, bvout[:l]...)
		buf = append(buf, []byte(t.BtxID)...)
		for i, index := range t.Indexes {
			index <<= 1
			if i == len(t.Indexes)-1 {
				index |= 1
			}
			l := p.BaseParser.PackVarint32(index, bvout)
			buf = append(buf, bvout[:l]...)
		}
		l = p.BaseParser.PackVaruint(uint(len(t.Assets)), bvout)
		buf = append(buf, bvout[:l]...)
		for _, asset := range t.Assets {
			l = p.BaseParser.PackVaruint64(asset, varBuf)
			buf = append(buf, varBuf[:l]...)
		}
	}
	return buf
}

func (p *SyscoinParser) PackAsset(asset *bchain.Asset) ([]byte, error) {
	buf := make([]byte, 0, 315)
	varBuf := make([]byte, 4)
	l := p.BaseParser.PackVaruint(uint(asset.Transactions), varBuf)
	buf = append(buf, varBuf[:l]...)
	buf = append(buf, p.BaseParser.PackVarBytes(asset.MetaData)...)
	var buffer bytes.Buffer
	err := asset.AssetObj.Serialize(&buffer)
	if err != nil {
		return nil, err
	}
	buf = append(buf, buffer.Bytes()...)
	return buf, nil
}

func (p *SyscoinParser) UnpackAsset(buf []byte) (*bchain.Asset, error) {
	var asset bchain.Asset
	var ll = 0
	transactions, l := p.BaseParser.UnpackVaruint(buf)
	asset.Transactions = uint32(transactions)
	asset.MetaData, ll = p.BaseParser.UnpackVarBytes(buf[l:])
	l += ll
	r := bytes.NewReader(buf[l:])
	err := asset.AssetObj.Deserialize(r)
	if err != nil {
		return nil, err
	}
	return &asset, nil
}