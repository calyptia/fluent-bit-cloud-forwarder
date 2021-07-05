package forwarder

import (
	"github.com/calyptia/cmetrics-go"
	"reflect"
	"testing"
	"time"

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
			encoded, _ := fd.fluentBitMetricsToCMetrics(&tc.args)
			ctx, _ := cmetrics.NewContextFromMsgPack(encoded)
			reEncoded, _ := ctx.EncodeMsgPack()
			if !reflect.DeepEqual(encoded, reEncoded) {
				t.Errorf("fluentBitMetricsToCMetrics() = %+v, want %+v", string(encoded), string(reEncoded))
			}
		})
	}
}
