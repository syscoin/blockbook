// +build unittest

package db

import (
	"blockbook/bchain"
	"blockbook/common"
	"blockbook/bchain/coins/btc"
	"blockbook/tests/dbtestdata"
	"encoding/hex"
	"reflect"
	"testing"

	"github.com/juju/errors"
)

type testSyscoinParser struct {
	*btc.BitcoinParser
}

func syscoinTestnetParser() *btc.BitcoinParser {
	return btc.NewBitcoinParser(btc.GetChainParams("test"),
	&btc.Configuration{BlockAddressesToKeep: 1})
}

func verifyAfterSyscoinTypeBlock1(t *testing.T, d *RocksDB, afterDisconnect bool) {
	if err := checkColumn(d, cfHeight, []keyPair{
		{
			"000370d5",
			"0000000076fbbed90fd75b0e18856aa35baa984e9c9d444cf746ad85e94e2997" + uintToHex(1521515026) + varuintToHex(2) + varuintToHex(1234567),
			nil,
		},
	}); err != nil {
		{
			t.Fatal(err)
		}
	}
	// the vout is encoded as signed varint, i.e. value * 2 for non negative values
	if err := checkColumn(d, cfAddresses, []keyPair{
		{addressKeyHex(dbtestdata.AddrS1, 225493, d), txIndexesHex(dbtestdata.TxidS1T1, []int32{0}, d), nil},
		{addressKeyHex(dbtestdata.AddrS2, 225493, d), txIndexesHex(dbtestdata.TxidS1T1, []int32{1, 2}, d), nil},
		{addressKeyHex(dbtestdata.AddrS3, 225493, d), txIndexesHex(dbtestdata.TxidS1T2, []int32{0}, d), nil},
		{addressKeyHex(dbtestdata.AddrS4, 225493, d), txIndexesHex(dbtestdata.TxidS1T2, []int32{1}, d), nil},
		{addressKeyHex(dbtestdata.AddrS5, 225493, d), txIndexesHex(dbtestdata.TxidS1T2, []int32{2}, d), nil},
	}); err != nil {
		{
			t.Fatal(err)
		}
	}
	if err := checkColumn(d, cfTxAddresses, []keyPair{
		{
			dbtestdata.TxidS1T1,
			varuintToHex(225493) +
				"00" +
				"03" +
				addressToPubKeyHexWithLength(dbtestdata.AddrS1, t, d) + bigintToHex(dbtestdata.SatS1T1A1, d) +
				addressToPubKeyHexWithLength(dbtestdata.AddrS2, t, d) + bigintToHex(dbtestdata.SatS1T1A2, d),
			nil,
		},
		{
			dbtestdata.TxidS1T2,
			varuintToHex(225493) +
				"00" +
				"03" +
				addressToPubKeyHexWithLength(dbtestdata.AddrS3, t, d) + bigintToHex(dbtestdata.SatS1T2A3, d) +
				addressToPubKeyHexWithLength(dbtestdata.AddrS4, t, d) + bigintToHex(dbtestdata.SatS1T2A4, d) +
				addressToPubKeyHexWithLength(dbtestdata.AddrS5, t, d) + bigintToHex(dbtestdata.SatS1T2A5, d),
			nil,
		},
	}); err != nil {
		{
			t.Fatal(err)
		}
	}
	if err := checkColumn(d, cfAddressBalance, []keyPair{
		{
			dbtestdata.AddressToPubKeyHex(dbtestdata.AddrS1, d.chainParser),
			"01" + bigintToHex(dbtestdata.SatZero, d) + bigintToHex(dbtestdata.SatS1T1A1, d) +
				dbtestdata.TxidS1T1 + varuintToHex(0) + varuintToHex(225493) + bigintToHex(dbtestdata.SatS1T1A1, d),
			nil,
		},
		{
			dbtestdata.AddressToPubKeyHex(dbtestdata.AddrS2, d.chainParser),
			"01" + bigintToHex(dbtestdata.SatZero, d) + bigintToHex(dbtestdata.SatS1T1A2Double, d) +
			dbtestdata.TxidS1T1 + varuintToHex(1) + varuintToHex(225493) + bigintToHex(dbtestdata.SatS1T1A2, d) +
			dbtestdata.TxidS1T1 + varuintToHex(2) + varuintToHex(225493) + bigintToHex(dbtestdata.SatS1T1A2, d),
			nil,
		},
		{
			dbtestdata.AddressToPubKeyHex(dbtestdata.AddrS3, d.chainParser),
			"01" + bigintToHex(dbtestdata.SatZero, d) + bigintToHex(dbtestdata.SatS1T2A3, d) +
				dbtestdata.TxidS1T2 + varuintToHex(0) + varuintToHex(225493) + bigintToHex(dbtestdata.SatS1T2A3, d),
			nil,
		},
		{
			dbtestdata.AddressToPubKeyHex(dbtestdata.AddrS4, d.chainParser),
			"01" + bigintToHex(dbtestdata.SatZero, d) + bigintToHex(dbtestdata.SatS1T2A4, d) +
				dbtestdata.TxidS1T2 + varuintToHex(1) + varuintToHex(225493) + bigintToHex(dbtestdata.SatS1T2A4, d),
			nil,
		},
		{
			dbtestdata.AddressToPubKeyHex(dbtestdata.AddrS5, d.chainParser),
			"01" + bigintToHex(dbtestdata.SatZero, d) + bigintToHex(dbtestdata.SatS1T2A5, d) +
				dbtestdata.TxidS1T2 + varuintToHex(2) + varuintToHex(225493) + bigintToHex(dbtestdata.SatS1T2A5, d),
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
				"000370d5",
				dbtestdata.TxidS1T1 + "00" + dbtestdata.TxidS1T2 + "00",
				nil,
			},
		}
	}

	if err := checkColumn(d, cfBlockTxs, blockTxsKp); err != nil {
		{
			t.Fatal(err)
		}
	}
}

