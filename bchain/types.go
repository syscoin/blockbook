package bchain

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"unsafe"
	"bytes"
	"github.com/golang/glog"
)

// ChainType is type of the blockchain
type ChainType int

const (
	// ChainBitcoinType is blockchain derived from bitcoin
	ChainBitcoinType = ChainType(iota)
	// ChainEthereumType is blockchain derived from ethereum
	ChainEthereumType
)

// errors with specific meaning returned by blockchain rpc
var (
	// ErrBlockNotFound is returned when block is not found
	// either unknown hash or too high height
	// can be returned from GetBlockHash, GetBlockHeader, GetBlock
	ErrBlockNotFound = errors.New("Block not found")
	// ErrAddressMissing is returned if address is not specified
	// for example To address in ethereum can be missing in case of contract transaction
	ErrAddressMissing = errors.New("Address missing")
	// ErrTxidMissing is returned if txid is not specified
	// for example coinbase transactions in Bitcoin
	ErrTxidMissing = errors.New("Txid missing")
	// ErrTxNotFound is returned if transaction was not found
	ErrTxNotFound = errors.New("Tx not found")
)

// Outpoint is txid together with output (or input) index
type Outpoint struct {
	Txid string
	Vout int32
}

// ScriptSig contains data about input script
type ScriptSig struct {
	// Asm string `json:"asm"`
	Hex string `json:"hex"`
}

// Vin contains data about tx output
type Vin struct {
	Coinbase  string    `json:"coinbase"`
	Txid      string    `json:"txid"`
	Vout      uint32    `json:"vout"`
	ScriptSig ScriptSig `json:"scriptSig"`
	Sequence  uint32    `json:"sequence"`
	Addresses []string  `json:"addresses"`
}

// ScriptPubKey contains data about output script
type ScriptPubKey struct {
	// Asm       string   `json:"asm"`
	Hex string `json:"hex,omitempty"`
	// Type      string   `json:"type"`
	Addresses []string `json:"addresses"`
}

// Vout contains data about tx output
type Vout struct {
	ValueSat     big.Int
	JsonValue    json.Number  `json:"value"`
	N            uint32       `json:"n"`
	ScriptPubKey ScriptPubKey `json:"scriptPubKey"`
}

// Tx is blockchain transaction
// unnecessary fields are commented out to avoid overhead
type Tx struct {
	Hex         string `json:"hex"`
	Txid        string `json:"txid"`
	Version     int32  `json:"version"`
	LockTime    uint32 `json:"locktime"`
	Vin         []Vin  `json:"vin"`
	Vout        []Vout `json:"vout"`
	BlockHeight uint32 `json:"blockHeight,omitempty"`
	// BlockHash     string `json:"blockhash,omitempty"`
	Confirmations    uint32      `json:"confirmations,omitempty"`
	Time             int64       `json:"time,omitempty"`
	Blocktime        int64       `json:"blocktime,omitempty"`
	CoinSpecificData interface{} `json:"-"`
}

// Block is block header and list of transactions
type Block struct {
	BlockHeader
	Txs []Tx `json:"tx"`
}

// BlockHeader contains limited data (as needed for indexing) from backend block header
type BlockHeader struct {
	Hash          string `json:"hash"`
	Prev          string `json:"previousblockhash"`
	Next          string `json:"nextblockhash"`
	Height        uint32 `json:"height"`
	Confirmations int    `json:"confirmations"`
	Size          int    `json:"size"`
	Time          int64  `json:"time,omitempty"`
}

// BlockInfo contains extended block header data and a list of block txids
type BlockInfo struct {
	BlockHeader
	Version    json.Number `json:"version"`
	MerkleRoot string      `json:"merkleroot"`
	Nonce      json.Number `json:"nonce"`
	Bits       string      `json:"bits"`
	Difficulty json.Number `json:"difficulty"`
	Txids      []string    `json:"tx,omitempty"`
}

// MempoolEntry is used to get data about mempool entry
type MempoolEntry struct {
	Size            uint32 `json:"size"`
	FeeSat          big.Int
	Fee             json.Number `json:"fee"`
	ModifiedFeeSat  big.Int
	ModifiedFee     json.Number `json:"modifiedfee"`
	Time            uint64      `json:"time"`
	Height          uint32      `json:"height"`
	DescendantCount uint32      `json:"descendantcount"`
	DescendantSize  uint32      `json:"descendantsize"`
	DescendantFees  uint32      `json:"descendantfees"`
	AncestorCount   uint32      `json:"ancestorcount"`
	AncestorSize    uint32      `json:"ancestorsize"`
	AncestorFees    uint32      `json:"ancestorfees"`
	Depends         []string    `json:"depends"`
}

