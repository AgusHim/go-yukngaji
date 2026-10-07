package user

import (
	"errors"
	"log"
	"mainyuk/internal/authz"
	"mainyuk/utils"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	oauth2api "google.golang.org/api/oauth2/v2"
)

type service struct {
	Repository
	ticketClaimer   TicketClaimer
	profileRewarder ProfileRewarder
}

func NewService(repository Repository) Service {
	return &service{
		Repository: repository,
	}
}

// SetTicketClaimer memasang pengklaim tiket rombongan.
func (s *service) SetTicketClaimer(tc TicketClaimer) {
	s.ticketClaimer = tc
}

// SetProfileRewarder memasang pemberi reward profil lengkap.
func (s *service) SetProfileRewarder(pr ProfileRewarder) {
	s.profileRewarder = pr
}

// rewardProfileComplete memicu penilaian "profil lengkap" setelah profil
// tersimpan. Best-effort: kegagalan dicatat, bukan dikembalikan, supaya
// penyimpanan profil tidak ikut gagal karena masalah XP.
func (s *service) rewardProfileComplete(c *gin.Context, userID string) {
	if s.profileRewarder == nil {
		return
	}
	if err := s.profileRewarder.OnProfileCompleted(c, userID); err != nil {
		log.Printf("[xp] gagal memberi reward profil lengkap untuk %s: %v", userID, err)
	}
}

// claimTickets mengaitkan tiket yang email pesertanya baru terdaftar.
// Best-effort: kegagalan dicatat, bukan dikembalikan, supaya pendaftaran
// dan login tidak ikut gagal.
func (s *service) claimTickets(c *gin.Context, u *User) {
	if s.ticketClaimer == nil || u == nil || u.Email == nil || *u.Email == "" {
		return
	}
	claimed, err := s.ticketClaimer.ClaimTicketsByEmail(c, u.ID, *u.Email)
	if err != nil {
		log.Printf("[ticket-claim] gagal mengklaim tiket untuk %s: %v", *u.Email, err)
		return
	}
	if claimed > 0 {
		log.Printf("[ticket-claim] %d tiket diklaim oleh %s", claimed, *u.Email)
	}
}

// Register implements Service
func (s *service) Register(c *gin.Context, req *CreateUser) (*User, error) {
	u, _ := s.GetUserByEmail(c, *req.Email)
	if u != nil {
		return nil, errors.New("EmailRegistered")
	}
	user := &User{}
	user.ID = uuid.NewString()
	user.Name = req.Name
	user.Username = req.Username
	user.Gender = req.Gender

	age, errAge := strconv.Atoi(req.Age)
	if errAge != nil {
		return nil, errAge
	}

	user.Age = age
	user.Phone = req.Phone
	user.Email = req.Email
	user.Address = req.Address
	user.Role = authz.RoleMember

	activity := strings.ToLower(req.Activity)
	user.Activity = &activity

	if req.Password != nil {
		hash, errHash := utils.HashPassword(*req.Password)
		if errHash != nil {
			return nil, errHash
		}
		user.Password = &hash
	}

	user.CreatedAt = time.Now()
	user.UpdatedAt = time.Now()

	user, err := s.Repository.CreateUser(c, user)
	if err != nil {
		return nil, err
	}
	s.claimTickets(c, user)
	return user, nil
}

// Register implements Service
func (s *service) Login(c *gin.Context, req *Login) (*User, error) {
	user, err := s.Repository.GetUserByEmail(c, req.Email)
	if err != nil {
		return nil, errors.New("EmailNotFound")
	}

	if err := utils.CheckPassword(req.Password, *user.Password); err != nil {
		return nil, errors.New("PasswordNotMatch")
	}

	return user, nil
}

