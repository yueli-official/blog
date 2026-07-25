// Package blogauthz owns Blog's instance-local authorization declaration.
// Foundation owns execution and persistence; Blog owns capabilities, scopes,
// roles, predicates, and resource relations.
package blogauthz

import (
	"context"

	"github.com/yueli-official/foundation/go/authorization"
)

const (
	RootScopeID authorization.ScopeID = "blog"

	ScopeSite    authorization.ScopeType = "site"
	ScopePost    authorization.ScopeType = "post"
	ScopeSeries  authorization.ScopeType = "series"
	ScopeComment authorization.ScopeType = "comment"

	RoleAdministrator authorization.RoleKey = "administrator"
	RoleAuthor        authorization.RoleKey = "author"

	CapabilityPublicRead          authorization.CapabilityKey = "blog.public.read"
	CapabilityPostCreate          authorization.CapabilityKey = "blog.post.create"
	CapabilityPostRead            authorization.CapabilityKey = "blog.post.read"
	CapabilityPostUpdate          authorization.CapabilityKey = "blog.post.update"
	CapabilityPostPublish         authorization.CapabilityKey = "blog.post.publish"
	CapabilityPostArchive         authorization.CapabilityKey = "blog.post.archive"
	CapabilityPostDelete          authorization.CapabilityKey = "blog.post.delete"
	CapabilityPostFlagsManage     authorization.CapabilityKey = "blog.post_flags.manage"
	CapabilitySeriesCreate        authorization.CapabilityKey = "blog.series.create"
	CapabilitySeriesUpdate        authorization.CapabilityKey = "blog.series.update"
	CapabilitySeriesDelete        authorization.CapabilityKey = "blog.series.delete"
	CapabilityCommentModerate     authorization.CapabilityKey = "blog.comment.moderate"
	CapabilityCommentDelete       authorization.CapabilityKey = "blog.comment.delete"
	CapabilityTagCreate           authorization.CapabilityKey = "blog.tag.create"
	CapabilityTaxonomyManage      authorization.CapabilityKey = "blog.taxonomy.manage"
	CapabilitySiteSettingsManage  authorization.CapabilityKey = "blog.site_settings.manage"
	CapabilityAssetSettingsManage authorization.CapabilityKey = "blog.asset_settings.manage"

	RelationOwner authorization.RelationKind = "owner"

	ConstraintNormalRoleOwnsResource authorization.ConstraintKey = "blog.normal_role_owns_resource"
	PredicateRegistrationAuthor      authorization.PredicateKey  = "blog.registration_auto_author"
	TriggerUserRegistered            authorization.TriggerKey    = "identity.user.registered"
	AutomaticRegistrationAuthorKey                               = "blog.registration_author"
)

func PostScopeID(id string) authorization.ScopeID {
	return authorization.ScopeID("post:" + id)
}

func SeriesScopeID(id string) authorization.ScopeID {
	return authorization.ScopeID("series:" + id)
}

func CommentScopeID(id string) authorization.ScopeID {
	return authorization.ScopeID("comment:" + id)
}

