package errmodel

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"go.opentelemetry.io/otel/trace"
)

// Category values for compact errors.
const (
	CategoryValidation = "validation"
	CategoryTool       = "tool"
	CategoryNetwork    = "network"
	CategoryModel      = "model"
	CategoryPolicy     = "policy"
	CategorySystem     = "system"
)

// Error is the compact error payload returned by APIs and used internally.
// It implements the error interface.
type Error struct {
	Category string         `json:"category"`
	Code     string         `json:"code"`
	Message  string         `json:"message"`
	Context  map[string]any `json:"context,omitempty"`
	Causes   []Error        `json:"causes,omitempty"`
}

func (e *Error) Error() string {
	if e == nil {
		return ""
	}
	if e.Code != "" {
		return e.Code + ": " + e.Message
	}
	return e.Message
}

// New constructs a new compact error.
func New(category, code, message string, ctx map[string]any, causes ...error) *Error {
	ce := &Error{Category: category, Code: code, Message: truncate(message, 512)}
	if len(ctx) > 0 {
		ce.Context = truncateContext(ctx)
	}
	for _, c := range causes {
		if c == nil {
			continue
		}
		ce.Causes = append(ce.Causes, *From(c))
	}
	return ce
}

// From converts any error into a compact Error. If err is already *Error, it's returned as-is.
func From(err error) *Error {
	var ce *Error
	if err == nil {
		return nil
	}
	if errors.As(err, &ce) {
		return ce
	}
	// Default to system/internal for unknown error types.
	return &Error{Category: CategorySystem, Code: "internal", Message: truncate(err.Error(), 512)}
}

// Convenience constructors.
func Validation(code, message string, ctx map[string]any) *Error {
	return New(CategoryValidation, code, message, ctx)
}

func Policy(code, message string, ctx map[string]any) *Error {
	return New(CategoryPolicy, code, message, ctx)
}

func System(code, message string, ctx map[string]any, cause error) *Error {
	if cause != nil {
		return New(CategorySystem, code, message, ctx, cause)
	}
	return New(CategorySystem, code, message, ctx)
}

// HTTPStatus maps category/code to HTTP status.
func HTTPStatus(e *Error) int {
	if e == nil {
		return http.StatusInternalServerError
	}
	switch e.Category {
	case CategoryValidation:
		// Special-case common codes
		switch e.Code {
		case "not_found":
			return http.StatusNotFound
		case "conflict":
			return http.StatusConflict
		default:
			return http.StatusBadRequest
		}
	case CategoryPolicy:
		switch e.Code {
		case "unauthorized":
			return http.StatusUnauthorized
		case "forbidden":
			return http.StatusForbidden
		case "method_not_allowed":
			return http.StatusMethodNotAllowed
		default:
			return http.StatusForbidden
		}
	case CategoryNetwork:
		return http.StatusBadGateway
	case CategoryTool, CategoryModel:
		return http.StatusBadGateway
	case CategorySystem:
		fallthrough
	default:
		return http.StatusInternalServerError
	}
}

// WriteHTTP writes a compact error envelope to the response writer.
// It attempts to include the trace_id if present in ctx.
func WriteHTTP(w http.ResponseWriter, r *http.Request, err error) {
	ce := From(err)
	if ce == nil {
		ce = &Error{Category: CategorySystem, Code: "internal", Message: "unknown error"}
	}
	status := HTTPStatus(ce)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	traceID := ""
	if r != nil {
		if span := trace.SpanFromContext(r.Context()); span != nil {
			sc := span.SpanContext()
			if sc.HasTraceID() {
				traceID = sc.TraceID().String()
			}
		}
	}
	// Envelope { error: Error, trace_id?: string }
	_ = json.NewEncoder(w).Encode(map[string]any{
		"error":    ce,
		"trace_id": traceID,
	})
}

// truncate trims a string to max characters.
func truncate(s string, max int) string {
	if max <= 0 || len(s) <= max {
		return s
	}
	if max <= 3 {
		return s[:max]
	}
	return s[:max-3] + "..."
}

// truncateContext trims long string values in the context map.
func truncateContext(ctx map[string]any) map[string]any {
	out := make(map[string]any, len(ctx))
	for k, v := range ctx {
		switch t := v.(type) {
		case string:
			out[k] = truncate(t, 256)
		default:
			// Try to stringify primitive slices to keep payload compact.
			b, err := json.Marshal(t)
			if err == nil && len(b) > 0 {
				// Avoid giant blobs; keep a preview
				s := string(b)
				if len(s) > 256 {
					s = truncate(s, 256)
				}
				out[k] = s
			} else {
				out[k] = t
			}
		}
	}
	return out
}

// IsCategory checks if err belongs to a specific category.
func IsCategory(err error, category string) bool {
	ce := From(err)
	return ce != nil && strings.EqualFold(ce.Category, category)
}
