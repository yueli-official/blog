package blogabuse

import (
	"net/netip"
	"strings"
	"time"

	"github.com/yueli-official/foundation/go/abuse"
)

const (
	ActionAnonymousComment abuse.ActionKey = "blog.comment.create.anonymous"
	ActionMemberComment    abuse.ActionKey = "blog.comment.create.member"
)

type Policy struct {
	AnonymousCapacity int64
	Window            time.Duration
	Challenge         *abuse.ChallengeDefinition
}

func Definition(policy Policy) abuse.Definition {
	if policy.AnonymousCapacity <= 0 {
		policy.AnonymousCapacity = 5
	}
	if policy.Window <= 0 {
		policy.Window = time.Minute
	}
	challengeAt := int64(0)
	if policy.Challenge != nil && policy.AnonymousCapacity > 1 {
		challengeAt = policy.AnonymousCapacity
	}
	return abuse.Definition{
		Version: 1, Consumer: "blog",
		Actions: []abuse.ActionDefinition{
			{
				Key:      ActionAnonymousComment,
				Required: abuse.SignalRequirements{Network: abuse.Required},
				Meters: []abuse.MeterDefinition{{
					ID: "blog.comment.anonymous.network", Slot: abuse.SlotNetwork,
					Algorithm:   abuse.SlidingWindow(policy.AnonymousCapacity, policy.Window),
					ChallengeAt: challengeAt,
				}},
				Challenge: policy.Challenge,
			},
			{
				Key: ActionMemberComment,
				Required: abuse.SignalRequirements{
					Network: abuse.Required, Actor: abuse.Required,
				},
				Meters: []abuse.MeterDefinition{
					{
						ID: "blog.comment.member.network", Slot: abuse.SlotNetwork,
						Algorithm: abuse.TokenBucket(
							policy.AnonymousCapacity*4,
							policy.AnonymousCapacity*4,
							policy.Window,
						),
					},
					{
						ID: "blog.comment.member.actor", Slot: abuse.SlotActor,
						Algorithm: abuse.SlidingWindow(
							policy.AnonymousCapacity*2, policy.Window,
						),
					},
				},
			},
		},
	}
}

type Actions struct {
	Anonymous abuse.Action
	Member    abuse.Action
}

func Bind(module abuse.Module) (Actions, error) {
	anonymous, err := module.Action(ActionAnonymousComment)
	if err != nil {
		return Actions{}, err
	}
	member, err := module.Action(ActionMemberComment)
	if err != nil {
		return Actions{}, err
	}
	return Actions{Anonymous: anonymous, Member: member}, nil
}

func NetworkPrefix(value string) (netip.Prefix, error) {
	address, err := netip.ParseAddr(strings.TrimSpace(value))
	if err != nil {
		return netip.Prefix{}, err
	}
	address = address.Unmap()
	bits := address.BitLen()
	if address.Is6() {
		bits = 64
	}
	return netip.PrefixFrom(address, bits).Masked(), nil
}
