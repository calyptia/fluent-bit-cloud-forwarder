package forwarder

import (
	"bytes"
	"testing"
	"time"

	cmetrics "github.com/calyptia/cmetrics-go"
	fluentbit "github.com/calyptia/go-fluent-bit-metrics"
)

func Test_fluentBitMetricsToCMetrics(t *testing.T) {
	now := time.Now().Truncate(time.Nanosecond)
	tt := []struct {
		name           string
		metrics        fluentbit.Metrics
		storageMetrics fluentbit.StorageMetrics
	}{
		{
			metrics: fluentbit.Metrics{
				Input: map[string]fluentbit.MetricInput{
					"testinput.0": {
						Records: 10,
						Bytes:   12,
					},
				},
				Output: map[string]fluentbit.MetricOutput{
					"testoutput.0": {
						ProcRecords:   2,
						ProcBytes:     100,
						Errors:        1,
						Retries:       5,
						RetriesFailed: 4,
					},
				},
			},
			storageMetrics: fluentbit.StorageMetrics{
				InputChunks: map[string]fluentbit.PluginStorage{
					"http.0": {
						Status: struct {
							Overlimit bool   `json:"overlimit"`
							MemSize   string `json:"mem_size"`
							MemLimit  string `json:"mem_limit"`
						}(struct {
							Overlimit bool
							MemSize   string
							MemLimit  string
						}{Overlimit: false, MemSize: "1b", MemLimit: "1b"}), Chunks: struct {
							Total    uint64 `json:"total"`
							Up       uint64 `json:"up"`
							Down     uint64 `json:"down"`
							Busy     uint64 `json:"busy"`
							BusySize string `json:"busy_size"`
						}(struct {
							Total    uint64
							Up       uint64
							Down     uint64
							Busy     uint64
							BusySize string
						}{Total: 1, Up: 1, Down: 2, Busy: 3, BusySize: "1b"})},
				},
			},
		},
	}
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			fd := &Forwarder{nowFunc: func() time.Time { return now }}
			got, err := fd.fluentBitMetricsToCMetrics(&tc.metrics, &tc.storageMetrics)
			if err != nil {
				t.Error(err)
				return
			}

			ctx, err := cmetrics.NewContextFromMsgPack(got, 0)
			if err != nil {
				t.Error(err)
				return
			}

			want, err := ctx.EncodeMsgPack()
			if err != nil {
				t.Error(err)
				return
			}

			if !bytes.Equal(want, got) {
				t.Errorf("fluentBitMetricsToCMetrics() = %v, want %v", got, want)
			}
		})
	}
}
