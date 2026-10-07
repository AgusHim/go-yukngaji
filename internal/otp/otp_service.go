package otp

import (
	"bytes"
	"crypto/rand"
	"errors"
	"html/template"
	"log"
	"mainyuk/internal/apperr"
	"mainyuk/internal/ratelimit"
	"mainyuk/internal/user"
	"mainyuk/utils"
	"math/big"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type service struct {
	Repository  Repository
	UserService user.Service
}

func NewService(repository Repository, userService user.Service) Service {
	return &service{
		Repository:  repository,
		UserService: userService,
	}
}

func GenerateOTP(length int) (string, error) {
	const charset = "1234567890"
	otp := make([]byte, length)
	for i := range otp {
		randomInt, err := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		if err != nil {
			return "", err
		}
		otp[i] = charset[randomInt.Int64()]
	}
	return string(otp), nil
}

func (s *service) RequestOTP(c *gin.Context, req ReqOtp) (*Otp, error) {
	email := NormalizeEmail(req.Email)
	if email == "" {
		return nil, ErrEmailInvalid
	}

	if !ratelimit.OTPRequest.Allow("otp:" + email) {
		return nil, ErrRateLimited
	}

	now := time.Now()

	active, _ := s.Repository.ShowActive(c, email)
	if active != nil {
		// Masih dalam masa tenggang: jangan kirim ulang.
		if !CanResend(active, now, ResendCooldown) {
			return active, nil
		}
		// Masih berlaku tetapi cooldown sudah lewat: kirim ulang kode yang sama.
		if now.Before(active.ExpiresAt) {
			s.sendOTPAsync(active)
			if err := s.Repository.TouchLastSent(c, active.ID); err != nil {
				log.Printf("[otp] gagal memperbarui last_sent_at %s: %v", active.ID, err)
			}
			return active, nil
		}
	}

	// Kode baru menggantikan kode aktif sebelumnya.
	if err := s.Repository.InvalidateActive(c, email); err != nil {
		return nil, err
	}

	code, err := GenerateOTP(6)
	if err != nil {
		return nil, err
	}

	otp := &Otp{}
	otp.ID = uuid.NewString()
	otp.Email = email
	otp.Code = code
	otp.CreatedAt = now
	otp.ExpiresAt = now.Add(CodeTTL)
	otp.LastSentAt = &now

	otp, err = s.Repository.Create(c, otp)
	if err != nil {
		return nil, err
	}

	s.sendOTPAsync(otp)
	return otp, nil
}

// sendOTPAsync mengirim email di goroutine terpisah agar tidak memblokir
// request. Kegagalan dicatat, tidak ditelan diam-diam.
func (s *service) sendOTPAsync(otp *Otp) {
	go func() {
		tmpl, err := template.ParseFiles("template/otp_template.tmpl")
		if err != nil {
			log.Printf("[otp] gagal membaca template: %s", err)
			return
		}

		var body bytes.Buffer
		if err := tmpl.Execute(&body, otp); err != nil {
			log.Printf("[otp] gagal merender template: %s", err)
			return
		}

		if err := utils.SendEmail(otp.Email, "no-reply@ynsolo.id", "Kode OTP untuk login di ynsolo.id", body.String()); err != nil {
			log.Printf("[otp] gagal mengirim OTP ke %s: %s", otp.Email, err)
		}
	}()
}

func (s *service) VerifyOTP(c *gin.Context, req ReqOtp) (*user.User, error) {
	email := NormalizeEmail(req.Email)
	if email == "" {
		return nil, ErrEmailInvalid
	}

	rateKey := "otpverify:" + email
	if !ratelimit.OTPVerify.Allow(rateKey) {
		return nil, ErrRateLimited
	}

	otp, err := s.Repository.ShowActive(c, email)
	if err != nil {
		return nil, ErrOTPNotFound
	}

	if err := ValidateOTP(otp, req.Code, time.Now(), MaxAttempts); err != nil {
		if errors.Is(err, ErrOTPMismatch) {
			if incErr := s.Repository.IncrementAttempts(c, otp.ID); incErr != nil {
				log.Printf("[otp] gagal menaikkan attempts %s: %v", otp.ID, incErr)
			}
		}
		return nil, err
	}

	// Kode sekali pakai. Bila baris sudah ditandai terpakai oleh request
	// paralel, MarkUsed mengembalikan false dan verifikasi ditolak.
	consumed, err := s.Repository.MarkUsed(c, otp.ID)
	if err != nil {
		return nil, err
	}
	if !consumed {
		return nil, ErrOTPUsed
	}

	ratelimit.OTPVerify.Reset(rateKey)

	return s.UserService.EnsureMemberByEmail(c, email)
}

func (s *service) GetUserIDAuth(c *gin.Context) (*user.User, error) {
	u, ok := user.FromContext(c)
	if !ok {
		return nil, apperr.ErrUnauthorized
	}
	return u, nil
}
