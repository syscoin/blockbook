//go:build unittest

package db

import (
	"encoding/hex"
	"math/big"
	"testing"

	vlq "github.com/bsm/go-vlq"
	"github.com/martinboehm/btcutil/chaincfg"
	syscoinwire "github.com/syscoin/syscoinwire/syscoin/wire"
	"github.com/trezor/blockbook/bchain"
	"github.com/trezor/blockbook/bchain/coins/btc"
	"github.com/trezor/blockbook/bchain/coins/sys"
	"github.com/trezor/blockbook/common"
	"github.com/trezor/blockbook/tests/dbtestdata"
)

type testSyscoinParser struct {
	*syscoin.SyscoinParser
}

func syscoinTestParser() *syscoin.SyscoinParser {
	return syscoin.NewSyscoinParser(syscoin.GetChainParams("main"),
		&btc.Configuration{BlockAddressesToKeep: 2})
}

func txIndexesHexSyscoin(tx string, assetsMask bchain.AssetsMask, assetGuids []uint64, indexes []int32, d *RocksDB) string {
	buf := make([]byte, vlq.MaxLen32)
	varBuf := make([]byte, vlq.MaxLen64)
	l := packVaruint(uint(assetsMask), buf)
	tx = hex.EncodeToString(buf[:l]) + tx
	for i, index := range indexes {
		index <<= 1
		if i == len(indexes)-1 {
			index |= 1
		}
		l = packVarint32(index, buf)
		tx += hex.EncodeToString(buf[:l])
	}
	l = packVaruint(uint(len(assetGuids)), buf)
	tx += hex.EncodeToString(buf[:l])
	for _, asset := range assetGuids {
		l = packVaruint64(asset, varBuf)
		tx += hex.EncodeToString(varBuf[:l])
	}
	return tx
}

func varintToHex(i int) string {
	buf := make([]byte, vlq.MaxLen64)
	l := packVarint(i, buf)
	return hex.EncodeToString(buf[:l])
}

func loadSyscoinAssets(d *RocksDB, tx *bchain.Tx) error {
	parser, ok := d.chainParser.(interface {
		LoadAssets(*bchain.Tx) error
	})
	if !ok {
		return nil
	}
	return parser.LoadAssets(tx)
}

func syscoinAssetTestBlock(parser bchain.BlockChainParser, height uint32, hash, txid string, vin []bchain.Vin, voutAddr string, assetGuid uint64, assetValue int64) *bchain.Block {
	return &bchain.Block{
		BlockHeader: bchain.BlockHeader{
			Height: height,
			Hash:   hash,
			Size:   1,
			Time:   int64(1598557000 + height),
		},
		Txs: []bchain.Tx{
			{
				Txid:    txid,
				Version: syscoin.SYSCOIN_TX_VERSION_ALLOCATION_SEND,
				Vin:     vin,
				Vout: []bchain.Vout{
					{
						N: 0,
						ScriptPubKey: bchain.ScriptPubKey{
							Hex: dbtestdata.AddressToPubKeyHex(voutAddr, parser),
						},
						ValueSat: *big.NewInt(0),
						AssetInfo: &bchain.AssetInfo{
							AssetGuid: assetGuid,
							ValueSat:  big.NewInt(assetValue),
						},
					},
				},
				Blocktime: int64(1598557000 + height),
				Time:      int64(1598557000 + height),
			},
		},
	}
}

