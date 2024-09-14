package region

import (
	"github.com/gin-gonic/gin"
)

type Region struct {
	Code string `json:"code" gorm:"column:kode;primaryKey"`
	Name string `json:"name" gorm:"column:nama"`
}

func (Region) TableName() string {
	return "wilayah"
}

type Repository interface {
	Index(ctx *gin.Context) ([]*Region, error)
}

type Service interface {
	Index(ctx *gin.Context) ([]*Region, error)
}

type Handler interface {
	Index(ctx *gin.Context)
}
