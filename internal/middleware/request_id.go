package middleware

import (
	"context"
	"net/http"

	"github.com/google/uuid"
)

const requestIDHeader = "X-Request-Id"

type requestMetadataKey struct{}

type requestMetadata struct {
	requestID string
	userID    string
}

func requestMetadataFromContext(ctx context.Context) *requestMetadata {
	metadata, _ := ctx.Value(requestMetadataKey{}).(*requestMetadata)
	return metadata
}

func ensureRequestMetadata(r *http.Request) (*http.Request, *requestMetadata) {
	if metadata := requestMetadataFromContext(r.Context()); metadata != nil && metadata.requestID != "" {
		return r, metadata
	}
	metadata := &requestMetadata{requestID: uuid.NewString()}
	ctx := context.WithValue(r.Context(), requestMetadataKey{}, metadata)
	return r.WithContext(ctx), metadata
}

// RequestIDFromContext returns the server-owned request ID, if present.
func RequestIDFromContext(ctx context.Context) string {
	if metadata := requestMetadataFromContext(ctx); metadata != nil {
		return metadata.requestID
	}
	return ""
}

// RequestID creates and publishes one server-owned request ID per request.
func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r, metadata := ensureRequestMetadata(r)
		w.Header().Set(requestIDHeader, metadata.requestID)
		next.ServeHTTP(w, r)
	})
}