func verifyAfterSyscoinTypeBlock1(t *testing.T, d *RocksDB, afterDisconnect bool) {
	// Check cfHeight
	if err := checkColumn(d, cfHeight, []keyPair{
		{
			"00000070",
			"00000797cfd9074de37a557bf0d47bd86c45846f31e163ba688e14dfc498527a" + uintToHex(1598556954) + varuintToHex(1) + varuintToHex(503),
			nil,
		},
	}); err != nil {
		t.Fatal(err)
	}
	// Because we only have one coinbase TX (TxidS1T0) paying out to AddrS1
	if err := checkColumn(d, cfAddresses, []keyPair{
		{
			addressKeyHex(dbtestdata.AddrS1, 112, d),
			txIndexesHex(dbtestdata.TxidS1T0, []int32{0}),
			nil,
		},
	}); err != nil {
		t.Fatal(err)
	}
	// Check cfAddressBalance for that single output
	if err := checkColumn(d, cfAddressBalance, []keyPair{
		{
			dbtestdata.AddressToPubKeyHex(dbtestdata.AddrS1, d.chainParser),
			varuintToHex(1) + bigintToHex(dbtestdata.SatZero) + bigintToHex(dbtestdata.SatS1T0A1) +
				varuintToHex(0) + // zero assets
				dbtestdata.TxidS1T0 + varuintToHex(0) + varuintToHex(112) + bigintToHex(dbtestdata.SatS1T0A1) +
				varuintToHex(0), // no asset info
			nil,
		},
	}); err != nil {
		t.Fatal(err)
	}

	// For blockTxs, if afterDisconnect = false, we expect 1 TX in block #1
	var blockTxsKp []keyPair
	if afterDisconnect {
		blockTxsKp = []keyPair{}
	} else {
		blockTxsKp = []keyPair{
			{
				"00000070",
				dbtestdata.TxidS1T0 + "01" +
					"0000000000000000000000000000000000000000000000000000000000000000" + varintToHex(0),
				nil,
			},
		}
	}
	if err := checkColumn(d, cfBlockTxs, blockTxsKp); err != nil {
		t.Fatal(err)
	}
}

// verifyAfterSyscoinTypeBlock2 checks DB after block2 is connected
func verifyAfterSyscoinTypeBlock2(t *testing.T, d *RocksDB) {
	// CFHeight
	if err := checkColumn(d, cfHeight, []keyPair{
		{
			"00000071",
			"00000cade5f8d530b3f0a3b6c9dceaca50627838f2c6fffb807390cba71974e7" +
				uintToHex(1598557012) + varuintToHex(1) + varuintToHex(554),
			nil,
		},
		{
			"00000070",
			"00000797cfd9074de37a557bf0d47bd86c45846f31e163ba688e14dfc498527a" +
				uintToHex(1598556954) + varuintToHex(1) + varuintToHex(503),
			nil,
		},
	}); err != nil {
		t.Fatal(err)
	}

	// Only coinbase TX in block2 => output to AddrS2
	if err := checkColumn(d, cfAddresses, []keyPair{
		{
			addressKeyHex(dbtestdata.AddrS1, 112, d),
			txIndexesHex(dbtestdata.TxidS1T0, []int32{0}),
			nil,
		},
		{
			addressKeyHex(dbtestdata.AddrS2, 113, d),
			txIndexesHex(dbtestdata.TxidS2T0, []int32{0}),
			nil,
		},
	}); err != nil {
		t.Fatal(err)
	}

	// Check address balance for AddrS2
	if err := checkColumn(d, cfAddressBalance, []keyPair{
		{
			dbtestdata.AddressToPubKeyHex(dbtestdata.AddrS1, d.chainParser),
			varuintToHex(1) + bigintToHex(dbtestdata.SatZero) + bigintToHex(dbtestdata.SatS1T0A1) +
				varuintToHex(0) + // zero assets
				dbtestdata.TxidS1T0 + varuintToHex(0) + varuintToHex(112) + bigintToHex(dbtestdata.SatS1T0A1) +
				varuintToHex(0), // no asset info
			nil,
		},
		{
			dbtestdata.AddressToPubKeyHex(dbtestdata.AddrS2, d.chainParser),
			varuintToHex(1) + bigintToHex(dbtestdata.SatZero) + bigintToHex(dbtestdata.SatS2T0A1) +
				varuintToHex(0) + // zero assets
				dbtestdata.TxidS2T0 + varuintToHex(0) + varuintToHex(113) + bigintToHex(dbtestdata.SatS2T0A1) +
				varuintToHex(0), // no asset info
			nil,
		},
	}); err != nil {
		t.Fatal(err)
	}
}

