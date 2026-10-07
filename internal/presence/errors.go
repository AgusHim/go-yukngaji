package presence

import (
	"errors"
	"net/http"

	"mainyuk/internal/apperr"
)

// Error yang bisa dibedakan pemanggil. Dulu semuanya dikembalikan sebagai
// HTTP 500 tanpa kode, sehingga UI pemindaian tidak bisa membedakan "tiket
// salah event" dari "server sedang rusak".
var (
	ErrEventNotFound     = errors.New("event not found")
	ErrTicketNotFound    = errors.New("ticket not found")
	ErrTicketWrongEvent  = errors.New("ticket is not for this event")
	ErrTicketNotPaid     = errors.New("ticket is not paid")
	ErrOrderNotFound     = errors.New("ticket order not found")
	ErrEventClosed       = errors.New("EventIsClosed")
	ErrRegisterClosed    = errors.New("EventRegisterClosed")
	ErrParticipantAbsent = errors.New("UserNotFound")
	ErrPresenceNotFound  = errors.New("presence not found")
)

// Kode mesin yang dikirim bersama pesan error. Frontend memakainya untuk
// memilih pesan yang tepat; teks `error` bisa berubah tanpa merusak UI.
const (
	CodeEventNotFound    = "EVENT_NOT_FOUND"
	CodeTicketNotFound   = "TICKET_NOT_FOUND"
	CodeTicketWrongEvent = "TICKET_WRONG_EVENT"
	CodeTicketNotPaid    = "TICKET_NOT_PAID"
	CodeOrderNotFound    = "ORDER_NOT_FOUND"
	CodeEventClosed      = "EVENT_CLOSED"
	CodeRegisterClosed   = "EVENT_REGISTER_CLOSED"
	CodeParticipantGone  = "PARTICIPANT_NOT_FOUND"
	CodePresenceMissing  = "PRESENCE_NOT_FOUND"
)

// ErrorCode memetakan error ke status HTTP dan kode mesin.
func ErrorCode(err error) (int, string) {
	switch {
	case err == nil:
		return http.StatusOK, ""
	case errors.Is(err, ErrEventNotFound):
		return http.StatusNotFound, CodeEventNotFound
	case errors.Is(err, ErrTicketNotFound):
		return http.StatusNotFound, CodeTicketNotFound
	case errors.Is(err, ErrTicketWrongEvent):
		return http.StatusBadRequest, CodeTicketWrongEvent
	case errors.Is(err, ErrTicketNotPaid):
		return http.StatusConflict, CodeTicketNotPaid
	case errors.Is(err, ErrOrderNotFound):
		return http.StatusNotFound, CodeOrderNotFound
	case errors.Is(err, ErrEventClosed):
		return http.StatusBadRequest, CodeEventClosed
	case errors.Is(err, ErrRegisterClosed):
		return http.StatusBadRequest, CodeRegisterClosed
	case errors.Is(err, ErrParticipantAbsent):
		return http.StatusNotFound, CodeParticipantGone
	case errors.Is(err, ErrPresenceNotFound):
		return http.StatusNotFound, CodePresenceMissing
	case errors.Is(err, apperr.ErrUnauthorized):
		return http.StatusUnauthorized, "UNAUTHORIZED"
	case errors.Is(err, apperr.ErrInvalidRequest):
		return http.StatusBadRequest, "INVALID_REQUEST"
	default:
		return http.StatusInternalServerError, "INTERNAL_ERROR"
	}
}