func verifyAfterSyscoinTypeBlock2(t *testing.T, d *RocksDB) {
	if err := checkColumn(d, cfHeight, []keyPair{
		{
			"000370d5",
			"0000000076fbbed90fd75b0e18856aa35baa984e9c9d444cf746ad85e94e2997" + uintToHex(1521515026) + varuintToHex(2) + varuintToHex(1234567),
			nil,
		},
		{
			"000370d6",
			"00000000eb0443fd7dc4a1ed5c686a8e995057805f9a161d9a5a77a95e72b7b6" + uintToHex(1521595678) + varuintToHex(4) + varuintToHex(2345678),
			nil,
		},
	}); err != nil {
		{
			t.Fatal(err)
		}
	}
	if err := checkColumn(d, cfAddresses, []keyPair{
		{addressKeyHex(dbtestdata.AddrS1, 225493, d), txIndexesHex(dbtestdata.TxidS1T1, []int32{0}, d), nil},
		{addressKeyHex(dbtestdata.AddrS2, 225493, d), txIndexesHex(dbtestdata.TxidS1T1, []int32{1, 2}, d), nil},
		{addressKeyHex(dbtestdata.AddrS3, 225493, d), txIndexesHex(dbtestdata.TxidS1T2, []int32{0}, d), nil},
		{addressKeyHex(dbtestdata.AddrS4, 225493, d), txIndexesHex(dbtestdata.TxidS1T2, []int32{1}, d), nil},
		{addressKeyHex(dbtestdata.AddrS5, 225493, d), txIndexesHex(dbtestdata.TxidS1T2, []int32{2}, d), nil},
		{addressKeyHex(dbtestdata.AddrS6, 225494, d), txIndexesHex(dbtestdata.TxidS2T2, []int32{^0}, d) + txIndexesHex(dbtestdata.TxidS2T1, []int32{0}, d), nil},
		{addressKeyHex(dbtestdata.AddrS7, 225494, d), txIndexesHex(dbtestdata.TxidS2T1, []int32{1}, d), nil},
		{addressKeyHex(dbtestdata.AddrS8, 225494, d), txIndexesHex(dbtestdata.TxidS2T2, []int32{0}, d), nil},
		{addressKeyHex(dbtestdata.AddrS9, 225494, d), txIndexesHex(dbtestdata.TxidS2T2, []int32{1}, d), nil},
		{addressKeyHex(dbtestdata.AddrS3, 225494, d), txIndexesHex(dbtestdata.TxidS2T1, []int32{^0}, d), nil},
		{addressKeyHex(dbtestdata.AddrS2, 225494, d), txIndexesHex(dbtestdata.TxidS2T1, []int32{^1}, d), nil},
		{addressKeyHex(dbtestdata.AddrS5, 225494, d), txIndexesHex(dbtestdata.TxidS2T3, []int32{0, ^0}, d), nil},
		{addressKeyHex(dbtestdata.AddrSA, 225494, d), txIndexesHex(dbtestdata.TxidS2T4, []int32{0}, d), nil},
		{addressKeyHex(dbtestdata.AddrS4, 225494, d), txIndexesHex(dbtestdata.TxidS2T2, []int32{^1}, d), nil},
	}); err != nil {
		{
			t.Fatal(err)
		}
	}
	if err := checkColumn(d, cfTxAddresses, []keyPair{
		{
			dbtestdata.TxidS1T1,
			varuintToHex(225493) +
				"00" +
				"03" +
				addressToPubKeyHexWithLength(dbtestdata.AddrS1, t, d) + bigintToHex(dbtestdata.SatS1T1A1, d) +
				spentAddressToPubKeyHexWithLength(dbtestdata.AddrS2, t, d) + bigintToHex(dbtestdata.SatS1T1A2, d) +
				addressToPubKeyHexWithLength(dbtestdata.AddrS2, t, d) + bigintToHex(dbtestdata.SatS1T1A2, d),
			nil,
		},
		{
			dbtestdata.TxidS1T2,
			varuintToHex(225493) +
				"00" +
				"03" +
				spentAddressToPubKeyHexWithLength(dbtestdata.AddrS3, t, d) + bigintToHex(dbtestdata.SatS1T2A3, d) +
				spentAddressToPubKeyHexWithLength(dbtestdata.AddrS4, t, d) + bigintToHex(dbtestdata.SatS1T2A4, d) +
				spentAddressToPubKeyHexWithLength(dbtestdata.AddrS5, t, d) + bigintToHex(dbtestdata.SatS1T2A5, d),
			nil,
		},
		{
			dbtestdata.TxidS2T1,
			varuintToHex(225494) +
				"02" +
				inputAddressToPubKeyHexWithLength(dbtestdata.AddrS3, t, d) + bigintToHex(dbtestdata.SatS1T2A3, d) +
				inputAddressToPubKeyHexWithLength(dbtestdata.AddrS2, t, d) + bigintToHex(dbtestdata.SatS1T1A2, d) +
				"03" +
				spentAddressToPubKeyHexWithLength(dbtestdata.AddrS6, t, d) + bigintToHex(dbtestdata.SatS2T1A6, d) +
				addressToPubKeyHexWithLength(dbtestdata.AddrS7, t, d) + bigintToHex(dbtestdata.SatS2T1A7, d) +
				hex.EncodeToString([]byte{byte(len(dbtestdata.TxidS2T1Output3OpReturn))}) + dbtestdata.TxidS2T1Output3OpReturn + bigintToHex(dbtestdata.SatZero, d),
			nil,
		},
		{
			dbtestdata.TxidS2T2,
			varuintToHex(225494) +
				"02" +
				inputAddressToPubKeyHexWithLength(dbtestdata.AddrS6, t, d) + bigintToHex(dbtestdata.SatS2T1A6, d) +
				inputAddressToPubKeyHexWithLength(dbtestdata.AddrS4, t, d) + bigintToHex(dbtestdata.SatS1T2A4, d) +
				"02" +
				addressToPubKeyHexWithLength(dbtestdata.AddrS8, t, d) + bigintToHex(dbtestdata.SatS2T2A8, d) +
				addressToPubKeyHexWithLength(dbtestdata.AddrS9, t, d) + bigintToHex(dbtestdata.SatS2T2A9, d),
			nil,
		},
		{
			dbtestdata.TxidS2T3,
			varuintToHex(225494) +
				"01" +
				inputAddressToPubKeyHexWithLength(dbtestdata.AddrS5, t, d) + bigintToHex(dbtestdata.SatS1T2A5, d) +
				"01" +
				addressToPubKeyHexWithLength(dbtestdata.AddrS5, t, d) + bigintToHex(dbtestdata.SatS2T3A5, d),
			nil,
		},
		{
			dbtestdata.TxidS2T4,
			varuintToHex(225494) +
				"01" + inputAddressToPubKeyHexWithLength("", t, d) + bigintToHex(dbtestdata.SatZero, d) +
				"02" +
				addressToPubKeyHexWithLength(dbtestdata.AddrSA, t, d) + bigintToHex(dbtestdata.SatS2T4AA, d) +
				addressToPubKeyHexWithLength("", t, d) + bigintToHex(dbtestdata.SatZero, d),
			nil,
		},
	}); err != nil {
		{
			t.Fatal(err)
		}
	}
	if err := checkColumn(d, cfAddressBalance, []keyPair{
		{
			dbtestdata.AddressToPubKeyHex(dbtestdata.AddrS1, d.chainParser),
			"01" + bigintToHex(dbtestdata.SatZero, d) + bigintToHex(dbtestdata.SatS1T1A1, d) +
				dbtestdata.TxidS1T1 + varuintToHex(0) + varuintToHex(225493) + bigintToHex(dbtestdata.SatS1T1A1, d),
			nil,
		},
		{
			dbtestdata.AddressToPubKeyHex(dbtestdata.AddrS2, d.chainParser),
			"02" + bigintToHex(dbtestdata.SatS1T1A2, d) + bigintToHex(dbtestdata.SatS1T1A2, d) +
			dbtestdata.TxidS1T1 + varuintToHex(2) + varuintToHex(225493) + bigintToHex(dbtestdata.SatS1T1A2, d),
			nil,
		},
		{
			dbtestdata.AddressToPubKeyHex(dbtestdata.AddrS3, d.chainParser),
			"02" + bigintToHex(dbtestdata.SatS1T2A3, d) + bigintToHex(dbtestdata.SatZero, d),
			nil,
		},
		{
			dbtestdata.AddressToPubKeyHex(dbtestdata.AddrS4, d.chainParser),
			"02" + bigintToHex(dbtestdata.SatS1T2A4, d) + bigintToHex(dbtestdata.SatZero, d),
			nil,
		},
		{
			dbtestdata.AddressToPubKeyHex(dbtestdata.AddrS5, d.chainParser),
			"02" + bigintToHex(dbtestdata.SatS1T2A5, d) + bigintToHex(dbtestdata.SatS2T3A5, d) +
				dbtestdata.TxidS2T3 + varuintToHex(0) + varuintToHex(225494) + bigintToHex(dbtestdata.SatS2T3A5, d),
			nil,
		},
		{
			dbtestdata.AddressToPubKeyHex(dbtestdata.AddrS6, d.chainParser),
			"02" + bigintToHex(dbtestdata.SatS2T1A6, d) + bigintToHex(dbtestdata.SatZero, d),
			nil,
		},
		{
			dbtestdata.AddressToPubKeyHex(dbtestdata.AddrS7, d.chainParser),
			"01" + bigintToHex(dbtestdata.SatZero, d) + bigintToHex(dbtestdata.SatS2T1A7, d) +
				dbtestdata.TxidS2T1 + varuintToHex(1) + varuintToHex(225494) + bigintToHex(dbtestdata.SatS2T1A7, d),
			nil,
		},
		{
			dbtestdata.AddressToPubKeyHex(dbtestdata.AddrS8, d.chainParser),
			"01" + bigintToHex(dbtestdata.SatZero, d) + bigintToHex(dbtestdata.SatS2T2A8, d) +
				dbtestdata.TxidS2T2 + varuintToHex(0) + varuintToHex(225494) + bigintToHex(dbtestdata.SatS2T2A8, d),
			nil,
		},
		{
			dbtestdata.AddressToPubKeyHex(dbtestdata.AddrS9, d.chainParser),
			"01" + bigintToHex(dbtestdata.SatZero, d) + bigintToHex(dbtestdata.SatS2T2A9, d) +
				dbtestdata.TxidS2T2 + varuintToHex(1) + varuintToHex(225494) + bigintToHex(dbtestdata.SatS2T2A9, d),
			nil,
		},
		{
			dbtestdata.AddressToPubKeyHex(dbtestdata.AddrSA, d.chainParser),
			"01" + bigintToHex(dbtestdata.SatZero, d) + bigintToHex(dbtestdata.SatS2T4AA, d) +
				dbtestdata.TxidS2T4 + varuintToHex(0) + varuintToHex(225494) + bigintToHex(dbtestdata.SatS2T4AA, d),
			nil,
		},
	}); err != nil {
		{
			t.Fatal(err)
		}
	}
	if err := checkColumn(d, cfBlockTxs, []keyPair{
		{
			"000370d6",
			dbtestdata.TxidS2T1 + "02" + dbtestdata.TxidS1T2 + "00" + dbtestdata.TxidS1T1 + "02" +
				dbtestdata.TxidS2T2 + "02" + dbtestdata.TxidS2T1 + "00" + dbtestdata.TxidS1T2 + "02" +
				dbtestdata.TxidS2T3 + "01" + dbtestdata.TxidS1T2 + "04" +
				dbtestdata.TxidS2T4 + "01" + "0000000000000000000000000000000000000000000000000000000000000000" + "00",
			nil,
		},
	}); err != nil {
		{
			t.Fatal(err)
		}
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
		BitcoinParser: syscoinTestnetParser(),
	})
	defer closeAndDestroyRocksDB(t, d)

	if len(d.is.BlockTimes) != 0 {
		t.Fatal("Expecting is.BlockTimes 0, got ", len(d.is.BlockTimes))
	}

	// connect 1st block - will log warnings about missing UTXO transactions in txAddresses column
	block1 := dbtestdata.GetTestSyscoinTypeBlock1(d.chainParser)
	if err := d.ConnectBlock(block1); err != nil {
		t.Fatal(err)
	}
	verifyAfterSyscoinTypeBlock1(t, d, false)

	if len(d.is.BlockTimes) != 1 {
		t.Fatal("Expecting is.BlockTimes 1, got ", len(d.is.BlockTimes))
	}

	// connect 2nd block - use some outputs from the 1st block as the inputs and 1 input uses tx from the same block
	block2 := dbtestdata.GetTestSyscoinTypeBlock2(d.chainParser)
	if err := d.ConnectBlock(block2); err != nil {
		t.Fatal(err)
	}
	verifyAfterSyscoinTypeBlock2(t, d)

	if len(d.is.BlockTimes) != 2 {
		t.Fatal("Expecting is.BlockTimes 1, got ", len(d.is.BlockTimes))
	}

	// get transactions for various addresses / low-high ranges
	verifyGetTransactions(t, d, dbtestdata.AddrS2, 0, 1000000, []txidIndex{
		{dbtestdata.TxidS2T1, ^1},
		{dbtestdata.TxidS1T1, 1},
		{dbtestdata.TxidS1T1, 2},
	}, nil)
	verifyGetTransactions(t, d, dbtestdata.AddrS2, 225493, 225493, []txidIndex{
		{dbtestdata.TxidS1T1, 1},
		{dbtestdata.TxidS1T1, 2},
	}, nil)
	verifyGetTransactions(t, d, dbtestdata.AddrS2, 225494, 1000000, []txidIndex{
		{dbtestdata.TxidS2T1, ^1},
	}, nil)
	verifyGetTransactions(t, d, dbtestdata.AddrS2, 500000, 1000000, []txidIndex{}, nil)
	verifyGetTransactions(t, d, dbtestdata.AddrS8, 0, 1000000, []txidIndex{
		{dbtestdata.TxidS2T2, 0},
	}, nil)
	verifyGetTransactions(t, d, dbtestdata.AddrS6, 0, 1000000, []txidIndex{
		{dbtestdata.TxidS2T2, ^0},
		{dbtestdata.TxidS2T1, 0},
	}, nil)
	verifyGetTransactions(t, d, "mtGXQvBowMkBpnhLckhxhbwYK44Gs9eBad", 500000, 1000000, []txidIndex{}, errors.New("checksum mismatch"))

	// GetBestBlock
	height, hash, err := d.GetBestBlock()
	if err != nil {
		t.Fatal(err)
	}
	if height != 225494 {
		t.Fatalf("GetBestBlock: got height %v, expected %v", height, 225494)
	}
	if hash != "00000000eb0443fd7dc4a1ed5c686a8e995057805f9a161d9a5a77a95e72b7b6" {
		t.Fatalf("GetBestBlock: got hash %v, expected %v", hash, "00000000eb0443fd7dc4a1ed5c686a8e995057805f9a161d9a5a77a95e72b7b6")
	}

	// GetBlockHash
	hash, err = d.GetBlockHash(225493)
	if err != nil {
		t.Fatal(err)
	}
	if hash != "0000000076fbbed90fd75b0e18856aa35baa984e9c9d444cf746ad85e94e2997" {
		t.Fatalf("GetBlockHash: got hash %v, expected %v", hash, "0000000076fbbed90fd75b0e18856aa35baa984e9c9d444cf746ad85e94e2997")
	}

	// Not connected block
	hash, err = d.GetBlockHash(225495)
	if err != nil {
		t.Fatal(err)
	}
	if hash != "" {
		t.Fatalf("GetBlockHash: got hash '%v', expected ''", hash)
	}

	// GetBlockHash
	info, err := d.GetBlockInfo(225494)
	if err != nil {
		t.Fatal(err)
	}
	iw := &bchain.DbBlockInfo{
		Hash:   "00000000eb0443fd7dc4a1ed5c686a8e995057805f9a161d9a5a77a95e72b7b6",
		Txs:    4,
		Size:   2345678,
		Time:   1521595678,
		Height: 225494,
	}
	if !reflect.DeepEqual(info, iw) {
		t.Errorf("GetBlockInfo() = %+v, want %+v", info, iw)
	}

	// Test tx caching functionality, leave one tx in db to test cleanup in DisconnectBlock
	testTxCache(t, d, block1, &block1.Txs[0])
	testTxCache(t, d, block2, &block2.Txs[0])
	if err = d.PutTx(&block2.Txs[1], block2.Height, block2.Txs[1].Blocktime); err != nil {
		t.Fatal(err)
	}
	// check that there is only the last tx in the cache
	packedTx, err := d.chainParser.PackTx(&block2.Txs[1], block2.Height, block2.Txs[1].Blocktime)
	if err := checkColumn(d, cfTransactions, []keyPair{
		{block2.Txs[1].Txid, hex.EncodeToString(packedTx), nil},
	}); err != nil {
		{
			t.Fatal(err)
		}
	}

	// try to disconnect both blocks, however only the last one is kept, it is not possible
	err = d.DisconnectBlockRangeBitcoinType(225493, 225494)
	if err == nil || err.Error() != "Cannot disconnect blocks with height 225493 and lower. It is necessary to rebuild index." {
		t.Fatal(err)
	}
	verifyAfterSyscoinTypeBlock2(t, d)

	// disconnect the 2nd block, verify that the db contains only data from the 1st block with restored unspentTxs
	// and that the cached tx is removed
	err = d.DisconnectBlockRangeBitcoinType(225494, 225494)
	if err != nil {
		t.Fatal(err)
	}
	verifyAfterSyscoinTypeBlock1(t, d, true)
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

	if len(d.is.BlockTimes) != 2 {
		t.Fatal("Expecting is.BlockTimes 1, got ", len(d.is.BlockTimes))
	}

	// test public methods for address balance and tx addresses
	ab, err := d.GetAddressBalance(dbtestdata.AddrS5, bchain.AddressBalanceDetailUTXO)
	if err != nil {
		t.Fatal(err)
	}
	abw := &bchain.AddrBalance{
		Txs:        2,
		SentSat:    *dbtestdata.SatS1T2A5,
		BalanceSat: *dbtestdata.SatS2T3A5,
		Utxos: []bchain.Utxo{
			{
				BtxID:    hexToBytes(dbtestdata.TxidS2T3),
				Vout:     0,
				Height:   225494,
				ValueSat: *dbtestdata.SatS2T3A5,
			},
		},
	}
	if !reflect.DeepEqual(ab, abw) {
		t.Errorf("GetAddressBalance() = %+v, want %+v", ab, abw)
	}
	rs := ab.ReceivedSat()
	rsw := dbtestdata.SatS1T2A5.Add(dbtestdata.SatS1T2A5, dbtestdata.SatS2T3A5)
	if rs.Cmp(rsw) != 0 {
		t.Errorf("GetAddressBalance().ReceivedSat() = %v, want %v", rs, rsw)
	}

	ta, err := d.GetTxAddresses(dbtestdata.TxidS2T1)
	if err != nil {
		t.Fatal(err)
	}
	taw := &bchain.TxAddresses{
		Height: 225494,
		Inputs: []bchain.TxInput{
			{
				AddrDesc: addressToAddrDesc(dbtestdata.AddrS3, d.chainParser),
				ValueSat: *dbtestdata.SatS1T2A3,
			},
			{
				AddrDesc: addressToAddrDesc(dbtestdata.AddrS2, d.chainParser),
				ValueSat: *dbtestdata.SatS1T1A2,
			},
		},
		Outputs: []bchain.TxOutput{
			{
				AddrDesc: addressToAddrDesc(dbtestdata.AddrS6, d.chainParser),
				Spent:    true,
				ValueSat: *dbtestdata.SatS2T1A6,
			},
			{
				AddrDesc: addressToAddrDesc(dbtestdata.AddrS7, d.chainParser),
				Spent:    false,
				ValueSat: *dbtestdata.SatS2T1A7,
			},
			{
				AddrDesc: hexToBytes(dbtestdata.TxidS2T1Output3OpReturn),
				Spent:    false,
				ValueSat: *dbtestdata.SatZero,
			},
		},
	}
	if !reflect.DeepEqual(ta, taw) {
		t.Errorf("GetTxAddresses() = %+v, want %+v", ta, taw)
	}
	ia, _, err := ta.Inputs[0].Addresses(d.chainParser)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(ia, []string{dbtestdata.AddrS3}) {
		t.Errorf("GetTxAddresses().Inputs[0].Addresses() = %v, want %v", ia, []string{dbtestdata.AddrS3})
	}

}

func Test_BulkConnect_SyscoinType(t *testing.T) {
	d := setupRocksDB(t, &testSyscoinParser{
		BitcoinParser: syscoinTestnetParser(),
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

	if err := bc.ConnectBlock(dbtestdata.GetTestSyscoinTypeBlock1(d.chainParser), false); err != nil {
		t.Fatal(err)
	}
	if err := checkColumn(d, cfBlockTxs, []keyPair{}); err != nil {
		{
			t.Fatal(err)
		}
	}

	if err := bc.ConnectBlock(dbtestdata.GetTestSyscoinTypeBlock2(d.chainParser), true); err != nil {
		t.Fatal(err)
	}

	if err := bc.Close(); err != nil {
		t.Fatal(err)
	}

	if d.is.DbState != common.DbStateOpen {
		t.Fatal("DB not in DbStateOpen")
	}

	verifyAfterSyscoinTypeBlock2(t, d)

	if len(d.is.BlockTimes) != 225495 {
		t.Fatal("Expecting is.BlockTimes 225495, got ", len(d.is.BlockTimes))
	}
}
