package service

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"platform/backend/internal/models"
)

type LowBalanceRepository interface {
	ListBelowBalance(ctx context.Context, thresholdMinor int64, defaultCurrency string) ([]models.LowBalanceRecipient, error)
}

type MessageSender interface {
	SendMessage(ctx context.Context, chatID int64, text string) error
}

type LowBalanceNotifierOptions struct {
	ThresholdMinor  int64
	DefaultCurrency string

	At         string
	Location   *time.Location
	PaymentURL string
}


type LowBalanceNotifier struct {
	repo   LowBalanceRepository
	sender MessageSender
	log    *slog.Logger
	opts   LowBalanceNotifierOptions
	hour   int
	minute int
}

func NewLowBalanceNotifier(repo LowBalanceRepository, sender MessageSender, log *slog.Logger, opts LowBalanceNotifierOptions) (*LowBalanceNotifier, error) {
	at, err := time.Parse("15:04", opts.At)
	if err != nil {
		return nil, fmt.Errorf("parse notify time %q: %w", opts.At, err)
	}
	if opts.Location == nil {
		opts.Location = time.Local
	}
	return &LowBalanceNotifier{
		repo:   repo,
		sender: sender,
		log:    log,
		opts:   opts,
		hour:   at.Hour(),
		minute: at.Minute(),
	}, nil
}


func (n *LowBalanceNotifier) Run(ctx context.Context) {
	for {
		next := n.nextRun(time.Now())
		n.log.Info("low balance notification scheduled", slog.Time("at", next))

		timer := time.NewTimer(time.Until(next))
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
		}

		n.Notify(ctx)
	}
}

func (n *LowBalanceNotifier) nextRun(now time.Time) time.Time {
	local := now.In(n.opts.Location)
	next := time.Date(local.Year(), local.Month(), local.Day(), n.hour, n.minute, 0, 0, n.opts.Location)
	if !next.After(local) {
		next = time.Date(local.Year(), local.Month(), local.Day()+1, n.hour, n.minute, 0, 0, n.opts.Location)
	}
	return next
}


func (n *LowBalanceNotifier) Notify(ctx context.Context) {
	recipients, err := n.repo.ListBelowBalance(ctx, n.opts.ThresholdMinor, n.opts.DefaultCurrency)
	if err != nil {
		n.log.Error("list low balance accounts failed", slog.Any("error", err))
		return
	}

	sent := 0
	for _, rcp := range recipients {
		if err := n.sender.SendMessage(ctx, rcp.TelegramID, n.message(rcp)); err != nil {
			n.log.Warn("low balance notification failed",
				slog.Int64("user_id", rcp.UserID),
				slog.Any("error", err),
			)
			continue
		}
		sent++
	}

	n.log.Info("low balance notifications sent",
		slog.Int("sent", sent),
		slog.Int("total", len(recipients)),
	)
}

func (n *LowBalanceNotifier) message(rcp models.LowBalanceRecipient) string {
	var b strings.Builder
	fmt.Fprintf(&b, "На вашем балансе %s %s — необходимо пополнить баланс.",
		models.FormatMinor(rcp.AmountMinor), rcp.Currency,
	)
	if n.opts.PaymentURL != "" {
		b.WriteString("\n\n")
		b.WriteString(n.opts.PaymentURL)
	}
	return b.String()
}
