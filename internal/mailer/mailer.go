package mailer

import (
	"bytes"
	"context"
	"embed"
	"fmt"
	"html/template"
	"log/slog"
	"time"

	"github.com/wneessen/go-mail"
)

//go:embed "templates"
var templateFS embed.FS

// Mailer is a wrapper struct around mail.Client
// with sender defined and added logging.
// It is used to send emails to users.
type Mailer struct {
	client *mail.Client
	from   string
	logger *slog.Logger
}

// New initializes a Mailer object with all the necessary SMTP settings.
// Returns an error if client creation fails.
func New(host string, port int, username, password, from string, logger *slog.Logger) (*Mailer, error) {
	client, err := mail.NewClient(host,
		mail.WithPort(port),
		mail.WithSMTPAuth(mail.SMTPAuthAutoDiscover),
		mail.WithUsername(username),
		mail.WithPassword(password),
		mail.WithTLSPortPolicy(mail.TLSMandatory),
		mail.WithTimeout(5*time.Second),
	)

	if err != nil {
		return nil, fmt.Errorf("mailer: create client: %w", err)
	}

	return &Mailer{
		client: client,
		from:   from,
		logger: logger,
	}, nil
}

// Send loads the specified email template, executes it with provided data
// and then tries to send it 3 times with a 500ms interval, logging each fail.
// Returns an error if it fails to send the email.
func (m *Mailer) Send(ctx context.Context, templateFile string, data any, to ...string) error {
	if len(to) == 0 {
		return fmt.Errorf("mailer: no recipients provided")
	}

	tmpl, err := template.New("email").ParseFS(templateFS, "templates/"+templateFile)
	if err != nil {
		return fmt.Errorf("mailer: parse template %s: %w", templateFile, err)
	}

	subject := new(bytes.Buffer)
	if err := tmpl.ExecuteTemplate(subject, "subject", data); err != nil {
		return fmt.Errorf("mailer: execute subject: %w", err)
	}

	plainBody := new(bytes.Buffer)
	if err := tmpl.ExecuteTemplate(plainBody, "plainBody", data); err != nil {
		return fmt.Errorf("mailer: execute plainBody: %w", err)
	}

	htmlBody := new(bytes.Buffer)
	if err := tmpl.ExecuteTemplate(htmlBody, "htmlBody", data); err != nil {
		return fmt.Errorf("mailer: execute htmlBody: %w", err)
	}

	message := mail.NewMsg()
	if err := message.From(m.from); err != nil {
		return fmt.Errorf("mailer: set from: %w", err)
	}
	if err := message.To(to...); err != nil {
		return fmt.Errorf("mailer: set to: %w", err)
	}
	message.Subject(subject.String())
	message.SetBodyString(mail.TypeTextPlain, plainBody.String())
	message.AddAlternativeString(mail.TypeTextHTML, htmlBody.String())

	var sendErr error
	for i := 1; i <= 3; i++ {
		sendErr = m.client.DialAndSendWithContext(ctx, message)
		if sendErr == nil {
			return nil
		}

		// we log the failed attempt
		m.logger.WarnContext(ctx, "mailer: send attempt failed",
			slog.Int("attempt", i),
			slog.Int("recipient_count", len(to)), // non-PII
			slog.String("error", sendErr.Error()))

		// timer is not gc'ed correctly, so we initialize it outside of the select
		timer := time.NewTimer(500 * time.Millisecond)
		select {
		case <-ctx.Done():
			timer.Stop()
			return fmt.Errorf("mailer: context done: %w", ctx.Err())
		case <-timer.C:
		}
	}

	return fmt.Errorf("mailer: send after 3 attempts: %w", sendErr)
}