// ChainInfo is used to get information about blockchain
type ChainInfo struct {
	Chain           string  `json:"chain"`
	Blocks          int     `json:"blocks"`
	Headers         int     `json:"headers"`
	Bestblockhash   string  `json:"bestblockhash"`
	Difficulty      string  `json:"difficulty"`
	SizeOnDisk      int64   `json:"size_on_disk"`
	Version         string  `json:"version"`
	Subversion      string  `json:"subversion"`
	ProtocolVersion string  `json:"protocolversion"`
	Timeoffset      float64 `json:"timeoffset"`
	Warnings        string  `json:"warnings"`
}

// RPCError defines rpc error returned by backend
type RPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (e *RPCError) Error() string {
	return fmt.Sprintf("%d: %s", e.Code, e.Message)
}

// AddressDescriptor is an opaque type obtained by parser.GetAddrDesc* methods
type AddressDescriptor []byte

func (ad AddressDescriptor) String() string {
	return "ad:" + hex.EncodeToString(ad)
}

// EthereumType specific

// Erc20Contract contains info about ERC20 contract
type Erc20Contract struct {
	Contract string `json:"contract"`
	Name     string `json:"name"`
	Symbol   string `json:"symbol"`
	Decimals int    `json:"decimals"`
}

// Erc20Transfer contains a single ERC20 token transfer
type Erc20Transfer struct {
	Contract string
	From     string
	To       string
	Tokens   big.Int
}

// MempoolTxidEntry contains mempool txid with first seen time
type MempoolTxidEntry struct {
	Txid string
	Time uint32
}

// Utxo holds information about unspent transaction output
type Utxo struct {
	BtxID    []byte
	Vout     int32
	Height   uint32
	ValueSat big.Int
}

// AddrBalance stores number of transactions and balances of an address
type AddrBalance struct {
	Txs        uint32
	SentSat    big.Int
	BalanceSat big.Int
	Utxos      []Utxo
	utxosMap   map[string]int
	SentAssetAllocatedSat map[uint32]big.Int
	BalanceAssetAllocatedSat map[uint32]big.Int
	SentAssetUnAllocatedSat map[uint32]big.Int
	BalanceAssetUnAllocatedSat map[uint32]big.Int
}


// ReceivedSat computes received amount from total balance and sent amount
func (ab *AddrBalance) ReceivedSat() *big.Int {
	var r big.Int
	r.Add(&ab.BalanceSat, &ab.SentSat)
	return &r
}

// addUtxo
func (ab *AddrBalance) addUtxo(u *Utxo) {
	ab.Utxos = append(ab.Utxos, *u)
	l := len(ab.Utxos)
	if l >= 16 {
		if len(ab.utxosMap) == 0 {
			ab.utxosMap = make(map[string]int, 32)
			for i := 0; i < l; i++ {
				s := string(ab.Utxos[i].BtxID)
				if _, e := ab.utxosMap[s]; !e {
					ab.utxosMap[s] = i
				}
			}
		} else {
			s := string(u.BtxID)
			if _, e := ab.utxosMap[s]; !e {
				ab.utxosMap[s] = l - 1
			}
		}
	}
}

// markUtxoAsSpent finds outpoint btxID:vout in utxos and marks it as spent
// for small number of utxos the linear search is done, for larger number there is a hashmap index
// it is much faster than removing the utxo from the slice as it would cause in memory copy operations
func (ab *AddrBalance) markUtxoAsSpent(btxID []byte, vout int32) {
	if len(ab.utxosMap) == 0 {
		for i := range ab.Utxos {
			utxo := &ab.Utxos[i]
			if utxo.Vout == vout && *(*int)(unsafe.Pointer(&utxo.BtxID[0])) == *(*int)(unsafe.Pointer(&btxID[0])) && bytes.Equal(utxo.BtxID, btxID) {
				// mark utxo as spent by setting vout=-1
				utxo.Vout = -1
				return
			}
		}
	} else {
		if i, e := ab.utxosMap[string(btxID)]; e {
			l := len(ab.Utxos)
			for ; i < l; i++ {
				utxo := &ab.Utxos[i]
				if utxo.Vout == vout {
					if bytes.Equal(utxo.BtxID, btxID) {
						// mark utxo as spent by setting vout=-1
						utxo.Vout = -1
						return
					}
					break
				}
			}
		}
	}
	glog.Errorf("Utxo %s:%d not found, using in map %v", hex.EncodeToString(btxID), vout, len(ab.utxosMap) != 0)
}

