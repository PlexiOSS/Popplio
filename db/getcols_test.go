package db

import (
	"reflect"
	"testing"
)

type getColsFixture struct {
	Included   string `db:"included"`
	Excluded   string `db:"-"`
	Untagged   string
	NoDBButRfl string `reflect:"ignore"`
	Both       string `db:"both" reflect:"ignore"`
}

func TestGetColsFiltersOutExcludedAndUntaggedFields(t *testing.T) {
	got := GetCols(getColsFixture{})
	want := []string{"included"}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("GetCols() = %v, want %v", got, want)
	}
}

type embeddedInner struct {
	InnerCol string `db:"inner_col"`
}

type getColsEmbeddedFixture struct {
	embeddedInner
	OuterCol string `db:"outer_col"`
}

func TestGetColsIncludesEmbeddedFields(t *testing.T) {
	// VisibleFields walks into embedded structs, and callers across the
	// codebase (types.Server et al.) rely on that to avoid re-declaring
	// shared columns — a regression here would silently drop columns from
	// every SELECT built on an embedding struct.
	got := GetCols(getColsEmbeddedFixture{})
	want := []string{"inner_col", "outer_col"}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("GetCols() = %v, want %v", got, want)
	}
}

func TestGetColsEmptyStructReturnsNil(t *testing.T) {
	type empty struct{}

	if got := GetCols(empty{}); got != nil {
		t.Errorf("GetCols(empty{}) = %v, want nil", got)
	}
}

func TestGetColsPreservesDeclarationOrder(t *testing.T) {
	type ordered struct {
		C string `db:"c"`
		A string `db:"a"`
		B string `db:"b"`
	}

	got := GetCols(ordered{})
	want := []string{"c", "a", "b"}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("GetCols() = %v, want %v", got, want)
	}
}
