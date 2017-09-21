package zipkinHelper

import (
	zipkin "github.com/openzipkin/zipkin-go-opentracing"
	"fmt"
	"os"
	"github.com/opentracing/opentracing-go"
	"net/http"
	"github.com/openzipkin/zipkin-go-opentracing/examples/middleware"
	"github.com/opentracing/opentracing-go/ext"
)
//--------------------zipkin----------------------

type Host struct {
	// Our service name.
	ServiceName string// = "svc1"
	// Host + port of our service.
	HostPort string// = "127.0.0.1:8001"
	// Endpoint to send Zipkin spans to.
	ZipkinHTTPEndpoint  string//= "http://172.16.0.80:9411/api/v1/spans"
	// Debug mode.
	Debug bool// = false
	// Base endpoint of our SVC2 service.
	Svc2Endpoint string//= "http://localhost:61002"
	// same span can be set to true for RPC style spans (Zipkin V1) vs Node style (OpenTracing)
	SameSpan bool//= true
	// make Tracer generate 128 bit traceID's for root spans.
	TraceID128Bit bool//= true
	//has init?
	hasInit bool
	//jwt host
	JwtHost string
}
func (h *Host)InitTrace(){
	if h.hasInit {
		return
	}
	// create collector.
	collector, err := zipkin.NewHTTPCollector(h.ZipkinHTTPEndpoint)
	if err != nil {
		fmt.Printf("unable to create Zipkin HTTP collector: %+v\n", err)
		os.Exit(-1)
	}
	// create recorder.
	recorder := zipkin.NewRecorder(collector, h.Debug, h.HostPort, h.ServiceName)
	// create tracer.
	tracer, err := zipkin.NewTracer(
		recorder,
		zipkin.ClientServerSameSpan(h.SameSpan),
		zipkin.TraceID128Bit(h.TraceID128Bit),
	)
	if err != nil {
		fmt.Printf("unable to create Zipkin tracer: %+v\n", err)
		os.Exit(-1)
	}
	// explicitly set our tracer to be the default tracer.
	opentracing.InitGlobalTracer(tracer)
	h.hasInit = true
}

func  (h *Host)Warp ( funName string , fun func(w http.ResponseWriter, req *http.Request)) http.Handler {
	var sumHandler http.Handler
	sumHandler = http.HandlerFunc(fun)
	// Wrap the Sum handler with our tracing middleware.
	sumHandler = middleware.FromHTTPRequest(h.GetTrace(), funName)(sumHandler)
	return sumHandler
}
func  (h *Host)GetTrace()(opentracing.Tracer){
	return opentracing.GlobalTracer()
}

func (h *Host)OuterCall(span opentracing.Span,out *Outer,i interface{}) interface{}{
	resourceSpan := opentracing.StartSpan(
		out.OpName,
		opentracing.ChildOf(span.Context()),
	)
	defer resourceSpan.Finish()
	// mark span as resource type
	ext.SpanKind.Set(resourceSpan, ext.SpanKindEnum("resource"))
	// name of the resource we try to reach
	ext.PeerService.Set(resourceSpan, out.PeerService)
	// hostname of the resource
	ext.PeerHostname.Set(resourceSpan, out.PeerHostname)
	// port of the resource
	ext.PeerPort.Set(resourceSpan, out.PeerPort)
	// let's binary annotate the query we run
	resourceSpan.SetTag(
		out.TagKey, out.TagValue,
	)
	return i.(func()interface{})()
}
func (h *Host)GetSpan(r *http.Request) opentracing.Span {
	return opentracing.SpanFromContext(r.Context())
}
func (h *Host)SetTag(span opentracing.Span,key string,value string)  {
	// Example binary annotations.
	span.SetTag(key, value)
}
func (h *Host)SetAnnotation(span opentracing.Span,anno string)  {
	span.LogEvent(anno)
}
type Outer struct {
	OpName string
	PeerService string
	PeerHostname string
	PeerPort uint16
	TagKey string
	TagValue string
}

