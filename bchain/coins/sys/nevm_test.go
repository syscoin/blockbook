package syscoin

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

func TestVaultManagerAddressUsesBridgeV2(t *testing.T) {
	want := common.HexToAddress("0x28bD37C0926575f2568ea8f297c0745EF16174Ab")
	if got := common.HexToAddress(vaultManagerAddress); got != want {
		t.Fatalf("vault manager address = %s, want Bridge V2 %s", got.Hex(), want.Hex())
	}
}

func TestFetchNEVMAssetDetailsSYSXDoesNotCallVault(t *testing.T) {
	asset, err := (&NEVMClient{}).FetchNEVMAssetDetails(123456)
	if err != nil {
		t.Fatal(err)
	}
	if got := string(asset.AssetObj.Symbol); got != "SYSX" {
		t.Fatalf("SYSX symbol = %q, want SYSX", got)
	}
	if got := asset.AssetObj.Precision; got != 8 {
		t.Fatalf("SYSX precision = %d, want 8", got)
	}
}
