package payment_method

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

func (r *repository) Create(c *gin.Context, method *PaymentMethod) (*PaymentMethod, error) {
	err := r.db.Create(method).Error
	if err != nil {
		return nil, err
	}
	return method, nil
}

func (r *repository) Show(c *gin.Context, id string) (*PaymentMethod, error) {
	method := &PaymentMethod{}
	err := r.db.Where("id = ?", id).Where("deleted_at IS NULL").First(&method).Error
	if err != nil {
		return nil, err
	}
	return method, nil
}

func (r *repository) Index(c *gin.Context) ([]*PaymentMethod, error) {
	var method []*PaymentMethod
	err := r.db.Where("deleted_at IS NULL").Find(&method).Error
	if err != nil {
		return nil, err
	}
	return method, nil
}

func (r *repository) Update(c *gin.Context, id string, method *PaymentMethod) (*PaymentMethod, error) {
	err := r.db.Save(&method).Error
	if err != nil {
		return nil, err
	}
	return method, nil
}
