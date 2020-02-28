package dbtestdata

import (
	"blockbook/bchain"
	"math/big"
)

// Txids, Xpubs and Addresses
const (
	TxidS1T1 = "00b2c06055e5e90e9c82bd4181fde310104391a7fa4f289b1704e5d90caa3840"
	TxidS1T2 = "effd9ef509383d536b1c8af5bf434c8efbf521a4f2befd4022bbd68694b4ac75"
	TxidS2T1 = "7c3be24063f268aaa1ed81b64776798f56088757641a34fb156c4f51ed2e9d25"
	TxidS2T2 = "3d90d15ed026dc45e19ffb52875ed18fa9e8012ad123d7f7212176e2b0ebdb71"
	TxidS2T3 = "05e2e48aeabdd9b75def7b48d756ba304713c2aba7b522bf9dbc893fc4231b07"
	TxidS2T4 = "fdd824a780cbb718eeb766eb05d83fdefc793a27082cd5e67f856d69798cf7db"

	AddrS1 = "mfcWp7DB6NuaZsExybTTXpVgWz559Np4Ti"  // 76a914010d39800f86122416e28f485029acf77507169288ac
	AddrS2 = "mtGXQvBowMkBpnhLckhxhbwYK44Gs9eEtz"  // 76a9148bdf0aa3c567aa5975c2e61321b8bebbe7293df688ac
	AddrS3 = "mv9uLThosiEnGRbVPS7Vhyw6VssbVRsiAw"  // 76a914a08eae93007f22668ab5e4a9c83c8cd1c325e3e088ac
	AddrS4 = "2MzmAKayJmja784jyHvRUW1bXPget1csRRG" // a91452724c5178682f70e0ba31c6ec0633755a3b41d987, xpub m/49'/1'/33'/0/0
	AddrS5 = "2NEVv9LJmAnY99W1pFoc5UJjVdypBqdnvu1" // a914e921fc4912a315078f370d959f2c4f7b6d2a683c87
	AddrS6 = "mzB8cYrfRwFRFAGTDzV8LkUQy5BQicxGhX"  // 76a914ccaaaf374e1b06cb83118453d102587b4273d09588ac
	AddrS7 = "mtR97eM2HPWVM6c8FGLGcukgaHHQv7THoL"  // 76a9148d802c045445df49613f6a70ddd2e48526f3701f88ac
	AddrS8 = "2N6utyMZfPNUb1Bk8oz7p2JqJrXkq83gegu" // a91495e9fbe306449c991d314afe3c3567d5bf78efd287, xpub m/49'/1'/33'/1/3
	AddrS9 = "mmJx9Y8ayz9h14yd9fgCW1bUKoEpkBAquP"  // 76a9143f8ba3fda3ba7b69f5818086e12223c6dd25e3c888ac
	AddrSA = "mzVznVsCHkVHX9UN8WPFASWUUHtxnNn4Jj"  // 76a914d03c0d863d189b23b061a95ad32940b65837609f88ac

	TxidS2T1Output3OpReturn = "6a072020f1686f6a20"
)

// Amounts in satoshis
var (
	SatS1T1A1       = big.NewInt(100000000)
	SatS1T1A2       = big.NewInt(12345)
	SatS1T1A2Double = big.NewInt(12345 * 2)
	SatS1T2A3       = big.NewInt(1234567890123)
	SatS1T2A4       = big.NewInt(1)
	SatS1T2A5       = big.NewInt(9876)
	SatS2T1A6       = big.NewInt(317283951061)
	SatS2T1A7       = big.NewInt(917283951061)
	SatS2T2A8       = big.NewInt(118641975500)
	SatS2T2A9       = big.NewInt(198641975500)
	SatS2T3A5       = big.NewInt(9000)
	SatS2T4AA       = big.NewInt(1360030331)
)

// GetTestSyscoinTypeBlock1 returns block #1
func GetTestSyscoinTypeBlock1(parser bchain.BlockChainParser) *bchain.Block {
	return &bchain.Block{
		BlockHeader: bchain.BlockHeader{
			Height:        225493,
			Hash:          "0000000076fbbed90fd75b0e18856aa35baa984e9c9d444cf746ad85e94e2997",
			Size:          1234567,
			Time:          1521515026,
			Confirmations: 2,
		},
		Txs: []bchain.Tx{
			{
				Txid: TxidS1T1,
				Vin:  []bchain.Vin{},
				Vout: []bchain.Vout{
					{
						N: 0,
						ScriptPubKey: bchain.ScriptPubKey{
							Hex: AddressToPubKeyHex(AddrS1, parser),
						},
						ValueSat: *SatS1T1A1,
					},
					{
						N: 1,
						ScriptPubKey: bchain.ScriptPubKey{
							Hex: AddressToPubKeyHex(AddrS2, parser),
						},
						ValueSat: *SatS1T1A2,
					},
					{
						N: 2,
						ScriptPubKey: bchain.ScriptPubKey{
							Hex: AddressToPubKeyHex(AddrS2, parser),
						},
						ValueSat: *SatS1T1A2,
					},
				},
				Blocktime:     1521515026,
				Time:          1521515026,
				Confirmations: 2,
			},
			{
				Txid: TxidS1T2,
				Vout: []bchain.Vout{
					{
						N: 0,
						ScriptPubKey: bchain.ScriptPubKey{
							Hex: AddressToPubKeyHex(AddrS3, parser),
						},
						ValueSat: *SatS1T2A3,
					},
					{
						N: 1,
						ScriptPubKey: bchain.ScriptPubKey{
							Hex: AddressToPubKeyHex(AddrS4, parser),
						},
						ValueSat: *SatS1T2A4,
					},
					{
						N: 2,
						ScriptPubKey: bchain.ScriptPubKey{
							Hex: AddressToPubKeyHex(AddrS5, parser),
						},
						ValueSat: *SatS1T2A5,
					},
				},
				Blocktime:     1521515026,
				Time:          1521515026,
				Confirmations: 2,
			},
		},
	}
}

