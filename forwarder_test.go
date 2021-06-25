package forwarder

import (
	"reflect"
	"testing"
	"time"

	"github.com/calyptia/fluent-bit-cloud-forwarder/cloud"
	fluentbit "github.com/calyptia/go-fluent-bit-metrics"
)

func Test_fluentBitMetricsToCloudMetrics(t *testing.T) {
	now := time.Now().Truncate(time.Nanosecond)
	ts := now.UnixNano()

	tt := []struct {
		name string
		args fluentbit.Metrics
		want cloud.AddMetricsInput
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
			want: cloud.AddMetricsInput{
				Metrics: []cloud.AddMetricInput{
					{
						Type: cloud.MetricTypeCounter,
						Options: cloud.MetricOpts{
							Namespace: "input",
							Subsystem: "testinput.0",
							Name:      "records",
							FQName:    "input.testinput.0.records",
						},
						Values: []cloud.MetricValue{
							{
								Timestamp: ts,
								Value:     10,
							},
						},
					},
					{
						Type: cloud.MetricTypeCounter,
						Options: cloud.MetricOpts{
							Namespace: "input",
							Subsystem: "testinput.0",
							Name:      "bytes",
							FQName:    "input.testinput.0.bytes",
						},
						Values: []cloud.MetricValue{
							{
								Timestamp: ts,
								Value:     12,
							},
						},
					},
					{
						Type: cloud.MetricTypeCounter,
						Options: cloud.MetricOpts{
							Namespace: "output",
							Subsystem: "testoutput.0",
							Name:      "proc_records",
							FQName:    "output.testoutput.0.proc_records",
						},
						Values: []cloud.MetricValue{
							{
								Timestamp: ts,
								Value:     2,
							},
						},
					},
					{
						Type: cloud.MetricTypeCounter,
						Options: cloud.MetricOpts{
							Namespace: "output",
							Subsystem: "testoutput.0",
							Name:      "proc_bytes",
							FQName:    "output.testoutput.0.proc_bytes",
						},
						Values: []cloud.MetricValue{
							{
								Timestamp: ts,
								Value:     100,
							},
						},
					},
					{
						Type: cloud.MetricTypeCounter,
						Options: cloud.MetricOpts{
							Namespace: "output",
							Subsystem: "testoutput.0",
							Name:      "errors",
							FQName:    "output.testoutput.0.errors",
						},
						Values: []cloud.MetricValue{
							{
								Timestamp: ts,
								Value:     1,
							},
						},
					},
					{
						Type: cloud.MetricTypeCounter,
						Options: cloud.MetricOpts{
							Namespace: "output",
							Subsystem: "testoutput.0",
							Name:      "retries",
							FQName:    "output.testoutput.0.retries",
						},
						Values: []cloud.MetricValue{
							{
								Timestamp: ts,
								Value:     5,
							},
						},
					},
					{
						Type: cloud.MetricTypeCounter,
						Options: cloud.MetricOpts{
							Namespace: "output",
							Subsystem: "testoutput.0",
							Name:      "retries_failed",
							FQName:    "output.testoutput.0.retries_failed",
						},
						Values: []cloud.MetricValue{
							{
								Timestamp: ts,
								Value:     4,
							},
						},
					},
				},
			},
		},
	}
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			fd := &Forwarder{nowFunc: func() time.Time { return now }}
			if got := fd.fluentBitMetricsToCloudMetrics(tc.args); !reflect.DeepEqual(got, tc.want) {
				t.Errorf("fluentBitMetricsToCloudMetrics() = %+v, want %+v", got, tc.want)
			}
		})
	}
}
