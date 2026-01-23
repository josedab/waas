package billing

// CalculateFlatRate calculates cost for a flat rate plan.
func CalculateFlatRate(plan *PricingPlan, usage int64) int64 {
	return plan.BasePrice
}

// CalculatePerEvent calculates cost for a per-event plan.
// Base price plus overage for events exceeding the included amount.
func CalculatePerEvent(plan *PricingPlan, usage int64) int64 {
	overage := CalculateOverage(plan, usage)
	if overage <= 0 {
		return plan.BasePrice
	}
	// Price is per 1000 events
	overageBlocks := (overage + 999) / 1000
	return plan.BasePrice + overageBlocks*plan.OveragePrice
}

// CalculateTiered calculates cost using tiered pricing.
// Each tier covers a range of usage at a specific per-unit price.
func CalculateTiered(tiers []PricingTier, usage int64) int64 {
	if len(tiers) == 0 || usage <= 0 {
		return 0
	}

	var total int64
	remaining := usage
	prevUpTo := int64(0)

	for _, tier := range tiers {
		if remaining <= 0 {
			break
		}

		var tierSize int64
		if tier.UpTo < 0 {
			// Unlimited tier — consume all remaining
			tierSize = remaining
		} else {
			tierSize = tier.UpTo - prevUpTo
		}

		used := remaining
		if used > tierSize {
			used = tierSize
		}

		total += used * tier.PricePerUnit
		remaining -= used

		if tier.UpTo >= 0 {
			prevUpTo = tier.UpTo
		}
	}

	return total
}

// CalculateOverage returns the number of events exceeding the plan's included events.
// Returns 0 if usage is within the included allocation.
func CalculateOverage(plan *PricingPlan, usage int64) int64 {
	if usage <= plan.IncludedEvents {
		return 0
	}
	return usage - plan.IncludedEvents
}
