//go:build unittest

package db

import (
	"reflect"
	"fmt"
	"bytes"
	"testing"
	"encoding/hex"
	"encoding/base64"
	
	"github.com/martinboehm/btcutil/chaincfg"
	"github.com/juju/errors"
	vlq "github.com/bsm/go-vlq"
	"github.com/syscoin/blockbook/bchain"
	"github.com/syscoin/blockbook/common"
	"github.com/syscoin/blockbook/bchain/coins/btc"
	"github.com/syscoin/blockbook/bchain/coins/sys"
	"github.com/syscoin/blockbook/tests/dbtestdata"
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
	l := d.chainParser.PackVaruint(uint(assetsMask), buf)
	tx = hex.EncodeToString(buf[:l]) + tx
	for i, index := range indexes {
		index <<= 1
		if i == len(indexes)-1 {
			index |= 1
		}
		l = d.chainParser.PackVarint32(index, buf)
		tx += hex.EncodeToString(buf[:l])
	}
	l = d.chainParser.PackVaruint(uint(len(assetGuids)), buf)
	tx += hex.EncodeToString(buf[:l])
	for _, asset := range assetGuids {
		l = d.chainParser.PackVaruint64(asset, varBuf)
		tx += hex.EncodeToString(varBuf[:l])
	}
	return tx
} 
func verifyAfterSyscoinTypeBlock1(t *testing.T, d *RocksDB, afterDisconnect bool) {
	if err := checkColumn(d, cfHeight, []keyPair{
		{
			"00000070",
			"00000797cfd9074de37a557bf0d47bd86c45846f31e163ba688e14dfc498527a" + uintToHex(1598556954) + varuintToHex(2) + varuintToHex(503),
			nil,
		},
	}); err != nil {
		{
			t.Fatal(err)
		}
	}
	// the vout is encoded as signed varint, i.e. value * 2 for non negative values
	if err := checkColumn(d, cfAddresses, []keyPair{
		{addressKeyHex(dbtestdata.AddrS1, 112, d), txIndexesHexSyscoin(dbtestdata.TxidS1T0, bchain.BaseCoinMask, []uint64{}, []int32{0}, d), nil},
		{addressKeyHex(dbtestdata.AddrS2, 112, d), txIndexesHexSyscoin(dbtestdata.TxidS1T1, bchain.AssetActivateMask, []uint64{2529870008}, []int32{1}, d), nil},
	
	}); err != nil {
		{
			t.Fatal(err)
		}
	}
	if err := checkColumn(d, cfAddressBalance, []keyPair{
		{
			dbtestdata.AddressToPubKeyHex(dbtestdata.AddrS1, d.chainParser),
			varuintToHex(1) + bigintToHex(dbtestdata.SatZero, d) + bigintToHex(dbtestdata.SatS1T0A1, d) +
			/*assetbalances*/varuintToHex(0) +	dbtestdata.TxidS1T0 + varuintToHex(0) + varuintToHex(112) + bigintToHex(dbtestdata.SatS1T0A1, d) + /*asset info*/varuintToHex(0),
			nil,
		},
		// asset activate
		{
			dbtestdata.AddressToPubKeyHex(dbtestdata.AddrS2, d.chainParser),
			varuintToHex(1) + bigintToHex(dbtestdata.SatZero, d) + bigintToHex(dbtestdata.SatS1T1A1, d) +
			varuintToHex(1) + varuintToHex(2529870008) + bigintToHex(dbtestdata.SatZero, d) + bigintToHex(dbtestdata.SatZero, d) + varuintToHex(1) +
			dbtestdata.TxidS1T1 + varuintToHex(1) + varuintToHex(112) + bigintToHex(dbtestdata.SatS1T1A1, d) + varuintToHex(1) + varuintToHex(2529870008) + bigintToHex(dbtestdata.SatZero, d),
			nil,
		},	
	}); err != nil {
		{
			t.Fatal(err)
		}
	}

	var blockTxsKp []keyPair
	if afterDisconnect {
		blockTxsKp = []keyPair{}
	} else {
		blockTxsKp = []keyPair{
			{
				"00000070",
				dbtestdata.TxidS1T0 + "01" + "0000000000000000000000000000000000000000000000000000000000000000" + varintToHex(0) +
				dbtestdata.TxidS1T1 + "01" + dbtestdata.TxidS1T1INPUT0 + varintToHex(0),
				nil,
			},
		}
	}

	if err := checkColumn(d, cfBlockTxs, blockTxsKp); err != nil {
		{
			t.Fatal(err)
		}
	}
	dBAsset, err := d.GetAsset(2529870008, nil)
	if dBAsset == nil || err != nil {
		if dBAsset == nil {
			t.Fatal("asset not found after block 1")
		}
		t.Fatal(err)
	}
	if dBAsset.Transactions != 1 {
		t.Fatal(fmt.Sprint("Block1: Property mismatch dbAsset.Transaction: ", dBAsset.Transactions, ". Expected: 1"))
	}
	if len(dBAsset.AssetObj.Allocation.VoutAssets) != 0 {
		t.Fatal(fmt.Sprint("Block1: Property mismatch len(dBAsset.AssetObj.Allocation.VoutAssets): ", len(dBAsset.AssetObj.Allocation.VoutAssets) , ". Expected: 0"))
	}

	if string(dBAsset.AssetObj.Symbol) != base64.StdEncoding.EncodeToString([]byte("CAT")) {
		t.Fatal(fmt.Sprint("Block1: Property mismatch dBAsset.AssetObj.Symbol: ", string(dBAsset.AssetObj.Symbol) , ". Expected: " + base64.StdEncoding.EncodeToString([]byte("CAT"))))
	}
	pubdata := "{\"desc\":\"" + base64.StdEncoding.EncodeToString([]byte("publicvalue")) + "\"}"
	if !bytes.Equal(dBAsset.AssetObj.PubData, []byte(pubdata)) {
		t.Fatal(fmt.Sprint("Block1: Property mismatch dBAsset.AssetObj.PubData: ", string(dBAsset.AssetObj.PubData)  , ". Expected: " + pubdata))
	}
	if dBAsset.AssetObj.UpdateCapabilityFlags != 127 {
		t.Fatal(fmt.Sprint("Block1: Property mismatch dBAsset.AssetObj.UpdateCapabilityFlags: ", dBAsset.AssetObj.UpdateCapabilityFlags  , ". Expected: 127"))
	}
	// init | pub data | capability flags
	if dBAsset.AssetObj.UpdateFlags != 193 {
		t.Fatal(fmt.Sprint("Block1: Property mismatch dBAsset.AssetObj.UpdateFlags: ", dBAsset.AssetObj.UpdateFlags  , ". Expected: 193"))
	}
	if dBAsset.AssetObj.TotalSupply != 0 {
		t.Fatal(fmt.Sprint("Block1: Property mismatch dBAsset.AssetObj.TotalSupply: ", dBAsset.AssetObj.TotalSupply  , ". Expected: 0"))
	}
	if dBAsset.AssetObj.MaxSupply != 100000000000 {
		t.Fatal(fmt.Sprint("Block1: Property mismatch dBAsset.AssetObj.MaxSupply: ", dBAsset.AssetObj.MaxSupply  , ". Expected: 100000000000"))
	}
	if dBAsset.AssetObj.Precision != 8 {
		t.Fatal(fmt.Sprint("Block1: Property mismatch dBAsset.AssetObj.Precision: ", dBAsset.AssetObj.Precision  , ". Expected: 8"))
	}
	if len(dBAsset.AssetObj.PrevPubData) > 0 {
		t.Fatal(fmt.Sprint("Block1: Property mismatch dBAsset.AssetObj.PrevPubData: ", string(dBAsset.AssetObj.PrevPubData)  , ". Expected: ''"))
	}
	if dBAsset.AssetObj.PrevUpdateCapabilityFlags != 0 {
		t.Fatal(fmt.Sprint("Block1: Property mismatch dBAsset.AssetObj.PrevUpdateCapabilityFlags: ", dBAsset.AssetObj.PrevUpdateCapabilityFlags  , ". Expected: 0"))
	}
}
func verifyAfterSyscoinTypeBlock2(t *testing.T, d *RocksDB) {
	if err := checkColumn(d, cfHeight, []keyPair{
		{
			"00000071",
			"00000cade5f8d530b3f0a3b6c9dceaca50627838f2c6fffb807390cba71974e7" + uintToHex(1598557012) + varuintToHex(2) + varuintToHex(554),
			nil,
		},
		{
			"00000070",
			"00000797cfd9074de37a557bf0d47bd86c45846f31e163ba688e14dfc498527a" + uintToHex(1598556954) + varuintToHex(2) + varuintToHex(503),
			nil,
		},
	}); err != nil {
		{
			t.Fatal(err)
		}
	}
	if err := checkColumn(d, cfAddresses, []keyPair{
		{addressKeyHex(dbtestdata.AddrS1, 112, d), txIndexesHexSyscoin(dbtestdata.TxidS1T0, bchain.BaseCoinMask, []uint64{}, []int32{0}, d), nil},
		{addressKeyHex(dbtestdata.AddrS2, 112, d), txIndexesHexSyscoin(dbtestdata.TxidS1T1, bchain.AssetActivateMask, []uint64{2529870008}, []int32{1}, d), nil},
		{addressKeyHex(dbtestdata.AddrS2, 113, d), txIndexesHexSyscoin(dbtestdata.TxidS2T1, bchain.AssetUpdateMask, []uint64{2529870008}, []int32{^0}, d), nil},
		{addressKeyHex(dbtestdata.AddrS3, 113, d), txIndexesHexSyscoin(dbtestdata.TxidS2T0, bchain.BaseCoinMask, []uint64{}, []int32{0}, d), nil},
		{addressKeyHex(dbtestdata.AddrS4, 113, d), txIndexesHexSyscoin(dbtestdata.TxidS2T1, bchain.AssetUpdateMask, []uint64{2529870008}, []int32{1}, d), nil},
	}); err != nil {
		{
			t.Fatal(err)
		}
	}
	if err := checkColumn(d, cfAddressBalance, []keyPair{
		{
			dbtestdata.AddressToPubKeyHex(dbtestdata.AddrS1, d.chainParser),
			varuintToHex(1) + bigintToHex(dbtestdata.SatZero, d) + bigintToHex(dbtestdata.SatS1T0A1, d) +
			/*assetbalances*/varuintToHex(0) +	dbtestdata.TxidS1T0 + varuintToHex(0) + varuintToHex(112) + bigintToHex(dbtestdata.SatS1T0A1, d) + /*asset info*/varuintToHex(0), 
			nil,
		},
		{
			dbtestdata.AddressToPubKeyHex(dbtestdata.AddrS2, d.chainParser),
			varuintToHex(2) + bigintToHex(dbtestdata.SatS1T1A1, d) + bigintToHex(dbtestdata.SatZero, d) +
			varuintToHex(1) + varuintToHex(2529870008) + bigintToHex(dbtestdata.SatZero, d) + bigintToHex(dbtestdata.SatZero, d) + /* 2 transfers, one activate one spend of activate */varuintToHex(2),
			nil,
		},
		{
			dbtestdata.AddressToPubKeyHex(dbtestdata.AddrS3, d.chainParser),
			varuintToHex(1) + bigintToHex(dbtestdata.SatZero, d) + bigintToHex(dbtestdata.SatS2T0A1, d) +
			varuintToHex(0) + dbtestdata.TxidS2T0 + varuintToHex(0) + varuintToHex(113) + bigintToHex(dbtestdata.SatS2T0A1, d) + varuintToHex(0),
			nil,
		},
		// asset update. asset activate should be spent
		{
			dbtestdata.AddressToPubKeyHex(dbtestdata.AddrS4, d.chainParser),
			varuintToHex(1) + bigintToHex(dbtestdata.SatZero, d) + bigintToHex(dbtestdata.SatS2T1A1, d) +
			varuintToHex(1) + varuintToHex(2529870008) + bigintToHex(dbtestdata.SatZero, d) + bigintToHex(dbtestdata.SatZero, d) + varuintToHex(1) +
			dbtestdata.TxidS2T1 + varuintToHex(1) + varuintToHex(113) + bigintToHex(dbtestdata.SatS2T1A1, d) +  varuintToHex(1) + varuintToHex(2529870008) + bigintToHex(dbtestdata.SatZero, d),
			nil,
		},
	}); err != nil {
		{
			t.Fatal(err)
		}
	}
	dBAsset, err := d.GetAsset(2529870008, nil)
	if dBAsset == nil || err != nil {
		if dBAsset == nil {
			t.Fatal("asset not found after block 1")
		}
		t.Fatal(err)
	}
	if dBAsset.Transactions != 2 {
		t.Fatal(fmt.Sprint("Block2: Property mismatch dbAsset.Transaction: ", dBAsset.Transactions, ". Expected: 2"))
	}
	if string(dBAsset.AssetObj.Symbol) != base64.StdEncoding.EncodeToString([]byte("CAT")) {
		t.Fatal(fmt.Sprint("Block2: Property mismatch dBAsset.AssetObj.Symbol: ", string(dBAsset.AssetObj.Symbol) , ". Expected: " + base64.StdEncoding.EncodeToString([]byte("CAT"))))
	}
	pubdata := "{\"desc\":\"" + base64.StdEncoding.EncodeToString([]byte("new publicvalue")) + "\"}"
	if !bytes.Equal(dBAsset.AssetObj.PubData, []byte(pubdata)) {
		t.Fatal(fmt.Sprint("Block2: Property mismatch dBAsset.AssetObj.PubData: ", string(dBAsset.AssetObj.PubData)  , ". Expected: " + pubdata))
	}
	if dBAsset.AssetObj.UpdateCapabilityFlags != 123 {
		t.Fatal(fmt.Sprint("Block2: Property mismatch dBAsset.AssetObj.UpdateCapabilityFlags: ", dBAsset.AssetObj.UpdateCapabilityFlags  , ". Expected: 123"))
	}
	// not wire update flags but cummulative, adds contract which is 2 (193+2)
	if dBAsset.AssetObj.UpdateFlags != 195 {
		t.Fatal(fmt.Sprint("Block2: Property mismatch dBAsset.AssetObj.UpdateFlags: ", dBAsset.AssetObj.UpdateFlags  , ". Expected: 195"))
	}
	if dBAsset.AssetObj.TotalSupply != 0 {
		t.Fatal(fmt.Sprint("Block2: Property mismatch dBAsset.AssetObj.TotalSupply: ", dBAsset.AssetObj.TotalSupply  , ". Expected: 0"))
	}
	if dBAsset.AssetObj.MaxSupply != 100000000000 {
		t.Fatal(fmt.Sprint("Block2: Property mismatch dBAsset.AssetObj.MaxSupply: ", dBAsset.AssetObj.MaxSupply  , ". Expected: 100000000000"))
	}
	if hex.EncodeToString(dBAsset.AssetObj.Contract) != "2b1e58b979e4b2d72d8bca5bb4646ccc032ddbfc" {
		t.Fatal(fmt.Sprint("Block2: Property mismatch dBAsset.AssetObj.Contract: ", dBAsset.AssetObj.MaxSupply  , ". Expected: 2b1e58b979e4b2d72d8bca5bb4646ccc032ddbfc"))
	}
	// prev contract is not persisted for performance reasons, wire info will have it
	if len(dBAsset.AssetObj.PrevContract) != 0 {
		t.Fatal(fmt.Sprint("Block2: Property mismatch dBAsset.AssetObj.PrevContract: ", string(dBAsset.AssetObj.PrevContract)  , ". Expected: ''"))
	}
	if dBAsset.AssetObj.Precision != 8 {
		t.Fatal(fmt.Sprint("Block2: Property mismatch dBAsset.AssetObj.Precision: ", dBAsset.AssetObj.Precision  , ". Expected: 8"))
	}
	// prev pub data is not persisted for performance reasons, wire info will have it
	if len(dBAsset.AssetObj.PrevPubData) != 0 {
		t.Fatal(fmt.Sprint("Block2: Property mismatch dBAsset.AssetObj.PrevPubData: ", string(dBAsset.AssetObj.PrevPubData)  , ". Expected: ''"))
	}
	if dBAsset.AssetObj.PrevUpdateCapabilityFlags != 0 {
		t.Fatal(fmt.Sprint("Block2: Property mismatch dBAsset.AssetObj.PrevUpdateCapabilityFlags: ", dBAsset.AssetObj.PrevUpdateCapabilityFlags  , ". Expected: 0"))
	}
}

