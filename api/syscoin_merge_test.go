//go:build unittest

// SYSCOIN: Regression coverage for asset filtering and summary pagination across upstream merges.
package api

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/trezor/blockbook/bchain"
	"github.com/trezor/blockbook/tests/dbtestdata"
)

func TestGetAddressSummaryPagesMempoolWithConfirmed(t *testing.T) {
	const pageSize = 2
	w, pending, confirmed := buildPagingWorker(t, 3, 3)
	want := append(append([]string{}, pending...), confirmed...)
	var walked []string
	for page := 1; ; page++ {
		addr, err := w.GetAddress(dbtestdata.Addr1, page, pageSize, AccountDetailsTxHistorySummary,
			&AddressFilter{Vout: AddressFilterVoutOff}, "")
		require.NoError(t, err)
		require.Equal(t, 3, addr.TotalPages)
		require.LessOrEqual(t, len(addr.Transactions), pageSize)
		for _, tx := range addr.Transactions {
			require.Empty(t, tx.Vin)
			require.Empty(t, tx.Vout)
			require.Equal(t, "received", tx.Direction)
			require.NotNil(t, tx.AddressValueOutSat)
			walked = append(walked, tx.Txid)
		}
		if page >= addr.TotalPages {
			break
		}
	}
	require.Equal(t, want, walked, "summary pages must return every pending and confirmed tx once")
}

func TestGetXpubSummaryPagesMempoolWithConfirmed(t *testing.T) {
	w, pending := buildXpubPagingWorker(t)
	confirmed, err := w.GetXpubAddress(dbtestdata.Xpub, 1, 100, AccountDetailsTxidHistory,
		&AddressFilter{Vout: AddressFilterVoutOff, OnlyConfirmed: true}, 5, "")
	require.NoError(t, err)
	want := append([]string{pending}, confirmed.Txids...)
	var walked []string
	for page := 1; ; page++ {
		addr, err := w.GetXpubAddress(dbtestdata.Xpub, page, 1, AccountDetailsTxHistorySummary,
			&AddressFilter{Vout: AddressFilterVoutOff}, 5, "")
		require.NoError(t, err)
		require.Len(t, addr.Transactions, 1)
		tx := addr.Transactions[0]
		require.Empty(t, tx.Vin)
		require.Empty(t, tx.Vout)
		require.NotNil(t, tx.AddressValueInSat)
		require.NotNil(t, tx.AddressValueOutSat)
		walked = append(walked, tx.Txid)
		if page >= addr.TotalPages {
			break
		}
	}
	require.Equal(t, want, walked)
}

func TestGetAddressAssetMaskStopsAtScanBudget(t *testing.T) {
	for _, tc := range []struct {
		name string
		vout int
	}{{"asset only", AddressFilterVoutOff}, {"asset and output", AddressFilterVoutOutputs}} {
		t.Run(tc.name, func(t *testing.T) {
			w, _, _ := buildPagingWorker(t, 10, 0)
			setFilteredScanBudget(t, 5)
			_, err := w.GetAddress(dbtestdata.Addr1, 1, 25, AccountDetailsTxidHistory,
				&AddressFilter{Vout: tc.vout, AssetsMask: bchain.AssetAllocationSendMask}, "")
			require.Equal(t, errFilteredScanLimit, err, "asset-mask rejections must count toward the same sparse-scan budget")
		})
	}
}

func TestGetAddressAssetMaskWithinScanBudget(t *testing.T) {
	w, _, confirmed := buildPagingWorker(t, 3, 0)
	setFilteredScanBudget(t, 3)
	addr, err := w.GetAddress(dbtestdata.Addr1, 1, 25, AccountDetailsTxidHistory,
		&AddressFilter{Vout: AddressFilterVoutOff, AssetsMask: bchain.AssetAllocationSendMask}, "")
	require.NoError(t, err)
	require.Empty(t, addr.Txids)
	addr, err = w.GetAddress(dbtestdata.Addr1, 1, 25, AccountDetailsTxidHistory,
		&AddressFilter{Vout: AddressFilterVoutOff, AssetsMask: bchain.BaseCoinMask}, "")
	require.NoError(t, err)
	require.Equal(t, confirmed, addr.Txids)
}
