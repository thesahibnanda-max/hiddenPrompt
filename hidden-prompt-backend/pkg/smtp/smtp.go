package smtp

import (
	"context"
	"errors"
	"fmt"
	"strings"

	mail "github.com/wneessen/go-mail"
)

type NewServiceRequest struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
	From     string `yaml:"from"`
}

type SendMailRequest struct {
	To      string
	Subject string
	Text    string
	HTML    string
}

type Service interface {
	SendMail(ctx context.Context, req SendMailRequest) error
}

type service struct {
	client *mail.Client
	from   string
}

func NewService(config NewServiceRequest) (Service, error) {
	switch {
	case strings.TrimSpace(config.Host) == "":
		return nil, errors.New("smtp host is required")
	case config.Port <= 0:
		return nil, errors.New("smtp port is required")
	case strings.TrimSpace(config.Username) == "":
		return nil, errors.New("smtp username is required")
	case strings.TrimSpace(config.Password) == "":
		return nil, errors.New("smtp password is required")
	case strings.TrimSpace(config.From) == "":
		return nil, errors.New("smtp from address is required")
	}

	client, err := mail.NewClient(
		config.Host,
		mail.WithPort(config.Port),
		mail.WithSMTPAuth(mail.SMTPAuthPlain),
		mail.WithUsername(config.Username),
		mail.WithPassword(config.Password),
		mail.WithTLSPolicy(mail.TLSMandatory),
	)
	if err != nil {
		return nil, fmt.Errorf("create smtp client: %w", err)
	}

	return &service{
		client: client,
		from:   config.From,
	}, nil
}

func (s *service) SendMail(ctx context.Context, req SendMailRequest) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	if strings.TrimSpace(req.To) == "" {
		return errors.New("recipient is required")
	}

	if strings.TrimSpace(req.Subject) == "" {
		return errors.New("subject is required")
	}

	if req.Text == "" && req.HTML == "" {
		return errors.New("either text or html body is required")
	}

	msg := mail.NewMsg()

	if err := msg.From(s.from); err != nil {
		return fmt.Errorf("set from: %w", err)
	}

	if err := msg.To(req.To); err != nil {
		return fmt.Errorf("set recipient: %w", err)
	}

	msg.Subject(req.Subject)

	switch {
	case req.Text != "" && req.HTML != "":
		msg.SetBodyString(mail.TypeTextPlain, req.Text)
		msg.AddAlternativeString(mail.TypeTextHTML, req.HTML)

	case req.HTML != "":
		msg.SetBodyString(mail.TypeTextHTML, req.HTML)

	default:
		msg.SetBodyString(mail.TypeTextPlain, req.Text)
	}

	if err := s.client.DialAndSendWithContext(ctx, msg); err != nil {
		return fmt.Errorf("send mail: %w", err)
	}

	return nil
}