func (s *service) Presence(c *gin.Context, req *CreateUser) (*User, error) {
	user := &User{}
	user.ID = uuid.NewString()
	user.Name = req.Name

	if req.Username == "" {
		user.Username = "anonim"
	} else {
		user.Username = req.Username
	}
	user.Gender = req.Gender

	age, errAge := strconv.Atoi(req.Age)
	if errAge != nil {
		return nil, errAge
	}

	user.Age = age
	user.Phone = req.Phone
	user.Email = req.Email
	user.Address = req.Address
	user.Role = authz.RoleJamaah

	activity := strings.ToLower(req.Activity)
	user.Activity = &activity

	if req.Password != nil {
		hash, errHash := utils.HashPassword(*req.Password)
		if errHash != nil {
			return nil, errHash
		}
		user.Password = &hash
	}

	user.CreatedAt = time.Now()
	user.UpdatedAt = time.Now()

	user, err := s.Repository.CreateUser(c, user)
	if err != nil {
		return nil, err
	}
	s.claimTickets(c, user)
	return user, nil
}

// EnsureMemberByEmail mengembalikan akun dengan email tersebut; bila belum
// ada, akun anggota baru dibuat. Dipakai alur login OTP.
func (s *service) EnsureMemberByEmail(c *gin.Context, email string) (*User, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	if email == "" {
		return nil, errors.New("EmailRequired")
	}

	u, _ := s.Repository.GetUserByEmail(c, email)
	if u != nil {
		return u, nil
	}

	created := &User{}
	created.ID = uuid.NewString()
	created.Name = ""
	created.Username = "anonim"
	created.Gender = "male"
	created.Age = 0
	created.Phone = ""
	created.Email = &email
	created.Address = ""
	created.Role = authz.RoleMember
	created.CreatedAt = time.Now()
	created.UpdatedAt = time.Now()

	created, err := s.Repository.CreateUser(c, created)
	if err != nil {
		return nil, err
	}
	s.claimTickets(c, created)
	return created, nil
}

func (s *service) DeleteByID(c *gin.Context, id string) error {
	err := s.Repository.DeleteByID(c, id)
	if err != nil {
		return err
	}
	return nil
}

func (s *service) Show(c *gin.Context, id string) (*User, error) {
	event, err := s.Repository.Show(c, id)
	if err != nil {
		return nil, err
	}
	return event, nil
}

func (s *service) CreateRanger(c *gin.Context, req *CreateUser) (*User, error) {
	u, _ := s.GetUserByEmail(c, *req.Email)
	if u != nil {
		return nil, errors.New("EmailRegistered")
	}
	user := &User{}
	user.ID = uuid.NewString()
	user.Name = req.Name
	user.Username = req.Username
	user.Gender = req.Gender

	age, errAge := strconv.Atoi(req.Age)
	if errAge != nil {
		return nil, errAge
	}
	user.Age = age

	user.Phone = req.Phone
	user.Email = req.Email
	user.Address = req.Address
	user.Role = authz.RoleRanger

	activity := strings.ToLower(req.Activity)
	user.Activity = &activity

	if req.Password != nil {
		hash, errHash := utils.HashPassword(*req.Password)
		if errHash != nil {
			return nil, errHash
		}
		user.Password = &hash
	}

	user.CreatedAt = time.Now()
	user.UpdatedAt = time.Now()

	user, err := s.Repository.CreateUser(c, user)
	if err != nil {
		return nil, err
	}
	return user, nil
}

