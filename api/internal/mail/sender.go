// Package mail is the platform's shared transactional-email transport. Each
// service renders its own subject/body; this delivers it (SMTP implicit-TLS, or
// a dev logger). Lifted from the identity service when the blog became the
// second consumer — see conventions "共享件提取标准" (path B: ≥2 real consumers,
// same concept, stable shape, extraction worth the coupling).
package mail

import "context"

// Sender delivers an HTML email best-effort. Implementations return before the
// SMTP round-trip (errors are logged, not surfaced), so response latency is
// identical whether or not a mail is sent — closing the account-enumeration
// timing side channel that auth flows (password reset) depend on.
type Sender interface {
	Send(ctx context.Context, to, subject, htmlBody string) error
}

// HealthChecker performs a side-effect-free transport probe. It must never
// send a message; callers apply their own timeout and authorization policy.
type HealthChecker interface {
	CheckHealth(ctx context.Context) error
}
