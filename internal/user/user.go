package user

import (
	"mainyuk/internal/region"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	oauth2api "google.golang.org/api/oauth2/v2"
)

type User struct {
	ID              string         `json:"id" `
	Name            string         `json:"name" binding:"required"`
	Username        string         `json:"username" binding:"required"`
	Gender          string         `json:"gender" binding:"required"`
	BirthDate       time.Time      `json:"birth_date"`
	Age             int            `json:"age" gorm:"default:0"`
	Phone           string         `json:"phone" binding:"required"`
	Email           *string        `json:"email" binding:"required"`
	Instagram       string         `json:"instagram"`
	Address         string         `json:"address" binding:"required"`
	Password        *string        `json:"-" binding:"required"`
	Role            string         `json:"role"`
	Activity        *string        `json:"activity"`
	Source          *string        `json:"source"`
	GoogleID        *string        `json:"-" gorm:"google_id"`
	ImageUrl        *string        `json:"image_url"`
	ProvinceCode    string         `json:"province_code" gorm:"province_code;size:2"`                        // Stores the first 2 digits of the ID
	DistrictCode    string         `json:"district_code" gorm:"district_code;size:5"`                        // Stores the first 5 digits of the ID
	SubDistrictCode string         `json:"sub_district_code" gorm:"sub_district_code;size:8"`                // Stores the first 8 digits of the code
	Province        *region.Region `json:"province" gorm:"foreignKey:province_code;references:kode"`         // Relation to Province
	District        *region.Region `json:"district" gorm:"foreignKey:district_code;references:kode"`         // Relation to District
	SubDistrict     *region.Region `json:"sub_district" gorm:"foreignKey:sub_district_code;references:kode"` // Relation to Sub-district
	CreatedAt       time.Time      `json:"created_at" `
	UpdatedAt       time.Time      `json:"updated_at" `
	DeletedAt       *time.Time     `json:"-" `
}

type Login struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type CreateUser struct {
	Name            string  `json:"name" binding:"required"`
	Gender          string  `json:"gender"`
	Age             string  `json:"age"`
	BirthDate       *string `json:"birth_date"`
	Phone           string  `json:"phone"`
	Email           *string `json:"email"`
	Instagram       *string `json:"instagram"`
	Username        string  `json:"username"`
	Address         string  `json:"address"`
	Password        *string `json:"password"`
	Activity        string  `json:"activity" binding:"required"`
	Source          string  `json:"source" binding:"required"`
	ProvinceCode    *string `json:"province_code" binding:"required"`
	DistrictCode    *string `json:"district_code" binding:"required"`
	SubDistrictCode *string `json:"sub_district_code" binding:"required"`
}

func CreateUserToUser(u CreateUser) (res *User, err error) {
	user := User{}
	user.Name = u.Name
	user.Gender = u.Gender
	age, err := strconv.Atoi(u.Age)
	if err != nil {
		return nil, err
	}
	user.Age = age

	user.Phone = u.Phone
	user.Username = u.Username
	user.Address = u.Address
	user.Activity = &u.Activity
	user.Email = u.Email
	user.Password = u.Password
	return &user, nil
}

type Repository interface {
	CreateUser(c *gin.Context, user *User) (*User, error)
	GetUserByEmail(c *gin.Context, email string) (*User, error)
	DeleteByID(c *gin.Context, id string) error
	Show(c *gin.Context, id string) (*User, error)
	Update(c *gin.Context, id string, user *User) (*User, error)
	ShowByGoogleID(c *gin.Context, id string) (*User, error)
}

type Service interface {
	Register(c *gin.Context, user *CreateUser) (*User, error)
	Login(c *gin.Context, user *Login) (*User, error)
	GetUserByEmail(c *gin.Context, email string) (*User, error)
	Show(c *gin.Context, id string) (*User, error)
	Presence(c *gin.Context, user *CreateUser) (*User, error)
	DeleteByID(c *gin.Context, id string) error
	CreateRanger(c *gin.Context, user *CreateUser) (*User, error)
	Update(c *gin.Context, id string, user *CreateUser) (*User, error)
	AuthGoogleCallback(c *gin.Context, userInfo *oauth2api.Userinfo) (*User, error)
}

type Handler interface {
	Register(c *gin.Context)
	Login(c *gin.Context)
	UpdateByAdmin(c *gin.Context)
	UpdateAuth(c *gin.Context)
	Show(c *gin.Context)
	AuthGoogleLogin(c *gin.Context)
	AuthGoogleCallback(c *gin.Context)
}