// TestRocksDB_Index_SyscoinType ensures we can connect/disconnect Syscoin blocks (v5)
func TestRocksDB_Index_SyscoinType(t *testing.T) {
	d := setupRocksDB(t, &testSyscoinParser{
		SyscoinParser: syscoinTestParser(),
	})
	defer closeAndDestroyRocksDB(t, d)

	// No blocks connected yet => 0 length blockTimes
	if len(d.is.BlockTimes) != 0 {
		t.Fatalf("Expecting is.BlockTimes 0, got %d", len(d.is.BlockTimes))
	}

	// Connect block1
	block1 := dbtestdata.GetTestSyscoinTypeBlock1(d.chainParser)
	for i := range block1.Txs {
		tx := &block1.Txs[i]
		err := loadSyscoinAssets(d, tx) // no-op for coinbase
		if err != nil {
			t.Fatal(err)
		}
	}
	if err := d.ConnectBlock(block1); err != nil {
		t.Fatal(err)
	}
	verifyAfterSyscoinTypeBlock1(t, d, false)

	// Should have block times indexed up to height 112.
	if len(d.is.BlockTimes) != 113 {
		t.Fatalf("Expecting is.BlockTimes 113, got %d", len(d.is.BlockTimes))
	}

	// Connect block2
	block2 := dbtestdata.GetTestSyscoinTypeBlock2(d.chainParser)
	for i := range block2.Txs {
		tx := &block2.Txs[i]
		err := loadSyscoinAssets(d, tx) // no-op
		if err != nil {
			t.Fatal(err)
		}
	}
	if err := d.ConnectBlock(block2); err != nil {
		t.Fatal(err)
	}
	verifyAfterSyscoinTypeBlock2(t, d)

	// Should have 2 blockTimes
	if len(d.is.BlockTimes) != 114 {
		t.Fatalf("Expecting is.BlockTimes 114, got %d", len(d.is.BlockTimes))
	}

	// Test some DB queries
	// Since block1 pays to AddrS1 and block2 pays to AddrS2, let's do a getTx check
	verifyGetTransactions(t, d, dbtestdata.AddrS1, 0, 200000, []txidIndex{
		{dbtestdata.TxidS1T0, 0}, // coinbase output
	}, nil)
	verifyGetTransactions(t, d, dbtestdata.AddrS2, 0, 200000, []txidIndex{
		{dbtestdata.TxidS2T0, 0}, // coinbase output
	}, nil)

	// Check best block
	height, hash, err := d.GetBestBlock()
	if err != nil {
		t.Fatal(err)
	}
	if height != 113 {
		t.Fatalf("GetBestBlock: got height %d, expected 113", height)
	}
	if hash != "00000cade5f8d530b3f0a3b6c9dceaca50627838f2c6fffb807390cba71974e7" {
		t.Fatalf("GetBestBlock: got hash %v, expected 00000cade5f8d530b3f0a3b6c9dceaca50627838f2c6fffb807390cba71974e7", hash)
	}

	// Block1 hash
	h, err := d.GetBlockHash(112)
	if err != nil {
		t.Fatal(err)
	}
	if h != "00000797cfd9074de37a557bf0d47bd86c45846f31e163ba688e14dfc498527a" {
		t.Fatalf("Block#112 hash mismatch, got %s", h)
	}

	// Disconnect block2
	if err := d.DisconnectBlockRangeBitcoinType(113, 113); err != nil {
		t.Fatal(err)
	}
	verifyAfterSyscoinTypeBlock1(t, d, false)

	// Reconnect block2
	if err := d.ConnectBlock(block2); err != nil {
		t.Fatal(err)
	}
	verifyAfterSyscoinTypeBlock2(t, d)
	// blockTxs
	if err := checkColumn(d, cfBlockTxs, []keyPair{
		{
			"00000071",
			dbtestdata.TxidS2T0 + "01" +
				"0000000000000000000000000000000000000000000000000000000000000000" + varintToHex(0),
			nil,
		},
		{
			"00000070",
			dbtestdata.TxidS1T0 + "01" +
				"0000000000000000000000000000000000000000000000000000000000000000" + varintToHex(0),
			nil,
		},
	}); err != nil {
		t.Fatal(err)
	}
}

