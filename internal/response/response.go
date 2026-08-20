package response

import "github.com/gin-gonic/gin"

const (
	ErrUnauthorized   = "ERR_UNAUTHORIZED"
	ErrInvalidRequest = "ERR_INVALID_REQUEST"
	ErrInternalServer = "ERR_INTERNAL_SERVER"
	ErrNotFound       = "ERR_NOT_FOUND"
	ErrConflict       = "ERR_CONFLICT"
)

type ErrorDetail struct {
	Code    string `json:"code"`
	Mesaage string `json:"message"`
}

func Success(c *gin.Context, status int, data any, meta any) {
	response := gin.H{"data": data}
	if meta != nil {
		response["meta"] = meta
	}
	c.JSON(status, response)
}

func Error(c *gin.Context, status int, code string, fallbackMessage string) {
	c.AbortWithStatusJSON(status, gin.H{
		"error": ErrorDetail{
			Code:    code,
			Mesaage: fallbackMessage,
		},
	})
}
