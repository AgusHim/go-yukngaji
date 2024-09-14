package region

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) Repository {
	return &repository{
		db: db,
	}
}

func (r *repository) Index(c *gin.Context) ([]*Region, error) {
	var region []*Region
	tx := r.db
	query := tx.Model(&Region{})

	province_code := c.Query("province_code")

	district_code := c.Query("district_code")

	if district_code != "" && province_code == "" {
		query.Where("kode LIKE ?", district_code+".%").Where("LENGTH(kode) = 8")
	} else if province_code != "" && district_code == "" {
		query.Where("kode LIKE ?", province_code+".%").Where("LENGTH(kode) = 5")
	} else {
		query.Where("LENGTH(kode) = 2")
	}

	err := query.Find(&region).Error
	if err != nil {
		return nil, err
	}
	return region, nil
}