func (s *service) Update(c *gin.Context, id string, u *CreateUser) (*User, error) {
	// Check User in Database
	user, err := s.Repository.Show(c, id)
	if err != nil {
		return nil, err
	}

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
	if u.ProvinceCode != nil {
		user.ProvinceCode = *u.ProvinceCode
	}
	if u.DistrictCode != nil {
		user.DistrictCode = *u.DistrictCode
	}
	if u.SubDistrictCode != nil {
		user.SubDistrictCode = *u.SubDistrictCode
	}
	user.Activity = &u.Activity
	user.Source = &u.Source

	// if u.Email != nil {
	// 	user.Email = u.Email
	// }

	if u.Instagram != nil {
		user.Instagram = *u.Instagram
	}

	if u.Password != nil && *u.Password != "" {
		hash, errHash := utils.HashPassword(*u.Password)
		if errHash != nil {
			return nil, errHash
		}
		user.Password = &hash
	}

	if u.BirthDate != nil && *u.BirthDate != "" {
		birthDate, errParsed := time.Parse("2006-01-02T15:04", *u.BirthDate)
		if errParsed != nil {
			return nil, errParsed
		}
		user.BirthDate = birthDate
	}

	user.UpdatedAt = time.Now()

	_, err = s.Repository.Update(c, id, user)
	if err != nil {
		return nil, err
	}

	user, err = s.Repository.Show(c, id)
	if err != nil {
		return nil, err
	}

	// Profil baru saja tersimpan; nilai ulang kelengkapannya. Idempoten —
	// hanya pemberian pertama yang menghasilkan XP.
	s.rewardProfileComplete(c, id)

	return user, nil
}

func (s *service) AuthGoogleCallback(c *gin.Context, userInfo *oauth2api.Userinfo) (*User, error) {
	// Check google id in table
	u, _ := s.Repository.ShowByGoogleID(c, userInfo.Id)
	if u != nil {
		s.claimTickets(c, u)
		return u, nil
	}

	u, _ = s.Repository.GetUserByEmail(c, userInfo.Email)
	if u != nil {
		u.GoogleID = &userInfo.Id
		u.UpdatedAt = time.Now()
		s.Repository.Update(c, u.ID, u)
		s.claimTickets(c, u)
		return u, nil
	}
	// Registered new user
	user := &User{}
	user.ID = uuid.NewString()
	user.Name = userInfo.Name
	user.GoogleID = &userInfo.Id
	user.ImageUrl = &userInfo.Picture
	user.Username = "anonim"
	user.Gender = strings.ToLower(userInfo.Gender)

	age := 0
	user.Age = age
	user.Phone = ""
	user.Email = &userInfo.Email
	user.Address = ""
	user.Role = authz.RoleMember

	activity := ""
	user.Activity = &activity

	user.CreatedAt = time.Now()
	user.UpdatedAt = time.Now()

	createdUser, err := s.Repository.CreateUser(c, user)
	if err != nil {
		return nil, err
	}
	s.claimTickets(c, createdUser)
	return createdUser, nil
}

// List mengembalikan satu halaman akun untuk dashboard pengurus.
//
// Hasilnya melewati ToAccountResponse, jadi kata sandi dan google_id tidak
// pernah ikut terkirim — sama seperti endpoint akun lain.
func (s *service) List(c *gin.Context, search, role string, page, perPage int) ([]*AccountResponse, bool, error) {
	page, perPage = normalizePagination(page, perPage)

	users, err := s.Repository.List(c, strings.TrimSpace(search), strings.TrimSpace(role), perPage+1, (page-1)*perPage)
	if err != nil {
		return nil, false, err
	}

	// Satu baris lebih diambil untuk mengetahui masih ada halaman berikutnya,
	// sehingga tidak perlu COUNT(*) terpisah.
	hasMore := len(users) > perPage
	if hasMore {
		users = users[:perPage]
	}

	res := make([]*AccountResponse, 0, len(users))
	for _, u := range users {
		res = append(res, ToAccountResponse(u))
	}
	return res, hasMore, nil
}

// normalizePagination membatasi page/per_page ke rentang yang sama dengan
// modul lain: halaman minimal 1, 20 baris bawaan, maksimum 100.
//
// Disalin, bukan diimpor dari gamification, karena gamification sudah
// mengimpor paket ini — memakai bantuannya akan membuat impor melingkar.
// Nilainya harus tetap sama dengan gamification.NormalizePagination.
func normalizePagination(page, perPage int) (int, int) {
	if page < 1 {
		page = 1
	}
	if perPage < 1 {
		perPage = 20
	}
	if perPage > 100 {
		perPage = 100
	}
	return page, perPage
}
