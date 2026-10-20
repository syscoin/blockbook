//go:build unittest

package main

import (
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/trezor/blockbook/api"
	"github.com/trezor/blockbook/bchain"
)

type probeInner struct {
	Value string `json:"value"`
}

type probePayloadA struct {
	A string `json:"a"`
}

type probePayloadB struct {
	B string `json:"b"`
}

type probeWrapper struct {
	PayloadType string `json:"payloadType"`
}

func (probeWrapper) ChainExtraPayload(p bchain.ChainExtraPayload) interface{} { return p.Tx }

type probeOuter struct {
	Always *probeInner   `json:"always" ts_nullable:"true"`
	Maybe  *probeInner   `json:"maybe,omitempty"`
	Extra  *probeWrapper `json:"extra,omitempty"`
}

var probePayloads = []bchain.ChainExtraPayload{
	{Type: "a", Tx: probePayloadA{}, Account: probePayloadA{}},
	{Type: "b", Tx: probePayloadB{}, Account: probePayloadB{}},
}

type probeOmitEmpty struct {
	Bad *probeInner `json:"bad,omitempty" ts_nullable:"true"`
}

type probeNonPointer struct {
	Bad probeInner `json:"bad" ts_nullable:"true"`
}

// TestCommittedFileIsCurrent fails while blockbook-api.ts differs from what the generator emits.
func TestCommittedFileIsCurrent(t *testing.T) {
	committed, err := os.ReadFile(filepath.Join("..", "..", "..", outputFile))
	if err != nil {
		t.Fatal(err)
	}
	want, err := generate(apiTypes, bchain.ChainExtraPayloads)
	if err != nil {
		t.Fatal(err)
	}
	if string(committed) != want {
		t.Fatalf("%s is stale, run 'make typescriptify'", outputFile)
	}
}

func TestGenerateNullableAndUnion(t *testing.T) {
	out, err := generate([]interface{}{probeOuter{}}, probePayloads)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		header,
		"export type probeWrapper = { payloadType: 'a'; payload?: probePayloadA } | { payloadType: 'b'; payload?: probePayloadB };\n",
		"export interface probePayloadA {\n",
		"export interface probePayloadB {\n",
		"\n    always: probeInner | null;\n",
		"\n    maybe?: probeInner;\n",
		"\n    extra?: probeWrapper;\n",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("output lacks %q:\n%s", want, out)
		}
	}
}

func TestGenerateRejectsShadowedUnion(t *testing.T) {
	if _, err := generate([]interface{}{probeWrapper{}}, probePayloads); err == nil {
		t.Error("expected an error for a wrapper listed as a root")
	}
}

func TestGenerateEmitsUnionOnlyWhenReachable(t *testing.T) {
	out, err := generate([]interface{}{probeInner{}}, probePayloads)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out, "probeWrapper") {
		t.Errorf("unreachable wrapper leaked into the output:\n%s", out)
	}
}

func TestGenerateRejectsBadPayloadRegistry(t *testing.T) {
	for name, payloads := range map[string][]bchain.ChainExtraPayload{
		"empty":        nil,
		"unknown type": {{Type: bchain.ChainExtraPayloadTypeUnknown, Tx: probePayloadA{}, Account: probePayloadA{}}},
		"non-struct":   {{Type: "a", Tx: "x", Account: probePayloadA{}}},
	} {
		if _, err := generate([]interface{}{probeOuter{}}, payloads); err == nil {
			t.Errorf("%s: expected an error", name)
		}
	}
}

func TestGenerateRejectsMisusedNullable(t *testing.T) {
	for _, root := range []interface{}{probeOmitEmpty{}, probeNonPointer{}} {
		_, err := generate([]interface{}{root}, probePayloads)
		if err == nil || !strings.Contains(err.Error(), "ts_nullable") {
			t.Errorf("%T: expected ts_nullable misuse error, got %v", root, err)
		}
	}
}

func TestMarkNullableIsScopedToInterface(t *testing.T) {
	body := "\nexport interface A {\n    value?: string;\n}\nexport interface B {\n    value?: string;\n}"
	out, err := applyNullable(body, []nullableField{{iface: "B", json: "value"}}, "    ")
	if err != nil {
		t.Fatal(err)
	}
	want := "\nexport interface A {\n    value?: string;\n}\nexport interface B {\n    value: string | null;\n}"
	if out != want {
		t.Errorf("got:\n%s\nwant:\n%s", out, want)
	}
	if _, err := applyNullable(body, []nullableField{{iface: "B", json: "missing"}}, "    "); err == nil {
		t.Error("expected error for a field the interface does not have")
	}
}

func TestTransactionMemoMatchesJSONWireType(t *testing.T) {
	memo := []byte{0, 1, 2, 0xff, 0x0a}
	encoded, err := json.Marshal(api.Tx{Memo: memo})
	if err != nil {
		t.Fatal(err)
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(encoded, &fields); err != nil {
		t.Fatal(err)
	}
	var wireMemo string
	if err := json.Unmarshal(fields["memo"], &wireMemo); err != nil {
		t.Fatalf("transaction memo must be a JSON string: %v", err)
	}
	if want := base64.StdEncoding.EncodeToString(memo); wireMemo != want {
		t.Fatalf("transaction memo = %q, want base64 %q", wireMemo, want)
	}
	out, err := generate([]interface{}{api.Tx{}}, bchain.ChainExtraPayloads)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "\n    memo?: string;\n") {
		t.Fatal("generated transaction memo does not match its optional JSON string representation")
	}
}

func TestSyscoinTokenStandardsMatchGeneratedTypes(t *testing.T) {
	for _, tc := range []struct {
		name     string
		root     interface{}
		standard bchain.TokenStandardName
	}{
		{"Token", api.Token{}, bchain.SPTTokenType},
		{"TokenTransfer", api.TokenTransfer{}, bchain.TokenStandardName("SPT")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			out, err := generate([]interface{}{tc.root}, bchain.ChainExtraPayloads)
			if err != nil {
				t.Fatal(err)
			}
			start := strings.Index(out, "export interface "+tc.name+" {\n")
			if start < 0 {
				t.Fatalf("missing %s interface", tc.name)
			}
			body := strings.SplitN(out[start:], "\n}", 2)[0]
			for _, field := range []string{"type", "standard"} {
				found := false
				for _, line := range strings.Split(body, "\n") {
					if strings.HasPrefix(strings.TrimSpace(line), field+":") {
						found = strings.Contains(line, "'"+string(tc.standard)+"'")
						break
					}
				}
				if !found {
					t.Errorf("%s.%s excludes emitted Syscoin value %q", tc.name, field, tc.standard)
				}
			}
		})
	}
}
