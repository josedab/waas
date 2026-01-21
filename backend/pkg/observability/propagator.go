package observability

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"regexp"
	"strings"
)

const (
	traceparentHeader = "traceparent"
	tracestateHeader  = "tracestate"
	
	traceparentVersion = "00"
	traceFlagSampled   = byte(0x01)
)

var traceparentRegex = regexp.MustCompile(`^([0-9a-f]{2})-([0-9a-f]{32})-([0-9a-f]{16})-([0-9a-f]{2})$`)

// TraceContextPropagator handles W3C Trace Context propagation
type TraceContextPropagator struct{}

// NewTraceContextPropagator creates a new propagator
func NewTraceContextPropagator() *TraceContextPropagator {
	return &TraceContextPropagator{}
}

// Extract extracts trace context from headers
func (p *TraceContextPropagator) Extract(headers map[string]string) (*TraceContext, error) {
	traceparent := ""
	tracestate := ""
	
	for k, v := range headers {
		switch strings.ToLower(k) {
		case traceparentHeader:
			traceparent = v
		case tracestateHeader:
			tracestate = v
		}
	}
	
	if traceparent == "" {
		return nil, nil
	}
	
	return p.parseTraceparent(traceparent, tracestate)
}

// Inject injects trace context into headers
func (p *TraceContextPropagator) Inject(ctx *TraceContext, headers map[string]string) {
	if ctx == nil {
		return
	}
	
	traceparent := fmt.Sprintf("%s-%s-%s-%02x",
		traceparentVersion,
		ctx.TraceID,
		ctx.SpanID,
		ctx.TraceFlags,
	)
	
	headers[traceparentHeader] = traceparent
	
	if ctx.TraceState != "" {
		headers[tracestateHeader] = ctx.TraceState
	}
}

// NewTraceContext creates a new trace context with generated IDs
func (p *TraceContextPropagator) NewTraceContext() (*TraceContext, error) {
	traceID, err := generateTraceID()
	if err != nil {
		return nil, err
	}
	
	spanID, err := generateSpanID()
	if err != nil {
		return nil, err
	}
	
	return &TraceContext{
		TraceID:    traceID,
		SpanID:     spanID,
		TraceFlags: traceFlagSampled,
	}, nil
}

// NewChildContext creates a child context from a parent
func (p *TraceContextPropagator) NewChildContext(parent *TraceContext) (*TraceContext, error) {
	spanID, err := generateSpanID()
	if err != nil {
		return nil, err
	}
	
	return &TraceContext{
		TraceID:      parent.TraceID,
		SpanID:       spanID,
		ParentSpanID: parent.SpanID,
		TraceFlags:   parent.TraceFlags,
		TraceState:   parent.TraceState,
	}, nil
}

func (p *TraceContextPropagator) parseTraceparent(traceparent, tracestate string) (*TraceContext, error) {
	matches := traceparentRegex.FindStringSubmatch(traceparent)
	if matches == nil {
		return nil, fmt.Errorf("invalid traceparent format")
	}
	
	version := matches[1]
	traceID := matches[2]
	spanID := matches[3]
	flagsHex := matches[4]
	
	// Check for all-zero trace ID or span ID
	if traceID == strings.Repeat("0", 32) {
		return nil, fmt.Errorf("trace ID cannot be all zeros")
	}
	if spanID == strings.Repeat("0", 16) {
		return nil, fmt.Errorf("span ID cannot be all zeros")
	}
	
	flags, err := hex.DecodeString(flagsHex)
	if err != nil {
		return nil, fmt.Errorf("invalid trace flags: %w", err)
	}
	
	ctx := &TraceContext{
		TraceID:    traceID,
		SpanID:     spanID,
		TraceFlags: flags[0],
		TraceState: tracestate,
	}
	
	// Handle unknown versions by accepting future versions
	if version != traceparentVersion {
		// Future versions - accept but don't process additional fields
	}
	
	return ctx, nil
}

// IsSampled returns true if the trace is sampled
func (tc *TraceContext) IsSampled() bool {
	return tc.TraceFlags&traceFlagSampled != 0
}

// SetSampled sets the sampled flag
func (tc *TraceContext) SetSampled(sampled bool) {
	if sampled {
		tc.TraceFlags |= traceFlagSampled
	} else {
		tc.TraceFlags &^= traceFlagSampled
	}
}

func generateTraceID() (string, error) {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

func generateSpanID() (string, error) {
	bytes := make([]byte, 8)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// ContextKey type for context values
type contextKey string

const traceContextKey contextKey = "trace_context"

// WithTraceContext adds trace context to a Go context
func WithTraceContext(ctx context.Context, tc *TraceContext) context.Context {
	return context.WithValue(ctx, traceContextKey, tc)
}

// TraceContextFromContext retrieves trace context from a Go context
func TraceContextFromContext(ctx context.Context) *TraceContext {
	tc, ok := ctx.Value(traceContextKey).(*TraceContext)
	if !ok {
		return nil
	}
	return tc
}
