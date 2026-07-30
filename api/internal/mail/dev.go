package mail

import (
	"context"

	"github.com/gogf/gf/v2/frame/g"
)

// DevSender logs the email instead of sending it — the default when SMTP is
// unconfigured, so local/offline flows are exercisable without a real inbox.
type DevSender struct{}

func NewDev() *DevSender { return &DevSender{} }

func (DevSender) Send(ctx context.Context, to, subject, _ string) error {
	g.Log().Infof(ctx, "[blog:mail:dev] to=%s subject=%q", to, subject)
	return nil
}

func (DevSender) CheckHealth(context.Context) error { return nil }
