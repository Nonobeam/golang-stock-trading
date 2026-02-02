package notification

import (
	"crypto/tls"
	"fmt"

	"gopkg.in/gomail.v2"

	"github.com/nonobeam/golang-stock-trading/internal/config"
	"github.com/nonobeam/golang-stock-trading/internal/errors"
	"github.com/nonobeam/golang-stock-trading/internal/logger"
)

type EmailService struct {
	host     string
	port     int
	user     string
	password string
	from     string
}

func NewEmailService(cfg *config.Config) *EmailService {
	return &EmailService{
		host:     cfg.SmtpHost,
		port:     cfg.SmtpPort,
		user:     cfg.SmtpUser,
		password: cfg.SmtpPassword,
		from:     cfg.SmtpFrom,
	}
}

func (s *EmailService) SendEmail(to, subject, body string) error {
	return s.SendHTMLEmail(to, subject, body, false)
}

func (s *EmailService) SendHTMLEmail(to, subject, body string, isHtml bool) error {
	m := gomail.NewMessage()
	m.SetHeader("From", s.from)
	m.SetHeader("To", to)
	m.SetHeader("Subject", subject)

	if isHtml {
		m.SetBody("text/html", body)
	} else {
		m.SetBody("text/plain", body)
	}

	d := gomail.NewDialer(s.host, s.port, s.user, s.password)
	d.TLSConfig = &tls.Config{InsecureSkipVerify: true}

	if err := d.DialAndSend(m); err != nil {
		logger.Error().Err(err).Str("to", to).Msg("Failed to send email")
		return errors.Wrap(err, errors.ErrEmailSendFailed)
	}

	logger.Info().Str("to", to).Str("subject", subject).Msg("Email sent successfully")
	return nil
}

func (s *EmailService) SendStockAlert(to, symbol string, price float64, alertType string) error {
	subject := "Stock Alert: " + symbol
	body := "Alert: " + alertType + "\n" +
		"Symbol: " + symbol + "\n" +
		"Price: " + formatPrice(price)

	return s.SendEmail(to, subject, body)
}

func formatPrice(price float64) string {
	return fmt.Sprintf("%.2f", price)
}
