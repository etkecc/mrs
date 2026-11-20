package apm

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"slices"
	"strings"

	"github.com/getsentry/sentry-go"
)

// handlerOp is a recorded WithAttrs or WithGroup call, replayed to render a record
type handlerOp struct {
	group string
	attrs []slog.Attr
}

// reportHandler passes records to next and reports error-level records to sentry and healthchecks
type reportHandler struct {
	next slog.Handler
	hub  *sentry.Hub
	ops  []handlerOp
}

// Enabled reports whether the wrapped handler handles level
func (h *reportHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.next.Enabled(ctx, level)
}

//nolint:gocritic // slog.Handler requires records by value
func (h *reportHandler) Handle(ctx context.Context, r slog.Record) error {
	if r.Level >= slog.LevelError {
		h.report(ctx, &r)
	} else if h.hub != nil {
		h.breadcrumb(ctx, &r)
	}
	return h.next.Handle(ctx, r)
}

// WithAttrs returns a handler with attrs attached to the console and rendered output
func (h *reportHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	if len(attrs) == 0 {
		return h
	}
	return h.withOp(handlerOp{attrs: attrs})
}

// WithGroup returns a handler with name attached to the console and rendered output
func (h *reportHandler) WithGroup(name string) slog.Handler {
	if name == "" {
		return h
	}
	return h.withOp(handlerOp{group: name})
}

// withOp records op and applies it to the console handler
func (h *reportHandler) withOp(op handlerOp) *reportHandler {
	copied := *h
	copied.ops = append(slices.Clone(h.ops), op)
	if op.group != "" {
		copied.next = h.next.WithGroup(op.group)
	} else {
		copied.next = h.next.WithAttrs(op.attrs)
	}
	return &copied
}

// report sends an error-level record to sentry and healthchecks
func (h *reportHandler) report(ctx context.Context, r *slog.Record) {
	if h.hub == nil && hc == nil {
		return
	}
	fields, err := h.fieldMap(ctx, r)
	if err != nil {
		fields = nil
	}
	if h.hub != nil {
		event := sentry.NewEvent()
		event.Timestamp = r.Time
		event.Level = sentryLevel(r.Level)
		event.Logger = "slog"
		event.Message = r.Message
		applyEventFields(event, fields)
		h.hub.CaptureEvent(event)
	}
	if hc == nil || r.Message == "" {
		return
	}
	body := r.Message
	if indented, err := json.MarshalIndent(fields, "", "  "); err == nil {
		body += "\n" + string(indented)
	}
	HealthcheckFail(strings.NewReader(body + "\n" + GetStackTrace()))
}

// breadcrumb records a below-error record on the sentry scope
func (h *reportHandler) breadcrumb(ctx context.Context, r *slog.Record) {
	fields, err := h.fieldMap(ctx, r)
	if err != nil {
		fields = nil
	}
	h.hub.AddBreadcrumb(&sentry.Breadcrumb{
		Message:   r.Message,
		Level:     sentryLevel(r.Level),
		Data:      withoutRecordKeys(fields),
		Timestamp: r.Time,
	}, nil)
}

// fieldMap renders the record, including handler attrs and groups, as a map
func (h *reportHandler) fieldMap(ctx context.Context, r *slog.Record) (map[string]any, error) {
	buf := bytes.Buffer{}
	var handler slog.Handler = slog.NewJSONHandler(&buf, &slog.HandlerOptions{AddSource: true, ReplaceAttr: dropTime})
	for _, op := range h.ops {
		if op.group != "" {
			handler = handler.WithGroup(op.group)
		} else {
			handler = handler.WithAttrs(normalizeAttrs(op.attrs))
		}
	}
	record := slog.NewRecord(r.Time, r.Level, r.Message, r.PC)
	r.Attrs(func(attr slog.Attr) bool {
		record.AddAttrs(normalizeAttr(attr))
		return true
	})
	if err := handler.Handle(ctx, record); err != nil {
		return nil, err
	}
	fields := map[string]any{}
	decoder := json.NewDecoder(&buf)
	decoder.UseNumber()
	if err := decoder.Decode(&fields); err != nil {
		return nil, err
	}
	return fields, nil
}

// normalizeAttrs renders error values as their message
func normalizeAttrs(attrs []slog.Attr) []slog.Attr {
	normalized := make([]slog.Attr, len(attrs))
	for i, attr := range attrs {
		normalized[i] = normalizeAttr(attr)
	}
	return normalized
}

// normalizeAttr renders an error value as its message, recursing into groups
func normalizeAttr(attr slog.Attr) slog.Attr {
	value := attr.Value.Resolve()
	switch value.Kind() {
	case slog.KindAny:
		if err, ok := value.Any().(error); ok {
			attr.Value = slog.StringValue(err.Error())
		}
	case slog.KindGroup:
		attr.Value = slog.GroupValue(normalizeAttrs(value.Group())...)
	default: // scalar values render as-is
	}
	return attr
}

// withoutRecordKeys drops the keys already carried by msg and level from fields
func withoutRecordKeys(fields map[string]any) map[string]any {
	if fields == nil {
		return nil
	}
	delete(fields, slog.MessageKey)
	delete(fields, slog.LevelKey)
	return fields
}
