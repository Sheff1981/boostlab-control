package api

type TierPolicy struct {
	AdsEnabled        bool `json:"ads_enabled"`
	MaxAutoCandidates int  `json:"max_auto_candidates"`
	PriorityRouting   bool `json:"priority_routing"`
}

type ClientPolicy struct {
	DefaultTier string     `json:"default_tier"`
	Free        TierPolicy `json:"free"`
	Premium     TierPolicy `json:"premium"`
}

func DefaultClientPolicy() ClientPolicy {
	return ClientPolicy{
		DefaultTier: "free",
		Free: TierPolicy{
			AdsEnabled:        true,
			MaxAutoCandidates: 8,
			PriorityRouting:   false,
		},
		Premium: TierPolicy{
			AdsEnabled:        false,
			MaxAutoCandidates: 8,
			PriorityRouting:   true,
		},
	}
}