// AddressBalanceDetail specifies what data are returned by GetAddressBalance
type AddressBalanceDetail int

// MempoolTxidEntries is array of MempoolTxidEntry
type MempoolTxidEntries []MempoolTxidEntry

// OnNewBlockFunc is used to send notification about a new block
type OnNewBlockFunc func(hash string, height uint32)

// OnNewTxAddrFunc is used to send notification about a new transaction/address
type OnNewTxAddrFunc func(tx *Tx, desc AddressDescriptor)

// AddrDescForOutpointFunc defines function that returns address descriptorfor given outpoint or nil if outpoint not found
type AddrDescForOutpointFunc func(outpoint Outpoint) AddressDescriptor

// Addresses index
type TxIndexes struct {
	btxID   []byte
	indexes []int32
}

// addressesMap is a map of addresses in a block
// each address contains a slice of transactions with indexes where the address appears
// slice is used instead of map so that order is defined and also search in case of few items
type AddressesMap map[string][]TxIndexes

// TxInput holds input data of the transaction in TxAddresses
type TxInput struct {
	AddrDesc AddressDescriptor
	ValueSat big.Int
}

// BlockInfo holds information about blocks kept in column height
type DbBlockInfo struct {
	Hash   string
	Time   int64
	Txs    uint32
	Size   uint32
	Height uint32 // Height is not packed!
}
// TxOutput holds output data of the transaction in TxAddresses
type TxOutput struct {
	AddrDesc AddressDescriptor
	Spent    bool
	ValueSat big.Int
}

// TxAddresses stores transaction inputs and outputs with amounts
type TxAddresses struct {
	Version int32
	Height  uint32
	Inputs  []TxInput
	Outputs []TxOutput
}

type DbOutpoint struct {
	btxID []byte
	index int32
}

type blockTxs struct {
	btxID  []byte
	inputs []DbOutpoint
}

// BlockChain defines common interface to block chain daemon
type BlockChain interface {
	// life-cycle methods
	// initialize the block chain connector
	Initialize() error
	// create mempool but do not initialize it
	CreateMempool(BlockChain) (Mempool, error)
	// initialize mempool, create ZeroMQ (or other) subscription
	InitializeMempool(AddrDescForOutpointFunc, OnNewTxAddrFunc) error
	// shutdown mempool, ZeroMQ and block chain connections
	Shutdown(ctx context.Context) error
	// chain info
	IsTestnet() bool
	GetNetworkName() string
	GetSubversion() string
	GetCoinName() string
	GetChainInfo() (*ChainInfo, error)
	// requests
	GetBestBlockHash() (string, error)
	GetBestBlockHeight() (uint32, error)
	GetBlockHash(height uint32) (string, error)
	GetBlockHeader(hash string) (*BlockHeader, error)
	GetBlock(hash string, height uint32) (*Block, error)
	GetBlockInfo(hash string) (*BlockInfo, error)
	GetMempoolTransactions() ([]string, error)
	GetTransaction(txid string) (*Tx, error)
	GetTransactionForMempool(txid string) (*Tx, error)
	GetTransactionSpecific(tx *Tx) (json.RawMessage, error)
	EstimateSmartFee(blocks int, conservative bool) (big.Int, error)
	EstimateFee(blocks int) (big.Int, error)
	SendRawTransaction(tx string) (string, error)
	GetMempoolEntry(txid string) (*MempoolEntry, error)
	// parser
	GetChainParser() BlockChainParser
	// EthereumType specific
	EthereumTypeGetBalance(addrDesc AddressDescriptor) (*big.Int, error)
	EthereumTypeGetNonce(addrDesc AddressDescriptor) (uint64, error)
	EthereumTypeEstimateGas(params map[string]interface{}) (uint64, error)
	EthereumTypeGetErc20ContractInfo(contractDesc AddressDescriptor) (*Erc20Contract, error)
	EthereumTypeGetErc20ContractBalance(addrDesc, contractDesc AddressDescriptor) (*big.Int, error)
}

