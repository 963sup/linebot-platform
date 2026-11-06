package bootstrap

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/963sup/linebot-platform/internal/adapter/inbound/cli"
)

type closeableLINEOpsRuntime interface {
	Close() error
}

type lineRuntimeConstructorCounts struct {
	messaging int
	insight   int
	blob      int
	audience  int
}

func countingLineRuntimeFactories(counts *lineRuntimeConstructorCounts) lineRuntimeFactories {
	return lineRuntimeFactories{
		messaging: func(string) (*LINEOpsRuntime, error) {
			counts.messaging++
			return &LINEOpsRuntime{}, nil
		},
		insight: func(string) (*LINEOpsRuntime, error) {
			counts.insight++
			return &LINEOpsRuntime{}, nil
		},
		blob: func(string) (*LINEOpsRuntime, error) {
			counts.blob++
			return &LINEOpsRuntime{}, nil
		},
		audience: func(string) (*LINEOpsRuntime, error) {
			counts.audience++
			return &LINEOpsRuntime{}, nil
		},
	}
}

func TestBuildOpsCLIProvidesEveryRuntimeFactory(t *testing.T) {
	dependencies := reflect.ValueOf(BuildOpsCLI())
	dependencyType := dependencies.Type()
	for i := 0; i < dependencies.NumField(); i++ {
		if dependencies.Field(i).IsNil() {
			t.Errorf("runtime factory %s is nil", dependencyType.Field(i).Name)
		}
	}
}

func TestRunOpsPropagatesCanceledRootContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := RunOps(ctx, []string{"push"}, func(string) string { return "" }); !errors.Is(err, context.Canceled) {
		t.Fatalf("RunOps canceled context error = %v", err)
	}
}

func TestBuildOpsCLIConstructsOnlyRequiredLINEClientFamily(t *testing.T) {
	tests := []struct {
		name string
		open func(cli.Dependencies) (closeableLINEOpsRuntime, error)
		want lineRuntimeConstructorCounts
	}{
		{
			name: "push",
			open: func(dependencies cli.Dependencies) (closeableLINEOpsRuntime, error) {
				return dependencies.PushRuntime("test-token")
			},
			want: lineRuntimeConstructorCounts{messaging: 1},
		},
		{
			name: "insights",
			open: func(dependencies cli.Dependencies) (closeableLINEOpsRuntime, error) {
				return dependencies.InsightsRuntime("test-token")
			},
			want: lineRuntimeConstructorCounts{insight: 1},
		},
		{
			name: "content",
			open: func(dependencies cli.Dependencies) (closeableLINEOpsRuntime, error) {
				return dependencies.ContentRuntime("test-token")
			},
			want: lineRuntimeConstructorCounts{blob: 1},
		},
		{
			name: "audience",
			open: func(dependencies cli.Dependencies) (closeableLINEOpsRuntime, error) {
				return dependencies.AudienceRuntime("test-token")
			},
			want: lineRuntimeConstructorCounts{audience: 1},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			counts := lineRuntimeConstructorCounts{}
			dependencies := buildOpsCLI(countingLineRuntimeFactories(&counts))
			runtime, err := test.open(dependencies)
			if err != nil {
				t.Fatal(err)
			}
			if err := runtime.Close(); err != nil {
				t.Fatal(err)
			}
			if counts != test.want {
				t.Fatalf("constructor counts = %+v, want %+v", counts, test.want)
			}
		})
	}
}

func BenchmarkBuildOpsCLIFamilyConstruction(b *testing.B) {
	dependencies := BuildOpsCLI()
	benchmarks := []struct {
		name string
		open func() (closeableLINEOpsRuntime, error)
	}{
		{name: "push", open: func() (closeableLINEOpsRuntime, error) { return dependencies.PushRuntime("test-token") }},
		{name: "insights", open: func() (closeableLINEOpsRuntime, error) { return dependencies.InsightsRuntime("test-token") }},
		{name: "content", open: func() (closeableLINEOpsRuntime, error) { return dependencies.ContentRuntime("test-token") }},
		{name: "audience", open: func() (closeableLINEOpsRuntime, error) { return dependencies.AudienceRuntime("test-token") }},
	}
	for _, benchmark := range benchmarks {
		b.Run(benchmark.name, func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				runtime, err := benchmark.open()
				if err != nil {
					b.Fatal(err)
				}
				if err := runtime.Close(); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
