package http

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"go.uber.org/yarpc/api/transport"
)

// ---------------------------------------------------------------------------
// Benchmark scenarios
//
// We compare the legacy path (Header.Set/Add with canonicalization) against
// the direct-map-write path introduced in this PoC.
//
// Three user-header profiles match the request from the conversation:
//   - many headers, all lowercase  (best case for canonicalization — already canonical)
//   - many headers, mixed case     (worst case — every key hits CanonicalMIMEHeaderKey)
//   - few headers, mixed case      (typical YARPC workload)
//
// "many" = 24 (realistic upper bound); "few" = 4.
// Header values vary in length to avoid unrealistic constant-folding.
// ---------------------------------------------------------------------------

type headerBenchScenario struct {
	name    string
	headers transport.Headers
}

func benchScenarios() []headerBenchScenario {
	return []headerBenchScenario{
		{
			name:    "many_headers_all_lowercase",
			headers: makeBenchHeaders(24, lowercaseOnly),
		},
		{
			name:    "many_headers_mixed_case",
			headers: makeBenchHeaders(24, mixedCase),
		},
		{
			name:    "few_headers_mixed_case",
			headers: makeBenchHeaders(4, mixedCase),
		},
	}
}

type caseStyle int

const (
	lowercaseOnly caseStyle = iota
	mixedCase
)

func makeBenchHeaders(n int, style caseStyle) transport.Headers {
	h := transport.NewHeadersWithCapacity(n)
	for i := 0; i < n; i++ {
		var key string
		switch style {
		case lowercaseOnly:
			key = fmt.Sprintf("x-custom-header-%03d", i)
		case mixedCase:
			if i%3 == 0 {
				key = fmt.Sprintf("X-Custom-Header-%03d", i)
			} else if i%3 == 1 {
				key = fmt.Sprintf("x-custom-HeaDer-%03d", i)
			} else {
				key = fmt.Sprintf("x-custom-header-%03d", i)
			}
		}
		// Vary value length so the compiler can't constant-fold.
		h = h.With(key, fmt.Sprintf("value-%d-padding-abcdefgh", i))
	}
	return h
}

// benchOutbound returns an Outbound with two static extra headers,
// matching a realistic config (e.g. addHeaders in YAML).
func benchOutbound() *Outbound {
	return &Outbound{
		urlTemplate:       defaultURLTemplate,
		bothResponseError: true,
		headers: http.Header{
			"X-Static-Token": []string{"tok_abc123"},
			"X-Request-Id":   []string{"rid-00000000"},
		},
	}
}

func benchTransportRequest(headers transport.Headers) *transport.Request {
	return &transport.Request{
		Caller:          "bench-caller",
		Service:         "bench-service",
		Encoding:        "raw",
		Procedure:       "BenchProcedure",
		ShardKey:        "shard-42",
		RoutingKey:      "routing-key",
		RoutingDelegate: "delegate-svc",
		CallerProcedure: "CallerProc",
		Headers:         headers,
	}
}

// ---------------------------------------------------------------------------
// Legacy code path (inlined so we can A/B without git stash)
// ---------------------------------------------------------------------------

// buildLegacy reproduces the pre-PoC code path:
//   - ToHTTPHeaders iterates Items() (canonicalized) and calls http.Header.Add
//   - Core headers use http.Header.Set
func buildLegacy(o *Outbound, treq *transport.Request, ttl time.Duration) (*http.Request, error) {
	newURL := *o.urlTemplate
	hreq, err := http.NewRequest("POST", newURL.String(), treq.Body)
	if err != nil {
		return nil, err
	}

	headers := applicationHeaders.deleteHTTP2PseudoHeadersIfNeeded(treq.Headers)
	hreq.Header = applicationHeaders.ToHTTPHeaders(headers, nil)

	for k, vs := range o.headers {
		for _, v := range vs {
			hreq.Header.Add(k, v)
		}
	}

	hreq.Header.Set(CallerHeader, treq.Caller)
	hreq.Header.Set(ServiceHeader, treq.Service)
	hreq.Header.Set(ProcedureHeader, treq.Procedure)
	if ttl != 0 {
		hreq.Header.Set(TTLMSHeader, fmt.Sprintf("%d", ttl/time.Millisecond))
	}
	if treq.ShardKey != "" {
		hreq.Header.Set(ShardKeyHeader, treq.ShardKey)
	}
	if treq.RoutingKey != "" {
		hreq.Header.Set(RoutingKeyHeader, treq.RoutingKey)
	}
	if treq.RoutingDelegate != "" {
		hreq.Header.Set(RoutingDelegateHeader, treq.RoutingDelegate)
	}
	if treq.CallerProcedure != "" {
		hreq.Header.Set(CallerProcedureHeader, treq.CallerProcedure)
	}
	if treq.Encoding != "" {
		hreq.Header.Set(EncodingHeader, string(treq.Encoding))
	}
	if o.bothResponseError {
		hreq.Header.Set(AcceptsBothResponseErrorHeader, AcceptTrue)
	}

	return hreq, nil
}

