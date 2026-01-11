package vango

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestResponse_StatusCodes_Meta_Pagination_JSON_Write(t *testing.T) {
	r := OK(map[string]any{"a": 1}).
		WithMeta("request_id", "abc").
		WithPagination(2, 10, 35)

	if r.StatusCode != http.StatusOK {
		t.Fatalf("StatusCode = %d, want %d", r.StatusCode, http.StatusOK)
	}
	if r.Meta["request_id"] != "abc" {
		t.Fatalf("Meta[request_id] = %v, want %q", r.Meta["request_id"], "abc")
	}
	if r.Meta["total_pages"] != 4 {
		t.Fatalf("Meta[total_pages] = %v, want %d", r.Meta["total_pages"], 4)
	}

	raw, err := r.JSON()
	if err != nil {
		t.Fatalf("JSON() error: %v", err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("json.Unmarshal error: %v", err)
	}
	if decoded["data"] == nil || decoded["meta"] == nil {
		t.Fatalf("expected JSON to include data and meta, got %s", string(raw))
	}
	if _, hasStatus := decoded["StatusCode"]; hasStatus {
		t.Fatalf("expected StatusCode to be omitted from JSON, got %s", string(raw))
	}

	rr := httptest.NewRecorder()
	if err := r.Write(rr); err != nil {
		t.Fatalf("Write() error: %v", err)
	}
	if rr.Code != http.StatusOK {
		t.Fatalf("Write() status = %d, want %d", rr.Code, http.StatusOK)
	}
	if ct := rr.Header().Get("Content-Type"); ct != "application/json" {
		t.Fatalf("Content-Type = %q, want %q", ct, "application/json")
	}
	if rr.Body.Len() == 0 {
		t.Fatalf("expected Write() to write a JSON body")
	}

	rr2 := httptest.NewRecorder()
	if err := NoContent[any]().Write(rr2); err != nil {
		t.Fatalf("NoContent().Write() error: %v", err)
	}
	if rr2.Code != http.StatusNoContent {
		t.Fatalf("NoContent().Write status = %d, want %d", rr2.Code, http.StatusNoContent)
	}
	if rr2.Body.Len() != 0 {
		t.Fatalf("NoContent().Write should not write a body, got %q", rr2.Body.String())
	}

	if got := Created("x").StatusCode; got != http.StatusCreated {
		t.Fatalf("Created().StatusCode = %d, want %d", got, http.StatusCreated)
	}
	if got := Accepted("x").StatusCode; got != http.StatusAccepted {
		t.Fatalf("Accepted().StatusCode = %d, want %d", got, http.StatusAccepted)
	}
}

func TestPaginatedResponseHelper(t *testing.T) {
	r := Paginated([]int{1, 2, 3}, 3, 2, 5)
	if r.StatusCode != http.StatusOK {
		t.Fatalf("StatusCode = %d, want %d", r.StatusCode, http.StatusOK)
	}
	if r.Data.Page != 3 || r.Data.PerPage != 2 || r.Data.Total != 5 || r.Data.TotalPages != 3 {
		t.Fatalf("PagedResponse = %+v, want page=3 per_page=2 total=5 total_pages=3", r.Data)
	}
	if len(r.Data.Items) != 3 || r.Data.Items[0] != 1 || r.Data.Items[2] != 3 {
		t.Fatalf("Items = %v, want [1 2 3]", r.Data.Items)
	}
}
