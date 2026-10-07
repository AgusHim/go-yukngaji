package user

import (
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type repository struct {
	db *gorm.DB
}

func (r *repository) CreateUser(c *gin.Context, user *User) (*User, error) {
	err := r.db.Create(user).Error
	if err != nil {
		return nil, err
	}
	return user, nil
}

func (r *repository) GetUserByEmail(c *gin.Context, email string) (*User, error) {
	var user User
	if err := r.db.Preload("Province").Preload("District").Preload("SubDistrict").Where("email = ?", email).Where("deleted_at IS NULL").Order("created_at DESC").First(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *repository) DeleteByID(c *gin.Context, id string) error {
	var user User
	if err := r.db.Where("id = ?", id).First(&user).Error; err != nil {
		return err
	}
	var now = time.Now()
	user.DeletedAt = &now
	if err := r.db.Save(&user).Error; err != nil {
		return err
	}
	return nil
}

// List membaca akun untuk dashboard pengurus.
//
// Penyaringnya sengaja sederhana: pencarian teks pada nama/username/email dan
// penyaring peran. Pengurutan tidak dapat dipilih pemanggil — urutannya tetap
// terbaru lebih dulu supaya halaman tidak berpindah-pindah antar permintaan.
func (r *repository) List(c *gin.Context, search, role string, limit, offset int) ([]*User, error) {
	users := []*User{}

	query := r.db.Model(&User{}).Where("deleted_at IS NULL")
	if search != "" {
		like := "%" + search + "%"
		query = query.Where("name ILIKE ? OR username ILIKE ? OR email ILIKE ?", like, like, like)
	}
	if role != "" {
		query = query.Where("role = ?", role)
	}

	err := query.
		Order("created_at DESC, id DESC").
		Limit(limit).
		Offset(offset).
		Find(&users).Error
	if err != nil {
		return nil, err
	}
	return users, nil
}

func (r *repository) Show(c *gin.Context, id string) (*User, error) {
	user := &User{}
	err := r.db.Preload("Province").Preload("District").Preload("SubDistrict").Where("id = ?", id).Where("deleted_at IS NULL").Order("created_at DESC").First(&user).Error
	if err != nil {
		return nil, err
	}
	return user, nil
}

func (r *repository) Update(c *gin.Context, id string, user *User) (*User, error) {
	// Gunakan map untuk memastikan hanya field yang ingin diupdate yang dikirim
	err := r.db.Model(&User{}).Where("id = ?", id).Updates(map[string]interface{}{
		"name":              user.Name,
		"gender":            user.Gender,
		"age":               user.Age,
		"phone":             user.Phone,
		"username":          user.Username,
		"address":           user.Address,
		"province_code":     user.ProvinceCode,
		"district_code":     user.DistrictCode,
		"sub_district_code": user.SubDistrictCode,
		"activity":          user.Activity,
		"source":            user.Source,
		"instagram":         user.Instagram,
		"password":          user.Password,
		"birth_date":        user.BirthDate,
		"updated_at":        time.Now(),
	}).Error

	if err != nil {
		return nil, err
	}

	// Kembalikan hasil terkini
	var updated User
	if err := r.db.First(&updated, "id = ?", id).Error; err != nil {
		return nil, err
	}

	return &updated, nil
}

func (r *repository) ShowByGoogleID(c *gin.Context, id string) (*User, error) {
	user := &User{}
	err := r.db.Preload("Province").Preload("District").Preload("SubDistrict").Where("google_id = ?", id).Where("deleted_at IS NULL").First(&user).Error
	if err != nil {
		return nil, err
	}
	return user, nil
}

func NewRepository(db *gorm.DB) Repository {
	return &repository{
		db: db,
	}
}
