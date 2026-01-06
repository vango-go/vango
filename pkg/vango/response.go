package vango

import (
	"encoding/json"
	"net/http"
)

// =============================================================================
// Response[T] - Typed API Response Wrapper
// =============================================================================

// Response wraps an API response with optional metadata.
// It provides a standard structure for JSON API responses.
//
// Example usage in API handlers:
//
//	func GET(ctx vango.Ctx) (*vango.Response[[]Project], error) {
//	    projects, err := db.Projects.All()
//	    if err != nil {
//	        return nil, vango.InternalError(err)
//	    }
//	    return vango.OK(projects), nil
//	}
//
//	func POST(ctx vango.Ctx, input CreateProjectInput) (*vango.Response[*Project], error) {
//	    project, err := db.Projects.Create(input)
//	    if err != nil {
//	        return nil, vango.BadRequest(err)
//	    }
//	    return vango.Created(project), nil
//	}
type Response[T any] struct {
	// Data is the response payload.
	Data T `json:"data,omitempty"`

	// Meta contains optional metadata (pagination, counts, etc.)
	Meta map[string]any `json:"meta,omitempty"`

	// StatusCode is the HTTP status code for this response.
	// Not included in JSON output.
	StatusCode int `json:"-"`
}

// OK creates a 200 OK response with the given data.
func OK[T any](data T) *Response[T] {
	return &Response[T]{
		Data:       data,
		StatusCode: http.StatusOK,
	}
}

// Created creates a 201 Created response with the given data.
func Created[T any](data T) *Response[T] {
	return &Response[T]{
		Data:       data,
		StatusCode: http.StatusCreated,
	}
}

// Accepted creates a 202 Accepted response with the given data.
func Accepted[T any](data T) *Response[T] {
	return &Response[T]{
		Data:       data,
		StatusCode: http.StatusAccepted,
	}
}

// NoContent creates a 204 No Content response.
// Typically used for successful DELETE operations.
func NoContent[T any]() *Response[T] {
	return &Response[T]{
		StatusCode: http.StatusNoContent,
	}
}

// WithMeta adds metadata to the response.
func (r *Response[T]) WithMeta(key string, value any) *Response[T] {
	if r.Meta == nil {
		r.Meta = make(map[string]any)
	}
	r.Meta[key] = value
	return r
}

// WithPagination adds pagination metadata to the response.
func (r *Response[T]) WithPagination(page, perPage, total int) *Response[T] {
	if r.Meta == nil {
		r.Meta = make(map[string]any)
	}
	r.Meta["page"] = page
	r.Meta["per_page"] = perPage
	r.Meta["total"] = total
	r.Meta["total_pages"] = (total + perPage - 1) / perPage
	return r
}

// JSON returns the JSON-encoded response.
func (r *Response[T]) JSON() ([]byte, error) {
	return json.Marshal(r)
}

// Write writes the response to an http.ResponseWriter.
func (r *Response[T]) Write(w http.ResponseWriter) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(r.StatusCode)

	if r.StatusCode == http.StatusNoContent {
		return nil
	}

	return json.NewEncoder(w).Encode(r)
}

// =============================================================================
// Paginated Response Helper
// =============================================================================

// PagedResponse is a convenience type for paginated list responses.
type PagedResponse[T any] struct {
	Items      []T `json:"items"`
	Page       int `json:"page"`
	PerPage    int `json:"per_page"`
	Total      int `json:"total"`
	TotalPages int `json:"total_pages"`
}

// Paginated creates a paginated response.
func Paginated[T any](items []T, page, perPage, total int) *Response[PagedResponse[T]] {
	return &Response[PagedResponse[T]]{
		Data: PagedResponse[T]{
			Items:      items,
			Page:       page,
			PerPage:    perPage,
			Total:      total,
			TotalPages: (total + perPage - 1) / perPage,
		},
		StatusCode: http.StatusOK,
	}
}