// TestRocksDB_Index_SyscoinType is an integration test probing the whole indexing functionality for Syscoin which is a BitcoinType chain
// It does the following:
// 1) Connect two blocks (inputs from 2nd block are spending some outputs from the 1st block)
// 2) GetTransactions for various addresses / low-high ranges
// 3) GetBestBlock, GetBlockHash
// 4) Test tx caching functionality
// 5) Disconnect the block 2 using BlockTxs column
// 6) Reconnect block 2 and check
// After each step, the content of DB is examined and any difference against expected state is regarded as failure
func TestRocksDB_Index_SyscoinType(t *testing.T) {
	d := setupRocksDB(t, &testSyscoinParser{
		SyscoinParser: syscoinTestParser(),
	})
	defer closeAndDestroyRocksDB(t, d)

	if len(d.is.BlockTimes) != 0 {
		t.Fatal("Expecting is.BlockTimes 0, got ", len(d.is.BlockTimes))
	}

	// connect 1st block - create asset
	block1 := dbtestdata.GetTestSyscoinTypeBlock1(d.chainParser)
	for i, _ := range block1.Txs {
		tx := &block1.Txs[i]
		err := d.chainParser.LoadAssets(tx)
		if err != nil {
			t.Fatal(err)
		}
	}
	if err := d.ConnectBlock(block1); err != nil {
		t.Fatal(err)
	}
	verifyAfterSyscoinTypeBlock1(t, d, false)

	if len(d.is.BlockTimes) != 1 {
		t.Fatal("Expecting is.BlockTimes 1, got ", len(d.is.BlockTimes))
	}

	// connect 2nd block - update asset
	block2 := dbtestdata.GetTestSyscoinTypeBlock2(d.chainParser)
	for i, _ := range block2.Txs {
		tx := &block2.Txs[i]
		err := d.chainParser.LoadAssets(tx)
		if err != nil {
			t.Fatal(err)
		}
	}
	if err := d.ConnectBlock(block2); err != nil {
		t.Fatal(err)
	}
	verifyAfterSyscoinTypeBlock2(t, d)

	if err := checkColumn(d, cfBlockTxs, []keyPair{
		{
			"00000071",
			dbtestdata.TxidS2T0 + "01" + "0000000000000000000000000000000000000000000000000000000000000000" + varintToHex(0) +
			dbtestdata.TxidS2T1 + "01" + dbtestdata.TxidS1T1 + varintToHex(1),
			nil,
		},
		{
			"00000070",
			dbtestdata.TxidS1T0 + "01" + "0000000000000000000000000000000000000000000000000000000000000000" + varintToHex(0) +
			dbtestdata.TxidS1T1 + "01" + dbtestdata.TxidS1T1INPUT0 + varintToHex(0),
			nil,
		},
	}); err != nil {
		{
			t.Fatal(err)
		}
	}

	if len(d.is.BlockTimes) != 2 {
		t.Fatal("Expecting is.BlockTimes 2, got ", len(d.is.BlockTimes))
	}
	

	// get transactions for various addresses / low-high ranges
	verifyGetTransactions(t, d, dbtestdata.AddrS2, 0, 1000000, []txidIndex{
		{dbtestdata.TxidS2T1, ^0},
		{dbtestdata.TxidS1T1, 1},
	}, nil)
	verifyGetTransactions(t, d, dbtestdata.AddrS2, 112, 112, []txidIndex{
		{dbtestdata.TxidS1T1, 1},
	}, nil)
	verifyGetTransactions(t, d, dbtestdata.AddrS2, 113, 1000000, []txidIndex{
		{dbtestdata.TxidS2T1, ^0},
	}, nil)
	verifyGetTransactions(t, d, dbtestdata.AddrS2, 500000, 1000000, []txidIndex{}, nil)
	verifyGetTransactions(t, d, dbtestdata.AddrS1, 0, 1000000, []txidIndex{
		{dbtestdata.TxidS1T0, 0},
	}, nil)
	verifyGetTransactions(t, d, dbtestdata.AddrS3, 0, 1000000, []txidIndex{
		{dbtestdata.TxidS2T0, 0},
	}, nil)
	verifyGetTransactions(t, d, dbtestdata.AddrS4, 0, 1000000, []txidIndex{
		{dbtestdata.TxidS2T1, 1},
	}, nil)
	verifyGetTransactions(t, d, "SgBVZhGLjqRz8ufXFwLhZvXpUMKqoduBad", 500000, 1000000, []txidIndex{}, errors.New("checksum mismatch"))

	// GetBestBlock
	height, hash, err := d.GetBestBlock()
	if err != nil {
		t.Fatal(err)
	}
	if height != 113 {
		t.Fatalf("GetBestBlock: got height %v, expected %v", height, 113)
	}
	if hash != "00000cade5f8d530b3f0a3b6c9dceaca50627838f2c6fffb807390cba71974e7" {
		t.Fatalf("GetBestBlock: got hash %v, expected %v", hash, "00000cade5f8d530b3f0a3b6c9dceaca50627838f2c6fffb807390cba71974e7")
	}

	// GetBlockHash
	hash, err = d.GetBlockHash(112)
	if err != nil {
		t.Fatal(err)
	}
	if hash != "00000797cfd9074de37a557bf0d47bd86c45846f31e163ba688e14dfc498527a" {
		t.Fatalf("GetBlockHash: got hash %v, expected %v", hash, "00000797cfd9074de37a557bf0d47bd86c45846f31e163ba688e14dfc498527a")
	}

	// Not connected block
	hash, err = d.GetBlockHash(114)
	if err != nil {
		t.Fatal(err)
	}
	if hash != "" {
		t.Fatalf("GetBlockHash: got hash '%v', expected ''", hash)
	}

	// GetBlockHash
	info, err := d.GetBlockInfo(113)
	if err != nil {
		t.Fatal(err)
	}
	iw := &bchain.DbBlockInfo{
		Hash:   "00000cade5f8d530b3f0a3b6c9dceaca50627838f2c6fffb807390cba71974e7",
		Txs:    2,
		Size:   554,
		Time:   1598557012,
		Height: 113,
	}
	if !reflect.DeepEqual(info, iw) {
		t.Errorf("GetBlockInfo() = %+v, want %+v", info, iw)
	}

	// try to disconnect both blocks
	err = d.DisconnectBlockRangeBitcoinType(112, 113)
	if err != nil {
		t.Fatal(err)
	}
	// connect blocka again and verify the state of db
	if err := d.ConnectBlock(block1); err != nil {
		t.Fatal(err)
	}
	verifyAfterSyscoinTypeBlock1(t, d, false)
	if err := d.ConnectBlock(block2); err != nil {
		t.Fatal(err)
	}
	verifyAfterSyscoinTypeBlock2(t, d)

	// disconnect the 2nd block, verify that the db contains only data from the 1st block with restored unspentTxs
	// and that the cached tx is removed
	err = d.DisconnectBlockRangeBitcoinType(113, 113)
	if err != nil {
		t.Fatal(err)
	}
	verifyAfterSyscoinTypeBlock1(t, d, false)
	if err := checkColumn(d, cfTransactions, []keyPair{}); err != nil {
		{
			t.Fatal(err)
		}
	}
	if len(d.is.BlockTimes) != 1 {
		t.Fatal("Expecting is.BlockTimes 1, got ", len(d.is.BlockTimes))
	}

	// connect block again and verify the state of db
	if err := d.ConnectBlock(block2); err != nil {
		t.Fatal(err)
	}
	verifyAfterSyscoinTypeBlock2(t, d)
	if err := checkColumn(d, cfBlockTxs, []keyPair{
		{
			"00000071",
			dbtestdata.TxidS2T0 + "01" + "0000000000000000000000000000000000000000000000000000000000000000" + varintToHex(0) +
			dbtestdata.TxidS2T1 + "01" + dbtestdata.TxidS1T1 + varintToHex(1),
			nil,
		},
		{
			"00000070",
			dbtestdata.TxidS1T0 + "01" + "0000000000000000000000000000000000000000000000000000000000000000" + varintToHex(0) +
			dbtestdata.TxidS1T1 + "01" + dbtestdata.TxidS1T1INPUT0 + varintToHex(0),
			nil,
		},
	}); err != nil {
		{
			t.Fatal(err)
		}
	}
	
	if len(d.is.BlockTimes) != 2 {
		t.Fatal("Expecting is.BlockTimes 2, got ", len(d.is.BlockTimes))
	}
	
	// test public methods for address balance and tx addresses
	ab, err := d.GetAddressBalance(dbtestdata.AddrS4, bchain.AddressBalanceDetailUTXO)
	if err != nil {
		t.Fatal(err)
	}
	abw := &bchain.AddrBalance{
		Txs:        1,
		SentSat:    *dbtestdata.SatZero,
		BalanceSat: *dbtestdata.SatS2T1A1,
		Utxos: []bchain.Utxo{
			{
				BtxID:    hexToBytes(dbtestdata.TxidS2T1),
				Vout:     1,
				Height:   113,
				ValueSat: *dbtestdata.SatS2T1A1,
				AssetInfo: &bchain.AssetInfo{AssetGuid: 2529870008, ValueSat: dbtestdata.SatZero},
			},
		},
		AssetBalances: map[uint64]*bchain.AssetBalance {
			2529870008: &bchain.AssetBalance{
				SentSat: 	dbtestdata.SatZero,
				BalanceSat: dbtestdata.SatZero,
				Transfers:	1,
			},
		},
	}
	if !reflect.DeepEqual(ab, abw) {
		t.Errorf("GetAddressBalance() = %+v, want %+v", ab, abw)
	}

	ta, err := d.GetTxAddresses(dbtestdata.TxidS2T1)
	if err != nil {
		t.Fatal(err)
	}
	// spends an asset (activate) output to another output
	taw := &bchain.TxAddresses{
		Version: 131,
		Height: 113,
		Inputs: []bchain.TxInput{
			{
				AddrDesc: addressToAddrDesc(dbtestdata.AddrS2, d.chainParser),
				ValueSat: *dbtestdata.SatS1T1A1,
				AssetInfo: &bchain.AssetInfo{AssetGuid: 2529870008, ValueSat: dbtestdata.SatZero},
			},
		},
		Outputs: []bchain.TxOutput{
			{
				AddrDesc: hexToBytes(dbtestdata.TxidS2T1OutputReturn),
				Spent:    false,
				ValueSat: *dbtestdata.SatZero,
			},
			{
				AddrDesc: addressToAddrDesc(dbtestdata.AddrS4, d.chainParser),
				Spent:    false,
				ValueSat: *dbtestdata.SatS2T1A1,
				AssetInfo: &bchain.AssetInfo{AssetGuid: 2529870008, ValueSat: dbtestdata.SatZero},
			},
		},
	}
	if !reflect.DeepEqual(ta, taw) {
		t.Errorf("GetTxAddresses() = %+v, want %+v", ta, taw)
	}
}