func TestRocksDB_DisconnectSyscoinAssetRestoresSpentUTXO(t *testing.T) {
	d := setupRocksDB(t, &testSyscoinParser{
		SyscoinParser: syscoinTestParser(),
	})
	defer closeAndDestroyRocksDB(t, d)

	const assetGuid uint64 = 12345
	const assetValue int64 = 25
	AssetCache = map[uint64]bchain.Asset{
		assetGuid: {},
	}

	txid1 := "1111111111111111111111111111111111111111111111111111111111111111"
	txid2 := "2222222222222222222222222222222222222222222222222222222222222222"
	block1 := syscoinAssetTestBlock(
		d.chainParser,
		1,
		"0000000000000000000000000000000000000000000000000000000000000001",
		txid1,
		[]bchain.Vin{{Coinbase: "01"}},
		dbtestdata.AddrS1,
		assetGuid,
		assetValue,
	)
	block2 := syscoinAssetTestBlock(
		d.chainParser,
		2,
		"0000000000000000000000000000000000000000000000000000000000000002",
		txid2,
		[]bchain.Vin{{Txid: txid1, Vout: 0}},
		dbtestdata.AddrS2,
		assetGuid,
		assetValue,
	)

	if err := d.ConnectBlock(block1); err != nil {
		t.Fatal(err)
	}
	if err := d.ConnectBlock(block2); err != nil {
		t.Fatal(err)
	}
	if err := d.DisconnectBlockRangeBitcoinType(2, 2); err != nil {
		t.Fatal(err)
	}

	addrDesc1, err := d.chainParser.GetAddrDescFromAddress(dbtestdata.AddrS1)
	if err != nil {
		t.Fatal(err)
	}
	balance1, err := d.GetAddrDescBalance(addrDesc1, addressBalanceDetailUTXOIndexed)
	if err != nil {
		t.Fatal(err)
	}
	if balance1 == nil {
		t.Fatal("missing AddrS1 balance after disconnect")
	}
	assetBalance1 := balance1.AssetBalances[assetGuid]
	if assetBalance1 == nil {
		t.Fatal("missing restored AddrS1 asset balance after disconnect")
	}
	if assetBalance1.BalanceSat.Cmp(big.NewInt(assetValue)) != 0 {
		t.Fatalf("AddrS1 asset balance = %s, want %d", assetBalance1.BalanceSat, assetValue)
	}
	if len(balance1.Utxos) != 1 || balance1.Utxos[0].AssetInfo == nil {
		t.Fatalf("restored UTXO missing asset info: %+v", balance1.Utxos)
	}
	if balance1.Utxos[0].AssetInfo.AssetGuid != assetGuid || balance1.Utxos[0].AssetInfo.ValueSat.Cmp(big.NewInt(assetValue)) != 0 {
		t.Fatalf("restored UTXO asset info = %+v, want asset %d value %d", balance1.Utxos[0].AssetInfo, assetGuid, assetValue)
	}

	addrDesc2, err := d.chainParser.GetAddrDescFromAddress(dbtestdata.AddrS2)
	if err != nil {
		t.Fatal(err)
	}
	balance2, err := d.GetAddrDescBalance(addrDesc2, addressBalanceDetailUTXOIndexed)
	if err != nil {
		t.Fatal(err)
	}
	if balance2 != nil && balance2.AssetBalances[assetGuid] != nil && balance2.AssetBalances[assetGuid].BalanceSat.Sign() != 0 {
		t.Fatalf("AddrS2 asset balance survived disconnected block: %+v", balance2.AssetBalances[assetGuid])
	}
}

