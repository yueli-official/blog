// Package blogprivacy adapts the Foundation privacy contracts to blog-owned
// data and processing decisions.
package blogprivacy

import (
	"time"

	"github.com/yueli-official/foundation/go/privacy"
)

const (
	BlogOwner         privacy.OwnerKey    = "blog"
	UserSubject       privacy.SubjectKind = "user"
	SubscriberSubject privacy.SubjectKind = "subscriber"

	PublicContent    privacy.DataCategoryKey = "content.public"
	CommentContact   privacy.DataCategoryKey = "comment.contact"
	NetworkSecurity  privacy.DataCategoryKey = "network.security"
	PublicAuthorship privacy.DataCategoryKey = "content.authorship"
	PrivateReaction  privacy.DataCategoryKey = "content.reaction"
	MarketingContact privacy.DataCategoryKey = "marketing.contact"

	BlogNewsletterDataset privacy.DatasetKey = "blog.newsletter"
	BlogCommentsDataset   privacy.DatasetKey = "blog.comments"
	BlogAuthorshipDataset privacy.DatasetKey = "blog.authorship"
	BlogReactionsDataset  privacy.DatasetKey = "blog.reactions"

	NewsletterPurpose  privacy.PurposeKey = "blog.newsletter_delivery"
	NewsletterNotice   privacy.NoticeKey  = "blog.newsletter"
	MeasurementPurpose privacy.PurposeKey = "blog.audience_measurement"
	GPCSignal          privacy.SignalKey  = "global_privacy_control"
)

var publishedAt = time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)

func Definition(ownerKeys ...privacy.OwnerKey) privacy.Definition {
	ownerKey := BlogOwner
	if len(ownerKeys) > 0 && ownerKeys[0] != "" {
		ownerKey = ownerKeys[0]
	}
	return privacy.Definition{
		Version: privacy.DefinitionVersion, Consumer: "blog",
		SubjectKinds: []privacy.SubjectKindDefinition{
			{Key: UserSubject, Description: "Identity user identifier", MaxRefBytes: 128},
			{Key: SubscriberSubject, Description: "Normalized newsletter address", MaxRefBytes: 320},
		},
		DataCategories: []privacy.DataCategoryDefinition{
			{Key: PublicContent, Description: "User-published content"},
			{Key: CommentContact, Description: "Non-public comment contact data"},
			{Key: NetworkSecurity, Description: "Network and client security metadata", Sensitive: true},
			{Key: PublicAuthorship, Description: "Public authorship attribution"},
			{Key: PrivateReaction, Description: "Private reactions and bookmarks"},
			{Key: MarketingContact, Description: "Marketing subscription contact"},
		},
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
				Categories:  []privacy.DataCategoryKey{MarketingContact},
				Notices:     []privacy.NoticeRef{{Key: NewsletterNotice, Revision: 1}},
				EffectiveAt: publishedAt,
			},
			{
				Ref:         privacy.PurposeRef{Key: MeasurementPurpose, Revision: 1},
				Basis:       privacy.BasisLegitimateInterest,
				Categories:  []privacy.DataCategoryKey{NetworkSecurity},
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
		Owner: ownerPtr(blogOwner(ownerKey)),
	}
}

func ownerPtr(owner privacy.OwnerDefinition) *privacy.OwnerDefinition { return &owner }

func blogOwner(ownerKey privacy.OwnerKey) privacy.OwnerDefinition {
	return privacy.OwnerDefinition{
		Ref:          privacy.OwnerRef{Key: ownerKey, Revision: 1},
		SubjectKinds: []privacy.SubjectKind{UserSubject, SubscriberSubject},
		Datasets: []privacy.DatasetDefinition{
			{
				Key: BlogNewsletterDataset, Categories: []privacy.DataCategoryKey{MarketingContact},
				Operations: []privacy.RightsOperation{privacy.RightErasure},
			},
			{
				Key:        BlogCommentsDataset,
				Categories: []privacy.DataCategoryKey{PublicContent, CommentContact, NetworkSecurity},
				Operations: []privacy.RightsOperation{privacy.RightErasure},
			},
			{
				Key:        BlogAuthorshipDataset,
				Categories: []privacy.DataCategoryKey{PublicAuthorship, PublicContent},
				Operations: []privacy.RightsOperation{privacy.RightErasure},
			},
			{
				Key: BlogReactionsDataset, Categories: []privacy.DataCategoryKey{PrivateReaction},
				Operations: []privacy.RightsOperation{privacy.RightErasure},
			},
		},
	}
}
