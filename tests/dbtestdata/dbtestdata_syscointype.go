package dbtestdata

import (
	"github.com/syscoin/blockbook/bchain"
	"math/big"
)

const (
    TxidS1T0 = "8d86636db959a190aed4e65b4ee7e67b6ee0189e03acc27e353e69b88288cacc"
    TxidS2T0 = "5bb051670143eeb1d0cfc3c992ab18e1bd4bb0c78d8914dc54feaee9a894174b"

    // We'll keep S1, S2 addresses as placeholders
    AddrS1 = "tsys1q4hg3e2lcyx87muctu26dvmnuz7lpm3lpvcaeyu"
    AddrS2 = "tsys1qq43tjdd753rct3jj39yvr855gytwf3y8p5kuf9"

    // Possibly keep these OP_RETURN if we need bridging
    TxidS1T0OutputReturn = "6a24aa21a9ed38a1...."
    TxidS2T0OutputReturn = "6a24aa21a9ed6866...."
)

// Amounts in satoshis
var (
    SatS1T0A1 = big.NewInt(3465003450)
    SatS2T0A1 = big.NewInt(3465003950)
)


// GetTestSyscoinTypeBlock1 returns block #1
func GetTestSyscoinTypeBlock1(parser bchain.BlockChainParser) *bchain.Block {
	return &bchain.Block{
		BlockHeader: bchain.BlockHeader{
			Height:        112,
			Hash:          "00000797cfd9074de37a557bf0d47bd86c45846f31e163ba688e14dfc498527a",
			Size:          503,
			Time:          1598556954,
			Confirmations: 2,
		},
		Txs: []bchain.Tx{
			{
				Txid: TxidS1T0,
				Vin: []bchain.Vin{
					{
						Coinbase: "01700101",
					},
				},
				Vout: []bchain.Vout{
					{
						N: 0,
						ScriptPubKey: bchain.ScriptPubKey{
							Hex: AddressToPubKeyHex(AddrS1, parser),
						},
						ValueSat: *SatS1T0A1,
					},
					{
						N: 1,
						ScriptPubKey: bchain.ScriptPubKey{
							Hex: TxidS1T0OutputReturn, // OP_RETURN script
						},
						ValueSat: *SatZero,
					},
				},
				Blocktime:     1598556954,
				Time:          1598556954,
				Confirmations: 2,
			},
		},
	}
}


// GetTestSyscoinTypeBlock2 returns block #2
func GetTestSyscoinTypeBlock2(parser bchain.BlockChainParser) *bchain.Block {
	return &bchain.Block{
		BlockHeader: bchain.BlockHeader{
			Height:        113,
			Hash:          "00000cade5f8d530b3f0a3b6c9dceaca50627838f2c6fffb807390cba71974e7",
			Size:          554,
			Time:          1598557012,
			Confirmations: 1,
		},
		Txs: []bchain.Tx{
			{
				Txid: TxidS2T0,
				Vin: []bchain.Vin{
					{
						Coinbase: "01710101",
					},
				},
				Vout: []bchain.Vout{
					{
						N: 0,
						ScriptPubKey: bchain.ScriptPubKey{
							Hex: AddressToPubKeyHex(AddrS2, parser),
						},
						ValueSat: *SatS2T0A1,
					},
					{
						N: 1,
						ScriptPubKey: bchain.ScriptPubKey{
							Hex: TxidS2T0OutputReturn, // OP_RETURN script
						},
						ValueSat: *SatZero,
					},
				},
				Blocktime:     1598557012,
				Time:          1598557012,
				Confirmations: 1,
			},
		},
	}
}