// BlockChainParser defines common interface to parsing and conversions of block chain data
type BlockChainParser interface {
	// type of the blockchain
	GetChainType() ChainType
	// KeepBlockAddresses returns number of blocks which are to be kept in blockTxs column
	// to be used for rollbacks
	KeepBlockAddresses() int
	// AmountDecimals returns number of decimal places in coin amounts
	AmountDecimals() int
	// MinimumCoinbaseConfirmations returns minimum number of confirmations a coinbase transaction must have before it can be spent
	MinimumCoinbaseConfirmations() int
	// AmountToDecimalString converts amount in big.Int to string with decimal point in the correct place
	AmountToDecimalString(a *big.Int) string
	// AmountToBigInt converts amount in json.Number (string) to big.Int
	// it uses string operations to avoid problems with rounding
	AmountToBigInt(n json.Number) (big.Int, error)
	// get max script length, in bitcoin base derivatives its 1024 
	// but for example in syscoin this is going to be 8000 for max opreturn output script for syscoin coloured tx
	GetMaxAddrLength() int
	// address descriptor conversions
	GetAddrDescFromVout(output *Vout) (AddressDescriptor, error)
	GetAddrDescFromAddress(address string) (AddressDescriptor, error)
	GetAddressesFromAddrDesc(addrDesc AddressDescriptor) ([]string, bool, error)
	GetScriptFromAddrDesc(addrDesc AddressDescriptor) ([]byte, error)
	IsAddrDescIndexable(addrDesc AddressDescriptor) bool
	// parsing/packing/unpacking specific to chain
	PackedTxidLen() int
	PackTxid(txid string) ([]byte, error)
	UnpackTxid(buf []byte) (string, error)
	ParseTx(b []byte) (*Tx, error)
	ParseTxFromJson(json.RawMessage) (*Tx, error)
	PackTx(tx *Tx, height uint32, blockTime int64) ([]byte, error)
	UnpackTx(buf []byte) (*Tx, uint32, error)
	GetAddrDescForUnknownInput(tx *Tx, input int) AddressDescriptor
	packAddrBalance(ab *AddrBalance, buf, varBuf []byte) []byte
	unpackAddrBalance(buf []byte, txidUnpackedLen int, detail AddressBalanceDetail) (*AddrBalance, error)
	packAddressKey(addrDesc AddressDescriptor, height uint32) []byte
	unpackAddressKey(key []byte) ([]byte, uint32, error)
	packTxAddresses(ta *TxAddresses, buf []byte, varBuf []byte) []byte
	appendTxInput(txi *TxInput, buf []byte, varBuf []byte) []byte
	appendTxOutput(txo *TxOutput, buf []byte, varBuf []byte) []byte
	unpackTxAddresses(buf []byte) (*TxAddresses, error)
	unpackTxInput(ti *TxInput, buf []byte) int
	unpackTxOutput(to *TxOutput, buf []byte) int
	packTxIndexes(txi []TxIndexes) []byte
	packOutpoints(outpoints []outpoint) []byte
	unpackNOutpoints(buf []byte) ([]outpoint, int, error)
	packBlockInfo(block *DbBlockInfo) ([]byte, error)
	unpackBlockInfo(buf []byte) (*DbBlockInfo, error)
	// packing/unpacking generic to all chain (expect this to be in baseparser)
	packUint(i uint32) []byte
	unpackUint(buf []byte) uint32
	packVarint32(i int32, buf []byte) int
	packVarint(i int, buf []byte) int
	packVaruint(i uint, buf []byte) int
	unpackVarint32(buf []byte) (int32, int)
	unpackVarint(buf []byte) (int, int)
	unpackVaruint(buf []byte) (uint, int)
	packBigint(bi *big.Int, buf []byte) int
	unpackBigint(buf []byte) (big.Int, int)


	// blocks
	PackBlockHash(hash string) ([]byte, error)
	UnpackBlockHash(buf []byte) (string, error)
	ParseBlock(b []byte) (*Block, error)
	// xpub
	DerivationBasePath(xpub string) (string, error)
	DeriveAddressDescriptors(xpub string, change uint32, indexes []uint32) ([]AddressDescriptor, error)
	DeriveAddressDescriptorsFromTo(xpub string, change uint32, fromIndex uint32, toIndex uint32) ([]AddressDescriptor, error)
	// EthereumType specific
	EthereumTypeGetErc20FromTx(tx *Tx) ([]Erc20Transfer, error)
	// SyscoinType specific
	IsSyscoinTx(nVersion int32) bool
	IsSyscoinAssetSend(nVersion int32) bool
	IsSyscoinMintTx(nVersion int32) bool
	IsAssetTx(nVersion int32) bool
	IsAssetAllocationTx(nVersion int32) bool
	TryGetOPReturn(script []byte) []byte
}

// Mempool defines common interface to mempool
type Mempool interface {
	Resync() (int, error)
	GetTransactions(address string) ([]Outpoint, error)
	GetAddrDescTransactions(addrDesc AddressDescriptor) ([]Outpoint, error)
	GetAllEntries() MempoolTxidEntries
	GetTransactionTime(txid string) uint32
}