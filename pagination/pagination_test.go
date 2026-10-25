package pagination

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func request(query string) *http.Request {
	return httptest.NewRequest(http.MethodGet, "/?"+query, nil)
}

func TestParseDefaultsToOneWhenAbsent(t *testing.T) {
	page, err := Parse(request(""))

	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	if page != 1 {
		t.Errorf("page = %d, want 1", page)
	}
}

func TestParseValidPage(t *testing.T) {
	page, err := Parse(request("page=5"))

	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	if page != 5 {
		t.Errorf("page = %d, want 5", page)
	}
}

func TestParseRejectsZero(t *testing.T) {
	if _, err := Parse(request("page=0")); err == nil {
		t.Error("expected an error for page 0")
	}
}

func TestParseAcceptsMaxPage(t *testing.T) {
	page, err := Parse(request("page=100000"))

	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	if page != MaxPage {
		t.Errorf("page = %d, want %d", page, MaxPage)
	}
}

func TestParseRejectsAboveMaxPage(t *testing.T) {
	if _, err := Parse(request("page=100001")); err == nil {
		t.Error("expected an error for a page above MaxPage")
	}
}

func TestParseRejectsNegative(t *testing.T) {
	if _, err := Parse(request("page=-1")); err == nil {
		t.Error("expected an error for a negative page")
	}
}

func TestParseRejectsNonNumeric(t *testing.T) {
	if _, err := Parse(request("page=abc")); err == nil {
		t.Error("expected an error for a non-numeric page")
	}
}

func TestParseRejectsDecimal(t *testing.T) {
	if _, err := Parse(request("page=1.5")); err == nil {
		t.Error("expected an error for a decimal page")
	}
}

func TestParseRejectsOverflow(t *testing.T) {
	if _, err := Parse(request("page=4294967296")); err == nil {
		t.Error("expected an error for a page number that overflows 32 bits")
	}
}
