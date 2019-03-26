package tracing

import (
	"io"
	"log"

	"go.elastic.co/apm"

	"github.com/opentracing/opentracing-go"
	"github.com/uber/jaeger-client-go"
	jaegercfg "github.com/uber/jaeger-client-go/config"
	jaegerlog "github.com/uber/jaeger-client-go/log"
	"github.com/uber/jaeger-lib/metrics"
	"go.elastic.co/apm/module/apmot"
)

var Enabled bool

var jaegerCloser io.Closer
var elasticTracer *apm.Tracer

func StartElastic(serviceName string, version string) {
	Enabled = true
	tracer, err := apm.NewTracer(serviceName, version)
	if err != nil {
		panic(err)
	}
	elasticTracer = tracer
	opentracing.SetGlobalTracer(apmot.New(apmot.WithTracer(tracer)))
}

func StopElastic() {
	Enabled = false
	elasticTracer.Close()
}

func StopJaeger() {
	Enabled = false
	jaegerCloser.Close()
}

func StartJaeger(serviceName string, version string) {
	Enabled = true

	// Sample configuration for testing. Use constant sampling to sample every trace
	// and enable LogSpan to log every span via configured Logger.
	cfg := jaegercfg.Configuration{
		Sampler: &jaegercfg.SamplerConfig{
			Type:  jaeger.SamplerTypeConst,
			Param: 1,
		},
		Reporter: &jaegercfg.ReporterConfig{
			LogSpans: false,
		},
	}

	// Example logger and metrics factory. Use github.com/uber/jaeger-client-go/log
	// and github.com/uber/jaeger-lib/metrics respectively to bind to real logging and metrics
	// frameworks.
	jLogger := jaegerlog.StdLogger
	jMetricsFactory := metrics.NullFactory

	// Initialize tracer with a logger and a metrics factory
	closer, err := cfg.InitGlobalTracer(
		serviceName,
		jaegercfg.Logger(jLogger),
		jaegercfg.Metrics(jMetricsFactory),
		jaegercfg.Tag("version", version),
	)
	if err != nil {
		log.Printf("Could not initialize jaeger tracer: %s", err.Error())
		return
	}
	jaegerCloser = closer
}
