package blogprivacy

import (
	"context"
	"testing"
	"time"

	"github.com/yueli-official/foundation/go/privacy"

	"platform/gokit/privacycatalog"
)

func TestDefinitionRequiresNewsletterConsentAndHonorsExplicitGPC(t *testing.T) {
	catalog, err := privacy.Compile(Definition())
	if err != nil {
		t.Fatal(err)
	}
	now := publishedAt.Add(24 * time.Hour)
	runtime, err := privacy.NewMemory(catalog, privacy.MemoryOptions{Clock: func() time.Time { return now }})
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	newsletter, _ := runtime.Purpose(NewsletterPurpose)
	subject := privacy.SubjectRef{
		Owner: privacycatalog.BlogOwner, Kind: privacycatalog.SubscriberSubject, Value: "reader@example.com",
	}
	decision, err := newsletter.Decide(ctx, privacy.DecisionInput{Subject: privacy.SingleSubject(subject)})
	if err != nil || decision.Kind != privacy.DecisionDeny {
		t.Fatalf("newsletter before consent = %#v, %v", decision, err)
	}
	_, err = runtime.Evidence().Consent(ctx, privacy.ConsentCommand{
		IdempotencyKey: "newsletter-consent-1", Subject: subject,
		Notice: newsletterNotice(), Purposes: []privacy.PurposeRef{newsletter.Ref()},
		OccurredAt: now, Channel: "double_opt_in",
	})
	if err != nil {
		t.Fatal(err)
	}
	decision, err = newsletter.Decide(ctx, privacy.DecisionInput{Subject: privacy.SingleSubject(subject)})
	if err != nil || !decision.Allows() {
		t.Fatalf("newsletter after consent = %#v, %v", decision, err)
	}
	measurement, _ := runtime.Purpose(MeasurementPurpose)
	allowed, err := measurement.Decide(ctx, privacy.DecisionInput{})
	if err != nil || !allowed.Allows() {
		t.Fatalf("measurement without signal = %#v, %v", allowed, err)
	}
	denied, err := measurement.Decide(ctx, privacy.DecisionInput{
		Signals: []privacy.ObservedSignal{{Signal: GPCSignal, AssertedAt: now}},
	})
	if err != nil || denied.Kind != privacy.DecisionDeny {
		t.Fatalf("measurement with GPC = %#v, %v", denied, err)
	}
}

func TestSubscriberReferenceUsesTheIndependentSiteOwner(t *testing.T) {
	service := &Service{ownerKey: "site.blog-ai"}
	subject := service.subscriber(" READER@example.com ")
	if subject.Owner != "site.blog-ai" || subject.Value != "reader@example.com" {
		t.Fatalf("subject = %#v", subject)
	}
}
