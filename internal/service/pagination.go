package service

import "wargame/internal/models"

const (
	DefaultPage     = 1
	DefaultPageSize = 20
	MaxPageSize     = 100
)

type PaginationParams struct {
	Page     int
	PageSize int
}

func NormalizePagination(page, pageSize int) (PaginationParams, error) {
	validator := newFieldValidator()
	if page < 0 {
		validator.fields = append(validator.fields, FieldError{Field: "page", Reason: "must be >= 0"})
	}
	if pageSize < 0 {
		validator.fields = append(validator.fields, FieldError{Field: "page_size", Reason: "must be >= 0"})
	}

	if page == 0 {
		page = DefaultPage
	}

	if pageSize == 0 {
		pageSize = DefaultPageSize
	}

	if pageSize > MaxPageSize {
		validator.fields = append(validator.fields, FieldError{Field: "page_size", Reason: "must be <= 100"})
	}

	if err := validator.Error(); err != nil {
		return PaginationParams{}, err
	}

	return PaginationParams{
		Page:     page,
		PageSize: pageSize,
	}, nil
}

func BuildPagination(page, pageSize, totalCount int) models.Pagination {
	totalPages := 0
	if totalCount > 0 {
		totalPages = (totalCount + pageSize - 1) / pageSize
	}

	return models.Pagination{
		Page:       page,
		PageSize:   pageSize,
		TotalCount: totalCount,
		TotalPages: totalPages,
		HasPrev:    page > 1 && totalPages > 0,
		HasNext:    page < totalPages,
	}
}