func Test_BulkConnect_SyscoinType(t *testing.T) {
	d := setupRocksDB(t, &testSyscoinParser{
		SyscoinParser: syscoinTestParser(),
	})
	defer closeAndDestroyRocksDB(t, d)

	bc, err := d.InitBulkConnect()
	if err != nil {
		t.Fatal(err)
	}

	if d.is.DbState != common.DbStateInconsistent {
		t.Fatal("DB not in DbStateInconsistent")
	}

	if len(d.is.BlockTimes) != 0 {
		t.Fatal("Expecting is.BlockTimes 0, got ", len(d.is.BlockTimes))
	}

	block1 := dbtestdata.GetTestSyscoinTypeBlock1(d.chainParser)
	for i, _ := range block1.Txs {
		tx := &block1.Txs[i]
		err := d.chainParser.LoadAssets(tx)
		if err != nil {
			t.Fatal(err)
		}
	}
	if err := bc.ConnectBlock(block1, false); err != nil {
		t.Fatal(err)
	}
	if err := checkColumn(d, cfBlockTxs, []keyPair{}); err != nil {
		{
			t.Fatal(err)
		}
	}

	block2 := dbtestdata.GetTestSyscoinTypeBlock2(d.chainParser)
	for i, _ := range block2.Txs {
		tx := &block2.Txs[i]
		err := d.chainParser.LoadAssets(tx)
		if err != nil {
			t.Fatal(err)
		}
	}
	if err := bc.ConnectBlock(block2, true); err != nil {
		t.Fatal(err)
	}

	if err := bc.Close(); err != nil {
		t.Fatal(err)
	}

	if d.is.DbState != common.DbStateOpen {
		t.Fatal("DB not in DbStateOpen")
	}

	verifyAfterSyscoinTypeBlock2(t, d)
	// because BlockAddressesToKeep == 1
	if err := checkColumn(d, cfBlockTxs, []keyPair{
		{
			"00000071",
			dbtestdata.TxidS2T0 + "01" + "0000000000000000000000000000000000000000000000000000000000000000" + varintToHex(0) +
			dbtestdata.TxidS2T1 + "01" + dbtestdata.TxidS1T1 + varintToHex(1),
			nil,
		},
	}); err != nil {
		{
			t.Fatal(err)
		}
	}
	if len(d.is.BlockTimes) != 114 {
		t.Fatal("Expecting is.BlockTimes 114, got ", len(d.is.BlockTimes))
	}
	chaincfg.ResetParams()
}
