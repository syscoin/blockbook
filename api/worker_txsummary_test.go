//go:build unittest

package api

import (
	"encoding/json"
	"math/big"
	"testing"

	"github.com/trezor/blockbook/bchain"
)

func TestSummarizeTxForAccountInputOwnership(t *testing.T) {
	owned := bchain.AddressDescriptor{1}
	ownedChange := bchain.AddressDescriptor{2}
	foreign := bchain.AddressDescriptor{3}
	accounts := map[string]struct{}{string(owned): {}, string(ownedChange): {}}
	amount := func(value int64) *Amount { return (*Amount)(big.NewInt(value)) }
	tests := []struct {
		name string
		vin  []Vin
		want bool
	}{
		{"one owned input", []Vin{{AddrDesc: owned, ValueSat: amount(100000226)}}, true},
		{"multiple owned inputs", []Vin{{AddrDesc: owned, ValueSat: amount(60000000)}, {AddrDesc: ownedChange, ValueSat: amount(40000226)}}, true},
		{"mixed ownership", []Vin{{AddrDesc: owned, ValueSat: amount(60000000)}, {AddrDesc: foreign, ValueSat: amount(40000226)}}, false},
		{"only foreign inputs", []Vin{{AddrDesc: foreign, ValueSat: amount(100000226)}}, false},
		{"zero owned input", []Vin{{AddrDesc: owned, ValueSat: amount(0)}}, false},
		{"foreign zero input alongside owned funding", []Vin{{AddrDesc: owned, ValueSat: amount(100000226)}, {AddrDesc: foreign, ValueSat: amount(0)}}, false},
		{"unknown value", []Vin{{AddrDesc: owned}}, false},
		{"negative value", []Vin{{AddrDesc: owned, ValueSat: amount(-1)}}, false},
		{"missing descriptor", []Vin{{ValueSat: amount(100000226)}}, false},
		{"empty inputs", []Vin{}, false},
		{"nil inputs", nil, false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			tx := &Tx{Txid: "tx", Vin: test.vin, Vout: []Vout{}, FeesSat: amount(226)}
			summary := (&Worker{}).summarizeTxForAccount(tx, accounts)
			if summary.AccountOwnsAllInputs == nil || *summary.AccountOwnsAllInputs != test.want {
				t.Fatalf("accountOwnsAllInputs = %v, want explicit %v", summary.AccountOwnsAllInputs, test.want)
			}
			if tx.AccountOwnsAllInputs != nil {
				t.Fatal("summary changed the materialized transaction")
			}
			if summary.Vin == nil || len(summary.Vin) != 0 || summary.Vout == nil || len(summary.Vout) != 0 {
				t.Fatal("summary must retain empty input/output arrays")
			}
		})
	}
	if summary := (&Worker{}).summarizeTxForAccount(nil, accounts); summary != nil {
		t.Fatal("nil transaction must still produce nil summary")
	}
	for _, descriptors := range []map[string]struct{}{nil, {}, {"": {}}} {
		tx := &Tx{Vin: []Vin{{AddrDesc: owned, ValueSat: amount(100000226)}}}
		summary := (&Worker{}).summarizeTxForAccount(tx, descriptors)
		if summary.AccountOwnsAllInputs == nil || *summary.AccountOwnsAllInputs {
			t.Fatal("missing account descriptors must not establish ownership")
		}
	}
	missingDescriptor := &Tx{Vin: []Vin{{ValueSat: amount(100000226)}}}
	summary := (&Worker{}).summarizeTxForAccount(missingDescriptor, map[string]struct{}{"": {}})
	if summary.AccountOwnsAllInputs == nil || *summary.AccountOwnsAllInputs {
		t.Fatal("an empty descriptor must not match an empty account descriptor")
	}
}

func TestSummarizeTxForAccountOwnershipPreservesAmounts(t *testing.T) {
	owned := bchain.AddressDescriptor{1}
	foreign := bchain.AddressDescriptor{2}
	accounts := map[string]struct{}{string(owned): {}}
	amount := func(value int64) *Amount { return (*Amount)(big.NewInt(value)) }
	for _, self := range []bool{false, true} {
		name := "external spend"
		accountOut := int64(30000000)
		outputs := []Vout{{AddrDesc: owned, ValueSat: amount(accountOut)}, {AddrDesc: foreign, ValueSat: amount(70000000)}}
		if self {
			name = "self spend"
			accountOut = 100000000
			outputs = []Vout{{AddrDesc: owned, ValueSat: amount(accountOut)}}
		}
		t.Run(name, func(t *testing.T) {
			tx := &Tx{
				Txid: "tx", Vin: []Vin{{AddrDesc: owned, ValueSat: amount(100000226)}}, Vout: outputs,
				ValueInSat: amount(100000226), ValueOutSat: amount(100000000), FeesSat: amount(226),
				Confirmations: 1, Blockheight: 100, Blocktime: 1700000000,
			}
			summary := (&Worker{}).summarizeTxForAccount(tx, accounts)
			if summary.AccountOwnsAllInputs == nil || !*summary.AccountOwnsAllInputs {
				t.Fatal("expected explicit complete input ownership")
			}
			if summary.AddressValueInSat.String() != "100000226" || summary.ValueInSat.String() != "100000226" || summary.AddressValueOutSat.AsInt64() != accountOut || summary.ValueOutSat.AsInt64() != accountOut {
				t.Fatal("account totals and legacy aliases changed")
			}
			if summary.FeesSat != tx.FeesSat || summary.Confirmations != tx.Confirmations || summary.Blockheight != tx.Blockheight || summary.Blocktime != tx.Blocktime {
				t.Fatal("fees or confirmation metadata changed")
			}
			if tx.ValueOutSat.String() != "100000000" || tx.AccountOwnsAllInputs != nil {
				t.Fatal("materialized transaction was modified")
			}
		})
	}
}

func TestTxAccountOwnsAllInputsJSON(t *testing.T) {
	for _, test := range []struct {
		name  string
		value *bool
		want  interface{}
	}{
		{"not a summary", nil, nil},
		{"owned summary", boolPointer(true), true},
		{"unproven summary", boolPointer(false), false},
	} {
		t.Run(test.name, func(t *testing.T) {
			encoded, err := json.Marshal(&Tx{AccountOwnsAllInputs: test.value})
			if err != nil {
				t.Fatal(err)
			}
			var fields map[string]interface{}
			if err := json.Unmarshal(encoded, &fields); err != nil {
				t.Fatal(err)
			}
			value, present := fields["accountOwnsAllInputs"]
			if present != (test.value != nil) || value != test.want {
				t.Fatalf("accountOwnsAllInputs = %v (present %v), want %v", value, present, test.want)
			}
		})
	}
}

func boolPointer(value bool) *bool { return &value }
