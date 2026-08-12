package api

import "testing"

func TestDecodeSyscoinBridgeAssetMetadata(t *testing.T) {
	metadata := decodeSyscoinBridgeAssetMetadata("7", []byte(`{"description":"ERC20 Token","assetType":"ERC20","originDecimals":2}`))
	if metadata.Description != "ERC20 Token" || metadata.AssetType != "ERC20" {
		t.Fatalf("decoded metadata = %+v", metadata)
	}
	if metadata.OriginDecimals == nil || *metadata.OriginDecimals != 2 {
		t.Fatalf("origin decimals = %v, want 2", metadata.OriginDecimals)
	}
}

func TestDecodeSyscoinBridgeAssetMetadataPreservesLegacyText(t *testing.T) {
	metadata := decodeSyscoinBridgeAssetMetadata("7", []byte("legacy metadata"))
	if metadata.Description != "legacy metadata" {
		t.Fatalf("description = %q, want legacy metadata", metadata.Description)
	}
	if metadata.AssetType != "" || metadata.OriginDecimals != nil || metadata.TokenID != "" {
		t.Fatalf("legacy metadata unexpectedly gained bridge fields: %+v", metadata)
	}
}

func TestDecodeSyscoinBridgeAssetMetadataRecognizesLegacySYSX(t *testing.T) {
	metadata := decodeSyscoinBridgeAssetMetadata("123456", []byte("Syscoin Native Asset"))
	if metadata.AssetType != "SYSX" || metadata.Description != "Syscoin Native Asset" {
		t.Fatalf("legacy SYSX metadata = %+v", metadata)
	}
}
