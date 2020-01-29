// +build !wasm

package tracing

import (
	"io"
	"log"

	"github.com/opentracing/opentracing-go"
	jaegercfg "github.com/uber/jaeger-client-go/config"
	"go.elastic.co/apm/module/apmot"
)

var Enabled bool

var jaegerCloser io.Closer

func StartElastic() {
	Enabled = true
	opentracing.SetGlobalTracer(apmot.New())
}

func StopJaeger() {
	Enabled = false
	jaegerCloser.Close()
}

func StartJaeger(serviceName string) {
	Enabled = true
	cfg, err := jaegercfg.FromEnv()
	if err != nil {
		// parsing errors might happen here, such as when we get a string where we expect a number
		log.Printf("Could not parse Jaeger env vars: %s", err.Error())
		return
	}

	tracer, closer, err := cfg.NewTracer()
	if err != nil {
		log.Printf("Could not initialize jaeger tracer: %s", err.Error())
		return
	}
	jaegerCloser = closer

	opentracing.SetGlobalTracer(tracer)
	jaegerCloser = closer
}
