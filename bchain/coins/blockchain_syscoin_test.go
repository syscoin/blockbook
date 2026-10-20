//go:build unittest

// SYSCOIN: Regression coverage for broadcast options and their metrics wrapper.
package coins

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/trezor/blockbook/bchain"
	"github.com/trezor/blockbook/common"
)

func TestSyscoinBroadcastOptionsThroughMetrics(t *testing.T) {
	for _, coin := range []string{"Syscoin", "Syscoin Testnet"} {
		t.Run(coin, func(t *testing.T) {
			requests := make(chan []string, 3)
			backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var request struct {
					Method string   `json:"method"`
					Params []string `json:"params"`
				}
				if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
					t.Errorf("decode request: %v", err)
					http.Error(w, "invalid request", http.StatusBadRequest)
					return
				}
				if request.Method != "sendrawtransaction" {
					t.Errorf("unexpected method %q", request.Method)
				}
				requests <- request.Params
				w.Header().Set("Content-Type", "application/json")
				if len(request.Params) > 0 && request.Params[0] == "rejected" {
					_, _ = w.Write([]byte(`{"error":{"code":-26,"message":"txn-mempool-conflict"}}`))
					return
				}
				_, _ = w.Write([]byte(`{"result":"accepted"}`))
			}))
			defer backend.Close()
			config, err := json.Marshal(map[string]interface{}{"rpc_url": backend.URL, "rpc_timeout": 5})
			if err != nil {
				t.Fatal(err)
			}
			factory, ok := BlockChainFactories[coin]
			if !ok {
				t.Fatalf("missing %s chain factory", coin)
			}
			chain, err := factory(config, nil)
			if err != nil {
				t.Fatal(err)
			}
			defer chain.Shutdown(context.Background())
			metrics := &common.Metrics{
				RPCLatency: prometheus.NewHistogramVec(prometheus.HistogramOpts{
					Name: "test_syscoin_rpc_latency",
				}, []string{"method", "error"}),
				SendTxRequests: prometheus.NewCounterVec(prometheus.CounterOpts{
					Name: "test_syscoin_send_requests_total",
				}, []string{"status", "reason"}),
			}
			wrapped := &blockChainWithMetrics{b: chain, m: metrics}
			if txid, err := wrapped.SendRawTransaction("default", false); err != nil || txid != "accepted" {
				t.Fatalf("default broadcast = %q, %v", txid, err)
			}
			if got, want := <-requests, []string{"default", "0.10", "150"}; !reflect.DeepEqual(got, want) {
				t.Fatalf("default broadcast params = %v, want %v", got, want)
			}
			maxFeeRate, maxBurnAmount := "0.02", "200"
			if txid, err := wrapped.SendRawTransactionWithOpts(bchain.SendRawTransactionParams{
				Hex: "custom", MaxFeeRate: &maxFeeRate, MaxBurnAmount: &maxBurnAmount,
			}); err != nil || txid != "accepted" {
				t.Fatalf("custom broadcast = %q, %v", txid, err)
			}
			if got, want := <-requests, []string{"custom", "0.02", "200"}; !reflect.DeepEqual(got, want) {
				t.Fatalf("custom broadcast params = %v, want %v", got, want)
			}
			if _, err := wrapped.SendRawTransactionWithOpts(bchain.SendRawTransactionParams{Hex: "rejected"}); err == nil {
				t.Fatal("backend rejection was lost")
			}
			if got, want := <-requests, []string{"rejected", "0.10", "150"}; !reflect.DeepEqual(got, want) {
				t.Fatalf("rejected broadcast params = %v, want %v", got, want)
			}
			for _, expected := range []struct {
				status string
				reason string
				count  float64
			}{
				{bchain.SendTxStatusSuccess, bchain.ReasonOK, 2},
				{bchain.SendTxStatusFailure, bchain.ReasonConflict, 1},
			} {
				var metric dto.Metric
				counter := metrics.SendTxRequests.WithLabelValues(expected.status, expected.reason)
				if err := counter.Write(&metric); err != nil {
					t.Fatal(err)
				}
				if got := metric.GetCounter().GetValue(); got != expected.count {
					t.Errorf("broadcast metric %s/%s = %v, want %v", expected.status, expected.reason, got, expected.count)
				}
			}
		})
	}
}