// buildDirect uses the new PoC code path (createRequest + withCoreHeaders).
func buildDirect(o *Outbound, treq *transport.Request, ttl time.Duration) (*http.Request, error) {
	hreq, err := o.createRequest(treq)
	if err != nil {
		return nil, err
	}
	return o.withCoreHeaders(hreq, treq, ttl), nil
}

func toHTTPHeadersPreserveCaseCompat(from transport.Headers, to http.Header) http.Header {
	mapperWithPreserveCase, ok := any(applicationHeaders).(interface {
		ToHTTPHeadersPreserveCase(transport.Headers, http.Header) http.Header
	})
	if ok {
		return mapperWithPreserveCase.ToHTTPHeadersPreserveCase(from, to)
	}
	return applicationHeaders.ToHTTPHeaders(from, to)
}

// ---------------------------------------------------------------------------
// Benchmark: ToHTTPHeaders vs ToHTTPHeadersPreserveCase (isolated mapper)
//
// This is the tightest comparison — just the header-map construction cost
// without http.NewRequest or core-header overhead.
// ---------------------------------------------------------------------------

func BenchmarkHeaderMapper(b *testing.B) {
	for _, tc := range benchScenarios() {
		b.Run(tc.name, func(b *testing.B) {
			b.Run("legacy_ToHTTPHeaders", func(b *testing.B) {
				h := tc.headers
				b.ReportAllocs()
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					out := applicationHeaders.ToHTTPHeaders(h, nil)
					if len(out) == 0 {
						b.Fatal("empty")
					}
				}
			})

			b.Run("direct_ToHTTPHeadersPreserveCase", func(b *testing.B) {
				h := tc.headers
				b.ReportAllocs()
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					out := toHTTPHeadersPreserveCaseCompat(h, nil)
					if len(out) == 0 {
						b.Fatal("empty")
					}
				}
			})
		})
	}
}

// ---------------------------------------------------------------------------
// Benchmark: full request build (createRequest + withCoreHeaders)
//
// Measures end-to-end CPU and alloc cost of constructing the outbound
// http.Request, which is the per-call hot path.
// ---------------------------------------------------------------------------

func BenchmarkOutboundBuildRequest(b *testing.B) {
	const benchTTL = 500 * time.Millisecond

	for _, tc := range benchScenarios() {
		b.Run(tc.name, func(b *testing.B) {
			b.Run("legacy", func(b *testing.B) {
				o := benchOutbound()
				treq := benchTransportRequest(tc.headers)
				b.ReportAllocs()
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					hreq, err := buildLegacy(o, treq, benchTTL)
					if err != nil {
						b.Fatal(err)
					}
					if len(hreq.Header) == 0 {
						b.Fatal("expected headers")
					}
				}
			})

			b.Run("direct", func(b *testing.B) {
				o := benchOutbound()
				treq := benchTransportRequest(tc.headers)
				b.ReportAllocs()
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					hreq, err := buildDirect(o, treq, benchTTL)
					if err != nil {
						b.Fatal(err)
					}
					if len(hreq.Header) == 0 {
						b.Fatal("expected headers")
					}
				}
			})
		})
	}
}
