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
		name string
		args fluentbit.Metrics
	}{
		{
			args: fluentbit.Metrics{
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
		},
	}
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			fd := &Forwarder{nowFunc: func() time.Time { return now }}
			got, err := fd.fluentBitMetricsToCMetrics(&tc.args)
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