func TestRocksDB_GetAssetUsesBuiltinSYSXFallback(t *testing.T) {
	d := setupRocksDB(t, &testSyscoinParser{
		SyscoinParser: syscoinTestParser(),
	})
	defer closeAndDestroyRocksDB(t, d)

	AssetCache = nil
	asset, err := d.GetAsset(syscoinSYSXAssetGuid, nil)
	if err != nil {
		t.Fatal(err)
	}
	if string(asset.AssetObj.Symbol) != "SYSX" {
		t.Fatalf("SYSX symbol = %q", asset.AssetObj.Symbol)
	}
	if asset.AssetObj.Precision != 8 {
		t.Fatalf("SYSX precision = %d, want 8", asset.AssetObj.Precision)
	}
	if len(asset.AssetObj.Contract) != 0 {
		t.Fatalf("SYSX contract length = %d, want empty native contract", len(asset.AssetObj.Contract))
	}
	if string(asset.MetaData) != "Syscoin Native Asset" {
		t.Fatalf("SYSX metadata = %q", asset.MetaData)
	}
	if cached, ok := AssetCache[syscoinSYSXAssetGuid]; !ok || string(cached.AssetObj.Symbol) != "SYSX" {
		t.Fatalf("SYSX asset not cached correctly: ok=%v asset=%+v", ok, cached)
	}

	AssetCache = map[uint64]bchain.Asset{
		syscoinSYSXAssetGuid: {
			Transactions: 150,
			AssetObj:     syscoinwire.AssetType{Symbol: []byte("SYSX"), Precision: 8},
			MetaData:     []byte("SYSX"),
		},
	}
	asset, err = d.GetAsset(syscoinSYSXAssetGuid, nil)
	if err != nil {
		t.Fatal(err)
	}
	if asset.Transactions != 150 {
		t.Fatalf("SYSX transactions = %d, want preserved count 150", asset.Transactions)
	}
	if string(asset.MetaData) != "Syscoin Native Asset" || len(asset.AssetObj.Contract) != 0 {
		t.Fatalf("SYSX cached metadata was not canonicalized: %+v", asset)
	}
}

// Test_BulkConnect_SyscoinType verifies that we can bulk-connect two Syscoin blocks
// (containing simple coinbase transactions) without any asset creation/updates.
func Test_BulkConnect_SyscoinType(t *testing.T) {
	d := setupRocksDB(t, &testSyscoinParser{
		SyscoinParser: syscoinTestParser(),
	})
	defer closeAndDestroyRocksDB(t, d)

	// The DB should be in an inconsistent state until BulkConnect is finished
	bc, err := d.InitBulkConnect()
	if err != nil {
		t.Fatal(err)
	}
	if d.is.DbState != common.DbStateInconsistent {
		t.Fatalf("Expected DbStateInconsistent, got %v", d.is.DbState)
	}

	// Nothing connected => blockTimes should be empty
	if len(d.is.BlockTimes) != 0 {
		t.Fatalf("Expecting is.BlockTimes=0 initially, got %d", len(d.is.BlockTimes))
	}

	// Prepare block1
	block1 := dbtestdata.GetTestSyscoinTypeBlock1(d.chainParser)
	for i := range block1.Txs {
		tx := &block1.Txs[i]
		// LoadAssets might do nothing here for coinbase, but we call it to keep the flow consistent
		if err := loadSyscoinAssets(d, tx); err != nil {
			t.Fatal(err)
		}
	}
	// Connect block1 in bulk mode without flushing
	if err := bc.ConnectBlock(block1, false); err != nil {
		t.Fatal(err)
	}

	// Prepare block2
	block2 := dbtestdata.GetTestSyscoinTypeBlock2(d.chainParser)
	for i := range block2.Txs {
		tx := &block2.Txs[i]
		if err := loadSyscoinAssets(d, tx); err != nil {
			t.Fatal(err)
		}
	}
	// Connect block2 with flush
	if err := bc.ConnectBlock(block2, true); err != nil {
		t.Fatal(err)
	}

	// Close the bulk connection => data is fully committed
	if err := bc.Close(); err != nil {
		t.Fatal(err)
	}

	// Now DB state is expected to be open
	if d.is.DbState != common.DbStateOpen {
		t.Fatalf("Expected DbStateOpen after bulk connect, got %v", d.is.DbState)
	}

	// Validate final DB state. This reuses the same verification method from the single-block test.
	// i.e. block2 is connected => we expect final state from block2
	verifyAfterSyscoinTypeBlock2(t, d)
	if err := checkColumn(d, cfBlockTxs, []keyPair{
		{
			"00000071",
			dbtestdata.TxidS2T0 + "01" + "0000000000000000000000000000000000000000000000000000000000000000" + varintToHex(0),
			nil,
		},
	}); err != nil {
		{
			t.Fatal(err)
		}
	}
	// Check that blockTimes was populated for all blocks from 0..113 inclusive => length 114
	// (The code increments blockTimes even for empty initial heights.)
	if len(d.is.BlockTimes) != 114 {
		t.Fatalf("Expecting is.BlockTimes=114, got %d", len(d.is.BlockTimes))
	}

	// Reset chaincfg if needed (depends on your test environment)
	chaincfg.ResetParams()
}
