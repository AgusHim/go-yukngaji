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

// ContextKey adalah kunci gin.Context tempat middleware menyimpan user aktif.
const ContextKey = "currentUser"

// FromContext mengambil user aktif yang dipasang middleware autentikasi.
// Handler harus memakai ini alih-alih mempercayai user_id dari body request.
func FromContext(c *gin.Context) (*User, bool) {
	v, exists := c.Get(ContextKey)
	if !exists {
		return nil, false
	}
	u, ok := v.(User)
	if !ok {
		return nil, false
	}
	return &u, true
}

// AccountResponse adalah kontrak publik data akun. Password dan GoogleID
// sengaja tidak disertakan agar data internal tidak ikut terkirim ke client.
type AccountResponse struct {
	ID              string         `json:"id"`
	Name            string         `json:"name"`
	Username        string         `json:"username"`
	Gender          string         `json:"gender"`
	BirthDate       time.Time      `json:"birth_date"`
	Age             int            `json:"age"`
	Phone           string         `json:"phone"`
	Email           *string        `json:"email"`
	Instagram       string         `json:"instagram"`
	Address         string         `json:"address"`
	Role            string         `json:"role"`
	Activity        *string        `json:"activity"`
	Source          *string        `json:"source"`
	ImageUrl        *string        `json:"image_url"`
	ProvinceCode    string         `json:"province_code"`
	DistrictCode    string         `json:"district_code"`
	SubDistrictCode string         `json:"sub_district_code"`
	Province        *region.Region `json:"province"`
	District        *region.Region `json:"district"`
	SubDistrict     *region.Region `json:"sub_district"`
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
}

// ToAccountResponse mengubah model internal menjadi DTO publik.
func ToAccountResponse(u *User) *AccountResponse {
	if u == nil {
		return nil
	}
	return &AccountResponse{
		ID:              u.ID,
		Name:            u.Name,
		Username:        u.Username,
		Gender:          u.Gender,
		BirthDate:       u.BirthDate,
		Age:             u.Age,
		Phone:           u.Phone,
		Email:           u.Email,
		Instagram:       u.Instagram,
		Address:         u.Address,
		Role:            u.Role,
		Activity:        u.Activity,
		Source:          u.Source,
		ImageUrl:        u.ImageUrl,
		ProvinceCode:    u.ProvinceCode,
		DistrictCode:    u.DistrictCode,
		SubDistrictCode: u.SubDistrictCode,
		Province:        u.Province,
		District:        u.District,
		SubDistrict:     u.SubDistrict,
		CreatedAt:       u.CreatedAt,
		UpdatedAt:       u.UpdatedAt,
	}
}

// TicketClaimer mengaitkan tiket rombongan yang belum diklaim ke akun
// berdasarkan email peserta. Diimplementasikan oleh user_ticket.Service.
// Dipisah sebagai interface agar paket user tidak perlu mengimpor user_ticket.
type TicketClaimer interface {
	ClaimTicketsByEmail(c *gin.Context, userID string, email string) (int64, error)
}

// ProfileRewarder memberi reward "profil lengkap". Diimplementasikan oleh
// gamification.Service. Dipisah sebagai interface agar paket user tidak perlu
// mengimpor gamification.
//
// Implementasinya wajib idempoten: hook ini dipanggil setiap kali profil
// diperbarui, dan hanya pemberian pertama yang boleh menghasilkan XP.
type ProfileRewarder interface {
	OnProfileCompleted(c *gin.Context, userID string) error
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
	// List membaca satu halaman akun. search mencocokkan nama, username, atau
	// email; role menyaring peran. Keduanya kosong berarti tanpa penyaring.
	List(c *gin.Context, search, role string, limit, offset int) ([]*User, error)
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
	// EnsureMemberByEmail mengembalikan akun dengan email tersebut, atau
	// membuat akun anggota baru bila belum ada. Dipakai alur OTP.
	EnsureMemberByEmail(c *gin.Context, email string) (*User, error)
	// SetTicketClaimer memasang pengklaim tiket rombongan. Best-effort:
	// kegagalan klaim tidak boleh menggagalkan pendaftaran/login.
	SetTicketClaimer(tc TicketClaimer)
	// SetProfileRewarder memasang pemberi reward profil lengkap. Best-effort:
	// kegagalan pemberian XP tidak boleh menggagalkan penyimpanan profil.
	SetProfileRewarder(pr ProfileRewarder)
	// List mengembalikan satu halaman akun dalam bentuk kontrak publik.
	// Mengembalikan has_more, bukan jumlah total, sama seperti daftar lain.
	List(c *gin.Context, search, role string, page, perPage int) ([]*AccountResponse, bool, error)
}

type Handler interface {
	Register(c *gin.Context)
	Login(c *gin.Context)
	UpdateByAdmin(c *gin.Context)
	UpdateAuth(c *gin.Context)
	Show(c *gin.Context)
	Me(c *gin.Context)
	AuthGoogleLogin(c *gin.Context)
	AuthGoogleCallback(c *gin.Context)
	List(c *gin.Context)
}
