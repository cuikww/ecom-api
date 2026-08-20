package utils

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"strconv"
)

type Pagination struct {
	Page  int `json:"page"`
	Limit int `json:"limit"`
}

func GeneratePaginationFromRequest(c *gin.Context) Pagination {
	limit := 20 // default limit
	page := 1   // default page

	queryLimit := c.Query("limit")
	if queryLimit != "" {
		parsedLimit, err := strconv.Atoi(queryLimit)
		if err == nil && parsedLimit > 0 {
			limit = parsedLimit
		}
		if limit > 100 {
			limit = 100
		}
	}

	queryPage := c.Query("page")
	if queryPage != "" {
		parsedPage, err := strconv.Atoi(queryPage)
		if err == nil && parsedPage > 0 {
			page = parsedPage
		}
	}

	return Pagination{
		Page:  page,
		Limit: limit,
	}
}
func Paginate(p Pagination) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		offset := (p.Page - 1) * p.Limit
		return db.Offset(offset).Limit(p.Limit)
	}
}