// GetTestSyscoinTypeBlock2 returns block #2
func GetTestSyscoinTypeBlock2(parser bchain.BlockChainParser) *bchain.Block {
	return &bchain.Block{
		BlockHeader: bchain.BlockHeader{
			Height:        225494,
			Hash:          "00000000eb0443fd7dc4a1ed5c686a8e995057805f9a161d9a5a77a95e72b7b6",
			Size:          2345678,
			Time:          1521595678,
			Confirmations: 1,
		},
		Txs: []bchain.Tx{
			{
				Txid: TxidS2T1,
				Vin: []bchain.Vin{
					// addr3
					{
						Txid: TxidS1T2,
						Vout: 0,
					},
					// addr2
					{
						Txid: TxidS1T1,
						Vout: 1,
					},
				},
				Vout: []bchain.Vout{
					{
						N: 0,
						ScriptPubKey: bchain.ScriptPubKey{
							Hex: AddressToPubKeyHex(AddrS6, parser),
						},
						ValueSat: *SatS2T1A6,
					},
					{
						N: 1,
						ScriptPubKey: bchain.ScriptPubKey{
							Hex: AddressToPubKeyHex(AddrS7, parser),
						},
						ValueSat: *SatS2T1A7,
					},
					{
						N: 2,
						ScriptPubKey: bchain.ScriptPubKey{
							Hex: TxidS2T1Output3OpReturn, // OP_RETURN script
						},
						ValueSat: *SatZero,
					},
				},
				Blocktime:     1521595678,
				Time:          1521595678,
				Confirmations: 1,
			},
			{
				Txid: TxidS2T2,
				Vin: []bchain.Vin{
					// spending an output in the same block - addr6
					{
						Txid: TxidS2T1,
						Vout: 0,
					},
					// spending an output in the previous block - addr4
					{
						Txid: TxidS1T2,
						Vout: 1,
					},
				},
				Vout: []bchain.Vout{
					{
						N: 0,
						ScriptPubKey: bchain.ScriptPubKey{
							Hex: AddressToPubKeyHex(AddrS8, parser),
						},
						ValueSat: *SatS2T2A8,
					},
					{
						N: 1,
						ScriptPubKey: bchain.ScriptPubKey{
							Hex: AddressToPubKeyHex(AddrS9, parser),
						},
						ValueSat: *SatS2T2A9,
					},
				},
				Blocktime:     1521595678,
				Time:          1521595678,
				Confirmations: 1,
			},
			// transaction from the same address in the previous block
			{
				Txid: TxidS2T3,
				Vin: []bchain.Vin{
					// addr5
					{
						Txid: TxidS1T2,
						Vout: 2,
					},
				},
				Vout: []bchain.Vout{
					{
						N: 0,
						ScriptPubKey: bchain.ScriptPubKey{
							Hex: AddressToPubKeyHex(AddrS5, parser),
						},
						ValueSat: *SatS2T3A5,
					},
				},
				Blocktime:     1521595678,
				Time:          1521595678,
				Confirmations: 1,
			},
			// mining transaction
			{
				Txid: TxidS2T4,
				Vin: []bchain.Vin{
					{
						Coinbase: "03bf1e1504aede765b726567696f6e312f50726f6a65637420425443506f6f6c2f01000001bf7e000000000000",
					},
				},
				Vout: []bchain.Vout{
					{
						N: 0,
						ScriptPubKey: bchain.ScriptPubKey{
							Hex: AddressToPubKeyHex(AddrSA, parser),
						},
						ValueSat: *SatS2T4AA,
					},
					{
						N:            1,
						ScriptPubKey: bchain.ScriptPubKey{},
						ValueSat:     *SatZero,
					},
				},
				Blocktime:     1521595678,
				Time:          1521595678,
				Confirmations: 1,
			},
		},
	}
}
