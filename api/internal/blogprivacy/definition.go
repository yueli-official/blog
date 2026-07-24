// Package blogprivacy adapts the Foundation privacy contracts to blog-owned
// data and processing decisions.
package blogprivacy

import (
	"time"

	"github.com/yueli-official/foundation/go/privacy"

	"platform/gokit/privacycatalog"
)

const (
	NewsletterPurpose  privacy.PurposeKey = "blog.newsletter_delivery"
	NewsletterNotice   privacy.NoticeKey  = "blog.newsletter"
	MeasurementPurpose privacy.PurposeKey = "blog.audience_measurement"
	GPCSignal          privacy.SignalKey  = "global_privacy_control"
)

var publishedAt = time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)

func Definition(ownerKeys ...privacy.OwnerKey) privacy.Definition {
	ownerKey := privacycatalog.BlogOwner
	if len(ownerKeys) > 0 && ownerKeys[0] != "" {
		ownerKey = ownerKeys[0]
	}
	return privacy.Definition{
		Version: privacy.DefinitionVersion, Consumer: "platform.blog",
		SubjectKinds:   privacycatalog.SubjectKinds(),
		DataCategories: privacycatalog.Categories(),
		RetentionRules: privacycatalog.RetentionRules(),
		Notices: []privacy.NoticeDefinition{
			{
				Ref:           privacy.NoticeRef{Key: NewsletterNotice, Revision: 1},
				ContentDigest: "5ed63bc4fe4cc949b8f7aaf6e853b49cbf8829234656fc0ee992283575d61f91",
				Locale:        "zh-CN",
				Purposes:      []privacy.PurposeRef{{Key: NewsletterPurpose, Revision: 1}},
				PublishedAt:   publishedAt,
			},
		},
		Purposes: []privacy.PurposeDefinition{
			{
				Ref:         privacy.PurposeRef{Key: NewsletterPurpose, Revision: 1},
				Basis:       privacy.BasisConsent,
				Categories:  []privacy.DataCategoryKey{privacycatalog.MarketingContact},
				Notices:     []privacy.NoticeRef{{Key: NewsletterNotice, Revision: 1}},
				EffectiveAt: publishedAt,
			},
			{
				Ref:         privacy.PurposeRef{Key: MeasurementPurpose, Revision: 1},
				Basis:       privacy.BasisLegitimateInterest,
				Categories:  []privacy.DataCategoryKey{privacycatalog.NetworkSecurity},
				SignalRules: []privacy.SignalRule{{Signal: GPCSignal, Effect: privacy.SignalDeny}},
				EffectiveAt: publishedAt,
			},
		},
		ActivePurposes: []privacy.ActivePurpose{
			{Key: NewsletterPurpose, Ref: privacy.PurposeRef{Key: NewsletterPurpose, Revision: 1}},
			{Key: MeasurementPurpose, Ref: privacy.PurposeRef{Key: MeasurementPurpose, Revision: 1}},
		},
		Signals: []privacy.SignalDefinition{
			{Key: GPCSignal, Description: "Sec-GPC: 1 explicit opt-out signal", MaxEvidenceAge: 24 * time.Hour},
		},
		Owner: ownerPtr(privacycatalog.BlogFor(ownerKey)),
	}
}

func ownerPtr(owner privacy.OwnerDefinition) *privacy.OwnerDefinition { return &owner }
