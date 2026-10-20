//go:build unittest

package db

import (
	"math/big"
	"strings"
	"testing"

	"github.com/linxGnu/grocksdb"
	"github.com/stretchr/testify/require"
	"github.com/trezor/blockbook/bchain"
	"github.com/trezor/blockbook/tests/dbtestdata"
)

func TestGetTxAddressesOutputSyscoin(t *testing.T) {
	for _, extendedIndex := range []bool{false, true} {
		name := "basic"
		if extendedIndex {
			name = "extendedIndex"
		}
		t.Run(name, func(t *testing.T) {
			d := setupRocksDBWithExtendedIndex(t, &testSyscoinParser{SyscoinParser: syscoinTestParser()}, extendedIndex)
			defer closeAndDestroyRocksDB(t, d)

			asset := &bchain.AssetInfo{AssetGuid: 1<<40 + 12345, ValueSat: new(big.Int).Lsh(big.NewInt(1), 120)}
			zeroAsset := &bchain.AssetInfo{AssetGuid: 54321, ValueSat: big.NewInt(0)}
			ta := &TxAddresses{
				Version: 132,
				Height:  12345,
				VSize:   512,
				Memo:    []byte("Syscoin memo after the outputs"),
				Inputs: []TxInput{
					{ValueSat: *big.NewInt(0)},
					{AddrDesc: addressToAddrDesc(dbtestdata.AddrS1, d.chainParser), ValueSat: *big.NewInt(1000), Txid: strings.Repeat("b2", 32), Vout: 130, AssetInfo: asset},
					{AddrDesc: addressToAddrDesc(dbtestdata.AddrS2, d.chainParser), ValueSat: *big.NewInt(2000), Txid: strings.Repeat("c3", 32), Vout: 3},
					{ValueSat: *big.NewInt(0), AssetInfo: zeroAsset},
				},
				Outputs: []TxOutput{
					{AddrDesc: addressToAddrDesc(dbtestdata.AddrS1, d.chainParser), ValueSat: *big.NewInt(0), AssetInfo: asset, Spent: true, SpentTxid: strings.Repeat("d4", 32), SpentIndex: 257, SpentHeight: 12346},
					{ValueSat: *big.NewInt(0), AssetInfo: zeroAsset},
					{AddrDesc: addressToAddrDesc(dbtestdata.AddrS2, d.chainParser), ValueSat: *big.NewInt(3000), Spent: true, SpentTxid: strings.Repeat("e5", 32), SpentIndex: 0, SpentHeight: 12347},
					{AddrDesc: addressToAddrDesc(dbtestdata.AddrS1, d.chainParser), ValueSat: *big.NewInt(4000), AssetInfo: asset},
				},
			}
			txid := strings.Repeat("a1", 32)
			btxID, err := d.chainParser.PackTxid(txid)
			require.NoError(t, err)
			wb := grocksdb.NewWriteBatch()
			defer wb.Destroy()
			require.NoError(t, d.storeTxAddresses(wb, map[string]*TxAddresses{string(btxID): ta}))
			require.NoError(t, d.db.Write(d.wo, wb))

			full, err := d.GetTxAddresses(txid)
			require.NoError(t, err)
			require.NotNil(t, full)
			require.Equal(t, ta.Version, full.Version)
			require.Equal(t, ta.Memo, full.Memo)
			for vout := range ta.Outputs {
				got, err := d.GetTxAddressesOutput(txid, uint32(vout))
				require.NoErrorf(t, err, "vout %d", vout)
				require.NotNilf(t, got, "vout %d", vout)
				require.Equalf(t, full.Outputs[vout], *got, "vout %d", vout)
				require.Equalf(t, ta.Outputs[vout].AssetInfo, got.AssetInfo, "vout %d asset", vout)
			}
			got, err := d.GetTxAddressesOutput(txid, uint32(len(ta.Outputs)))
			require.NoError(t, err)
			require.Nil(t, got)
			got, err = d.GetTxAddressesOutput(strings.Repeat("ff", 32), 0)
			require.NoError(t, err)
			require.Nil(t, got)

			// Mempool outpoint resolution must retain asset metadata on both sides.
			_, _, inputAsset := d.AddrDescForOutpoint(bchain.Outpoint{Txid: txid, Vout: ^int32(1)})
			require.Equal(t, asset, inputAsset)
			_, _, outputAsset := d.AddrDescForOutpoint(bchain.Outpoint{Txid: txid, Vout: 3})
			require.Equal(t, asset, outputAsset)
		})
	}
}

