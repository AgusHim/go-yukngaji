package metrics

import (
	"time"

	"mainyuk/internal/period"

	"github.com/gin-gonic/gin"
)

type service struct {
	Repository
}

func NewService(repository Repository) Service {
	return &service{Repository: repository}
}

// Show menyusun seluruh angka laporan.
//
// Pembacaannya berurutan, bukan paralel: satu laporan bukan jalur panas, dan
// menjalankannya serentak akan menambah kerumitan yang tidak sepadan.
//
// Setiap kegagalan mengembalikan galat apa adanya. Laporan yang sebagian
// kosong lebih berbahaya daripada laporan yang gagal, karena angka nol pada
// dashboard terbaca sebagai "tidak ada aktivitas".
func (s *service) Show(c *gin.Context) (*View, error) {
	w := WindowsFor(time.Now(), period.JakartaLocation)

	monthActive, err := s.Repository.ActiveAccounts(c, w.MonthFrom, w.MonthTo)
	if err != nil {
		return nil, err
	}
	day7Active, err := s.Repository.ActiveAccounts(c, w.Day7From, w.Now)
	if err != nil {
		return nil, err
	}
	day30Active, err := s.Repository.ActiveAccounts(c, w.Day30From, w.Now)
	if err != nil {
		return nil, err
	}

	approvedTotal, err := s.Repository.ApprovedClaimsTotal(c)
	if err != nil {
		return nil, err
	}
	approvedMonth, err := s.Repository.ApprovedClaims(c, w.MonthFrom, w.MonthTo)
	if err != nil {
		return nil, err
	}

	donationsTotal, err := s.Repository.ConfirmedDonationsTotal(c)
	if err != nil {
		return nil, err
	}
	donationsMonth, err := s.Repository.ConfirmedDonations(c, w.MonthFrom, w.MonthTo)
	if err != nil {
		return nil, err
	}

	ordersMonth, err := s.Repository.ShopOrders(c, w.MonthFrom, w.MonthTo)
	if err != nil {
		return nil, err
	}
	ordersByPayment, err := s.Repository.ShopOrdersByPaymentStatus(c)
	if err != nil {
		return nil, err
	}
	ordersByFulfillment, err := s.Repository.ShopOrdersByFulfillmentStatus(c)
	if err != nil {
		return nil, err
	}

	threadsMonth, err := s.Repository.Threads(c, w.MonthFrom, w.MonthTo)
	if err != nil {
		return nil, err
	}
	commentsMonth, err := s.Repository.ThreadComments(c, w.MonthFrom, w.MonthTo)
	if err != nil {
		return nil, err
	}
	reactionsMonth, err := s.Repository.ThreadReactions(c, w.MonthFrom, w.MonthTo)
	if err != nil {
		return nil, err
	}
	reportsMonth, err := s.Repository.ThreadReports(c, w.MonthFrom, w.MonthTo)
	if err != nil {
		return nil, err
	}
	reportsByStatus, err := s.Repository.ThreadReportsByStatus(c)
	if err != nil {
		return nil, err
	}

	return &View{
		GeneratedAt: w.Now,
		ActiveUsers: ActiveUsers{
			CalendarMonth: window(w.MonthFrom, w.MonthTo, monthActive),
			Last7Days:     window(w.Day7From, w.Now, day7Active),
			Last30Days:    window(w.Day30From, w.Now, day30Active),
		},
		Missions: MissionMetrics{
			ApprovedTotal:   approvedTotal,
			ApprovedInMonth: window(w.MonthFrom, w.MonthTo, approvedMonth),
		},
		Donations: DonationMetrics{
			PaidTotal:   donationsTotal,
			PaidInMonth: window(w.MonthFrom, w.MonthTo, donationsMonth),
		},
		Orders: OrderMetrics{
			ByPaymentStatus:     ordersByPayment,
			ByFulfillmentStatus: ordersByFulfillment,
			CreatedInMonth:      window(w.MonthFrom, w.MonthTo, ordersMonth),
		},
		Community: CommunityMetrics{
			ThreadsInMonth:   window(w.MonthFrom, w.MonthTo, threadsMonth),
			CommentsInMonth:  window(w.MonthFrom, w.MonthTo, commentsMonth),
			ReactionsInMonth: window(w.MonthFrom, w.MonthTo, reactionsMonth),
			ReportsInMonth:   window(w.MonthFrom, w.MonthTo, reportsMonth),
			ReportsByStatus:  reportsByStatus,
		},
	}, nil
}

func window(from, to time.Time, value int64) Window {
	return Window{From: from, To: to, Value: value}
}
