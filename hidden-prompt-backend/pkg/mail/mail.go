package mail

import (
	"context"
	"fmt"
	"hidden-prompt-backend/pkg/config"
	"hidden-prompt-backend/pkg/constants"
	"hidden-prompt-backend/pkg/smtp"
	"hidden-prompt-backend/pkg/utils"
	"time"

	"github.com/samber/lo"
)

type Mailer interface {
	SendOTPMail(ctx context.Context, toMail string, otp string) error
}

type mailer struct {
	threadSafeWeightedMap utils.ThreadSafeWeightedMap[string]
	m                     map[string]smtp.Service
}

func NewMailer() (Mailer, error) {
	cfg, err := config.GetConfig()
	if err != nil {
		return nil, err
	}
	mapOfConfig := lo.SliceToMap(cfg.MailConnections, func(mCfg config.MailConnectionConfig) (string, config.MailConnectionConfig) {
		return mCfg.Username, mCfg
	})

	mapForService := make(map[string]smtp.Service)
	for key, mailConfig := range mapOfConfig {
		svc, err := smtp.NewService(smtp.NewServiceRequest{
			Host:     mailConfig.Host,
			Port:     mailConfig.Port,
			Username: mailConfig.Username,
			Password: mailConfig.Password,
			From:     mailConfig.Username,
		})
		if err != nil {
			return nil, err
		}
		mapForService[key] = svc
	}

	threadSafe, err := utils.NewThreadSafeWeightedMap(lo.MapEntries(mapOfConfig, func(key string, _ config.MailConnectionConfig) (string, uint) {
		return key, 1
	}))
	if err != nil {
		return nil, err
	}
	return &mailer{
		threadSafeWeightedMap: threadSafe,
		m:                     mapForService,
	}, nil
}

func (m *mailer) SendOTPMail(ctx context.Context, toMail string, otp string) error {
	smtpKey := m.threadSafeWeightedMap.GetRandom()
	serviceSMTP, ok := m.m[smtpKey]
	if !ok {
		return fmt.Errorf("random opted key not found in map: %s", smtpKey)
	}

	return serviceSMTP.SendMail(ctx, smtp.SendMailRequest{
		To:      toMail,
		Subject: "Hidden Prompt Game - Your Verification Code",
		Text: fmt.Sprintf(
			`Your Hidden Prompt Game verification code is:

%s

This code is valid for %d minutes.

If you did not request this code, you can safely ignore this email.`,
			otp,
			int(constants.OTPValidity/time.Minute),
		),
		HTML: fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
<meta charset="UTF-8">
<title>Hidden Prompt Game Verification</title>
</head>
<body style="margin:0;padding:24px;background:#f5f5f5;font-family:Arial,Helvetica,sans-serif;">
	<table role="presentation" width="100%%" cellspacing="0" cellpadding="0">
		<tr>
			<td align="center">
				<table role="presentation" width="480" cellspacing="0" cellpadding="0" style="background:#ffffff;border-radius:12px;padding:32px;">
					<tr>
						<td align="center">
							<h1 style="margin:0;color:#111111;">Hidden Prompt Game</h1>
							<p style="margin:24px 0 8px;color:#555555;font-size:16px;">
								Your verification code is
							</p>

							<div style="margin:24px 0;padding:18px;background:#f2f4f7;border-radius:8px;font-size:36px;font-weight:bold;letter-spacing:8px;color:#111111;">
								%s
							</div>

							<p style="color:#555555;font-size:15px;">
								This code is valid for <strong>%d minutes</strong>.
							</p>

							<p style="margin-top:32px;color:#888888;font-size:13px;">
								If you didn't request this verification code, you can safely ignore this email.
							</p>
						</td>
					</tr>
				</table>
			</td>
		</tr>
	</table>
</body>
</html>`,
			otp,
			int(constants.OTPValidity/time.Minute),
		),
	})
}