func TestSyscoinExtendedSpendingIndexConnectDisconnect(t *testing.T) {
	for _, bulk := range []bool{false, true} {
		name := "normal"
		if bulk {
			name = "bulk"
		}
		t.Run(name, func(t *testing.T) {
			d := setupRocksDBWithExtendedIndex(t, &testSyscoinParser{SyscoinParser: syscoinTestParser()}, true)
			defer closeAndDestroyRocksDB(t, d)
			const assetGuid uint64 = 12345
			const assetValue int64 = 25
			oldAssetCache := AssetCache
			AssetCache = map[uint64]bchain.Asset{assetGuid: {}}
			defer func() { AssetCache = oldAssetCache }()
			txid1, txid2 := strings.Repeat("11", 32), strings.Repeat("22", 32)
			block1 := syscoinAssetTestBlock(d.chainParser, 1, strings.Repeat("01", 32), txid1, []bchain.Vin{{Coinbase: "01"}}, dbtestdata.AddrS1, assetGuid, assetValue)
			block2 := syscoinAssetTestBlock(d.chainParser, 2, strings.Repeat("02", 32), txid2, []bchain.Vin{{Txid: txid1, Vout: 0}}, dbtestdata.AddrS2, assetGuid, assetValue)
			if bulk {
				bc, err := d.InitBulkConnect()
				require.NoError(t, err)
				require.NoError(t, bc.ConnectBlock(block1, false))
				require.NoError(t, bc.ConnectBlock(block2, true))
				require.NoError(t, bc.Close())
			} else {
				require.NoError(t, d.ConnectBlock(block1))
				require.NoError(t, d.ConnectBlock(block2))
			}

			output, err := d.GetTxAddressesOutput(txid1, 0)
			require.NoError(t, err)
			require.NotNil(t, output)
			require.True(t, output.Spent)
			require.Equal(t, txid2, output.SpentTxid)
			require.Equal(t, uint32(0), output.SpentIndex)
			require.Equal(t, uint32(2), output.SpentHeight)
			require.Equal(t, &bchain.AssetInfo{AssetGuid: assetGuid, ValueSat: big.NewInt(assetValue)}, output.AssetInfo)

			require.NoError(t, d.DisconnectBlockRangeBitcoinType(2, 2))
			output, err = d.GetTxAddressesOutput(txid1, 0)
			require.NoError(t, err)
			require.NotNil(t, output)
			require.False(t, output.Spent)
			require.Empty(t, output.SpentTxid)
			require.Zero(t, output.SpentIndex)
			require.Zero(t, output.SpentHeight)
			require.Equal(t, &bchain.AssetInfo{AssetGuid: assetGuid, ValueSat: big.NewInt(assetValue)}, output.AssetInfo)
			removed, err := d.GetTxAddressesOutput(txid2, 0)
			require.NoError(t, err)
			require.Nil(t, removed)
			addrDesc, err := d.chainParser.GetAddrDescFromAddress(dbtestdata.AddrS1)
			require.NoError(t, err)
			balance, err := d.GetAddrDescBalance(addrDesc, AddressBalanceDetailUTXO)
			require.NoError(t, err)
			require.NotNil(t, balance)
			require.Len(t, balance.Utxos, 1)
			require.Equal(t, output.AssetInfo, balance.Utxos[0].AssetInfo)
			require.Equal(t, big.NewInt(assetValue), balance.AssetBalances[assetGuid].BalanceSat)
		})
	}
}