func Definition() authorization.Definition {
	return authorization.Definition{
		Consumer: "blog",
		Version:  1,
		Capabilities: []authorization.CapabilityDefinition{
			{
				Key: CapabilityPublicRead, Version: 1,
				Binding:       authorization.BindingAccessLayerEligible,
				AllowedScopes: []authorization.ScopeType{ScopeSite, ScopePost, ScopeSeries, ScopeComment},
			},
			normalCapability(CapabilityPostCreate, ScopeSite),
			ownedCapability(CapabilityPostRead, ScopeSite, ScopePost),
			ownedCapability(CapabilityPostUpdate, ScopePost),
			ownedCapability(CapabilityPostPublish, ScopePost),
			ownedCapability(CapabilityPostArchive, ScopePost),
			ownedCapability(CapabilityPostDelete, ScopePost),
			protectedCapability(CapabilityPostFlagsManage),
			normalCapability(CapabilitySeriesCreate, ScopeSite),
			ownedCapability(CapabilitySeriesUpdate, ScopeSeries),
			ownedCapability(CapabilitySeriesDelete, ScopeSeries),
			ownedCapability(CapabilityCommentModerate, ScopeSite, ScopeComment),
			ownedCapability(CapabilityCommentDelete, ScopeSite, ScopeComment),
			normalCapability(CapabilityTagCreate, ScopeSite),
			protectedCapability(CapabilityTaxonomyManage),
			protectedCapability(CapabilitySiteSettingsManage),
			protectedCapability(CapabilityAssetSettingsManage),
		},
		Scopes: authorization.ScopeSchema{Types: []authorization.ScopeTypeDefinition{
			{Key: ScopeSite, Root: true, Children: []authorization.ScopeType{ScopePost, ScopeSeries}},
			{Key: ScopePost, Children: []authorization.ScopeType{ScopeComment}},
			{Key: ScopeSeries},
			{Key: ScopeComment},
		}},
		AccessLayers: []authorization.AccessLayerDefinition{
			{
				Key:          authorization.AccessLayerVisitor,
				Capabilities: []authorization.CapabilityKey{CapabilityPublicRead},
			},
			{
				Key: authorization.AccessLayerAuthenticated,
				Capabilities: []authorization.CapabilityKey{
					authorization.CapabilityApplicationCreate,
					authorization.CapabilityApplicationReadOwn,
					authorization.CapabilityApplicationWithdraw,
					authorization.CapabilityInvitationAccept,
				},
			},
		},
		Roles: []authorization.RoleDefinition{
			{
				Key: RoleAdministrator, DisplayName: "管理员", Protected: true,
				Capabilities: []authorization.CapabilityKey{
					authorization.CapabilityManage,
					authorization.CapabilityAuditRead,
					CapabilityPostCreate,
					CapabilityPostRead,
					CapabilityPostUpdate,
					CapabilityPostPublish,
					CapabilityPostArchive,
					CapabilityPostDelete,
					CapabilityPostFlagsManage,
					CapabilitySeriesCreate,
					CapabilitySeriesUpdate,
					CapabilitySeriesDelete,
					CapabilityCommentModerate,
					CapabilityCommentDelete,
					CapabilityTagCreate,
					CapabilityTaxonomyManage,
					CapabilitySiteSettingsManage,
					CapabilityAssetSettingsManage,
				},
			},
			{
				Key: RoleAuthor, DisplayName: "作者",
				Capabilities: []authorization.CapabilityKey{
					CapabilityPostCreate,
					CapabilityPostRead,
					CapabilityPostUpdate,
					CapabilityPostPublish,
					CapabilityPostArchive,
					CapabilityPostDelete,
					CapabilitySeriesCreate,
					CapabilitySeriesUpdate,
					CapabilitySeriesDelete,
					CapabilityCommentModerate,
					CapabilityCommentDelete,
					CapabilityTagCreate,
				},
				Assignment: authorization.AssignmentPolicy{Sources: []authorization.GrantSource{
					authorization.GrantSourceApplication,
					authorization.GrantSourceInvitation,
					authorization.GrantSourceDirect,
					authorization.GrantSourceAutomatic,
					authorization.GrantSourceGroup,
				}},
			},
		},
		Constraints: []authorization.ConstraintDefinition{{
			Key: ConstraintNormalRoleOwnsResource, Version: 1,
			Mode: authorization.ConstraintSource,
			Capabilities: []authorization.CapabilityKey{
				CapabilityPostRead,
				CapabilityPostUpdate,
				CapabilityPostPublish,
				CapabilityPostArchive,
				CapabilityPostDelete,
				CapabilitySeriesUpdate,
				CapabilitySeriesDelete,
				CapabilityCommentModerate,
				CapabilityCommentDelete,
			},
			AllNormalRoles: true,
		}},
		Automatic: []authorization.AutomaticRuleDefinition{{
			Key: AutomaticRegistrationAuthorKey, Trigger: TriggerUserRegistered,
			Predicate: PredicateRegistrationAuthor, Role: RoleAuthor, Enabled: false,
		}},
	}
}

func ConstraintEvaluators() map[authorization.ConstraintKey]authorization.ConstraintEvaluator {
	return map[authorization.ConstraintKey]authorization.ConstraintEvaluator{
		ConstraintNormalRoleOwnsResource: authorization.ConstraintFunc(
			func(_ context.Context, input authorization.ConstraintInput) authorization.ConstraintResult {
				for _, owner := range input.Resource.Relations[RelationOwner] {
					if owner == input.Subject {
						return authorization.ConstraintResult{}
					}
				}
				return authorization.ConstraintResult{Denied: true}
			},
		),
	}
}

func PredicateEvaluators() map[authorization.PredicateKey]authorization.PredicateEvaluator {
	return map[authorization.PredicateKey]authorization.PredicateEvaluator{
		PredicateRegistrationAuthor: authorization.PredicateFunc(
			func(_ context.Context, input authorization.PredicateInput) bool {
				return input.Subject.Kind == authorization.SubjectUser
			},
		),
	}
}

func normalCapability(
	key authorization.CapabilityKey,
	scopes ...authorization.ScopeType,
) authorization.CapabilityDefinition {
	return authorization.CapabilityDefinition{
		Key: key, Version: 1, Binding: authorization.BindingNormal,
		AllowedScopes: scopes,
		EligibleSubjects: []authorization.SubjectKind{
			authorization.SubjectUser,
			authorization.SubjectService,
		},
		Delegable: true,
	}
}

func ownedCapability(
	key authorization.CapabilityKey,
	scopes ...authorization.ScopeType,
) authorization.CapabilityDefinition {
	definition := normalCapability(key, scopes...)
	definition.QueryableRelation = RelationOwner
	return definition
}

func protectedCapability(key authorization.CapabilityKey) authorization.CapabilityDefinition {
	return authorization.CapabilityDefinition{
		Key: key, Version: 1, Binding: authorization.BindingProtectedOnly,
		Risk: authorization.RiskHigh, Audit: authorization.AuditFull,
		AllowedScopes: []authorization.ScopeType{ScopeSite},
		EligibleSubjects: []authorization.SubjectKind{
			authorization.SubjectUser,
			authorization.SubjectService,
		},
	}
}
