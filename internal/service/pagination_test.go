package service

import "testing"

func TestNormalizePagination(t *testing.T) {
	t.Run("defaults", func(t *testing.T) {
		params, err := NormalizePagination(0, 0)
		if err != nil {
			t.Fatalf("NormalizePagination default: %v", err)
		}

		if params.Page != DefaultPage || params.PageSize != DefaultPageSize {
			t.Fatalf("unexpected params: %+v", params)
		}
	})

	t.Run("invalid negative values", func(t *testing.T) {
		if _, err := NormalizePagination(-1, 10); err == nil {
			t.Fatalf("expected negative page error")
		}

		if _, err := NormalizePagination(1, -10); err == nil {
			t.Fatalf("expected negative page_size error")
		}
	})

	t.Run("max page size", func(t *testing.T) {
		if _, err := NormalizePagination(1, MaxPageSize+1); err == nil {
			t.Fatalf("expected page_size upper bound error")
		}
	})
}

func TestBuildPagination(t *testing.T) {
	t.Run("with total count", func(t *testing.T) {
		p := BuildPagination(2, 10, 25)
		if p.TotalPages != 3 || !p.HasPrev || !p.HasNext {
			t.Fatalf("unexpected pagination: %+v", p)
		}
	})

	t.Run("empty total count", func(t *testing.T) {
		p := BuildPagination(1, 20, 0)
		if p.TotalPages != 0 || p.HasPrev || p.HasNext {
			t.Fatalf("unexpected empty pagination: %+v", p)
		}
	})
}
