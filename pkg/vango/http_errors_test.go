package vango

import (
	"errors"
	"testing"
)

func TestHTTPError_Error_Unwrap_StatusCode(t *testing.T) {
	underlying := errors.New("boom")
	e := &HTTPError{Code: 418, Message: "teapot", Err: underlying}

	if got := e.StatusCode(); got != 418 {
		t.Fatalf("StatusCode() = %d, want 418", got)
	}
	if got := e.Error(); got != "teapot: boom" {
		t.Fatalf("Error() = %q, want %q", got, "teapot: boom")
	}
	if !errors.Is(e, underlying) {
		t.Fatalf("errors.Is should match underlying error")
	}
}

func TestHTTPErrorHelpers_DefaultMessagesAndOverrides(t *testing.T) {
	if got := BadRequest(nil); got.StatusCode() != 400 || got.Error() != "bad request" {
		t.Fatalf("BadRequest(nil) = (%d, %q), want (400, %q)", got.StatusCode(), got.Error(), "bad request")
	}

	inner := errors.New("invalid input")
	if got := BadRequest(inner); got.StatusCode() != 400 || got.Error() != "invalid input: invalid input" {
		t.Fatalf("BadRequest(err) = (%d, %q), want (400, %q)", got.StatusCode(), got.Error(), "invalid input: invalid input")
	}

	if got := BadRequestf("x=%d", 7); got.StatusCode() != 400 || got.Error() != "x=7" {
		t.Fatalf("BadRequestf = (%d, %q), want (400, %q)", got.StatusCode(), got.Error(), "x=7")
	}

	if got := Unauthorized(); got.StatusCode() != 401 || got.Error() != "unauthorized" {
		t.Fatalf("Unauthorized() = (%d, %q), want (401, %q)", got.StatusCode(), got.Error(), "unauthorized")
	}
	if got := Unauthorized("no token"); got.StatusCode() != 401 || got.Error() != "no token" {
		t.Fatalf("Unauthorized(msg) = (%d, %q), want (401, %q)", got.StatusCode(), got.Error(), "no token")
	}

	if got := Forbidden(); got.StatusCode() != 403 || got.Error() != "forbidden" {
		t.Fatalf("Forbidden() = (%d, %q), want (403, %q)", got.StatusCode(), got.Error(), "forbidden")
	}
	if got := NotFound(); got.StatusCode() != 404 || got.Error() != "not found" {
		t.Fatalf("NotFound() = (%d, %q), want (404, %q)", got.StatusCode(), got.Error(), "not found")
	}
	if got := Conflict(); got.StatusCode() != 409 || got.Error() != "conflict" {
		t.Fatalf("Conflict() = (%d, %q), want (409, %q)", got.StatusCode(), got.Error(), "conflict")
	}
	if got := UnprocessableEntity(); got.StatusCode() != 422 || got.Error() != "unprocessable entity" {
		t.Fatalf("UnprocessableEntity() = (%d, %q), want (422, %q)", got.StatusCode(), got.Error(), "unprocessable entity")
	}
	if got := ServiceUnavailable(); got.StatusCode() != 503 || got.Error() != "service unavailable" {
		t.Fatalf("ServiceUnavailable() = (%d, %q), want (503, %q)", got.StatusCode(), got.Error(), "service unavailable")
	}

	ie := InternalError(inner)
	if ie.StatusCode() != 500 || ie.Error() != "internal server error: invalid input" {
		t.Fatalf("InternalError(err) = (%d, %q), want (500, %q)", ie.StatusCode(), ie.Error(), "internal server error: invalid input")
	}
	if !errors.Is(ie, inner) {
		t.Fatalf("InternalError should unwrap underlying error")
	}
}

