package syscoin

import (
	"encoding/json"
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
	var metadata bridgeAssetMetadata
	if err := json.Unmarshal(asset.MetaData, &metadata); err != nil {
		t.Fatal(err)
	}
	if metadata.AssetType != "SYSX" {
		t.Fatalf("SYSX asset type = %q, want SYSX", metadata.AssetType)
	}
}

func TestEncodeBridgeAssetMetadataPreservesZeroOriginDecimals(t *testing.T) {
	originDecimals := uint8(0)
	encoded, err := encodeBridgeAssetMetadata(bridgeAssetMetadata{
		Description:    "ERC1155 Token ID 7",
		AssetType:      "ERC1155",
		OriginDecimals: &originDecimals,
		TokenID:        "7",
	})
	if err != nil {
		t.Fatal(err)
	}

	var decoded bridgeAssetMetadata
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.OriginDecimals == nil || *decoded.OriginDecimals != 0 {
		t.Fatalf("origin decimals = %v, want explicit zero", decoded.OriginDecimals)
	}
	if decoded.AssetType != "ERC1155" || decoded.TokenID != "7" {
		t.Fatalf("decoded metadata = %+v", decoded)
	}
}

func TestSanitizeBridgeTokenSymbolRemovesControlsAndBoundsLength(t *testing.T) {
	got := sanitizeBridgeTokenSymbol("  SYS\u202eFAKE\x00" + "1234567890123456789012345678901234567890")
	want := "SYSFAKE1234567890123456789012345"
	if got != want {
		t.Fatalf("sanitized symbol = %q, want %q", got, want)
	}
	if len([]rune(got)) != maxBridgeTokenSymbolRunes {
		t.Fatalf("sanitized symbol length = %d, want %d", len([]rune(got)), maxBridgeTokenSymbolRunes)
	}
}
