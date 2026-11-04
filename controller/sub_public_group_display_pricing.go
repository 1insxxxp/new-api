package controller

import (
	"fmt"
	"math"
	"os"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
)

const publicGroupDisplayPricingOption = "PublicGroupDisplayPricing"
const publicGroupBillingModelsOption = "PublicGroupSyncBillingModels"

func (p PublicGroupSyncModel) hasSourcePricing() bool {
	return p.SourceGroupID != 0 || p.SourceModel != "" || p.SourceGroupRatio != nil
}

func (p PublicGroupSyncModel) displayPrice() (model.PublicGroupDisplayPrice, error) {
	result := model.PublicGroupDisplayPrice{BillingMode: strings.ToLower(strings.TrimSpace(p.BillingMode))}
	if result.BillingMode == "" {
		result.BillingMode = "token"
	}
	if p.SourceGroupID <= 0 || strings.TrimSpace(p.SourceModel) == "" || p.SourceGroupRatio == nil {
		return result, fmt.Errorf("incomplete source pricing")
	}
	rate := *p.SourceGroupRatio
	if rate < 0 || math.IsNaN(rate) || math.IsInf(rate, 0) {
		return result, fmt.Errorf("invalid source group ratio")
	}
	fields := []struct {
		raw    *float64
		target **float64
	}{
		{p.InputPrice, &result.InputPrice}, {p.OutputPrice, &result.OutputPrice},
		{p.ImageInputPrice, &result.ImageInputPrice}, {p.ImageOutputPrice, &result.ImageOutputPrice},
		{p.PerRequestPrice, &result.PerRequestPrice}, {p.CacheReadPrice, &result.CacheReadPrice},
		{p.CacheWritePrice, &result.CacheWritePrice},
	}
	for _, field := range fields {
		if field.raw == nil {
			continue
		}
		value := *field.raw * rate
		if *field.raw < 0 || math.IsNaN(*field.raw) || math.IsInf(*field.raw, 0) || math.IsNaN(value) || math.IsInf(value, 0) {
			return result, fmt.Errorf("invalid source price")
		}
		*field.target = &value
	}
	return result, nil
}

// projectPublicGroupPricing only changes this GET /api/pricing response. It
// never changes the shared pricing cache, abilities, token permissions or routes.
func projectPublicGroupPricing(pricing []model.Pricing, usable map[string]string) ([]model.Pricing, error) {
	common.OptionMapRWMutex.RLock()
	raw := common.OptionMap[publicGroupDisplayPricingOption]
	common.OptionMapRWMutex.RUnlock()
	display := map[string]map[string]model.PublicGroupDisplayPrice{}
	if raw != "" {
		if err := common.UnmarshalJsonStr(raw, &display); err != nil {
			return nil, err
		}
	}
	hiddenIDs := map[int]bool{}
	for _, value := range strings.Split(os.Getenv("PUBLIC_GROUP_SYNC_HIDDEN_CHANNEL_IDS"), ",") {
		if strings.TrimSpace(value) == "" {
			continue
		}
		id, err := strconv.Atoi(strings.TrimSpace(value))
		if err != nil || id <= 0 {
			return nil, fmt.Errorf("invalid PUBLIC_GROUP_SYNC_HIDDEN_CHANNEL_IDS")
		}
		hiddenIDs[id] = true
	}
	type association struct{ group, model string }
	hidden := map[association]bool{}
	visible := map[association]bool{}
	if len(hiddenIDs) > 0 {
		abilities, err := model.GetAllEnableAbilityWithChannels()
		if err != nil {
			return nil, err
		}
		hiddenGroups, visibleGroups := map[string]bool{}, map[string]bool{}
		for _, ability := range abilities {
			pair := association{ability.Group, ability.Model}
			if hiddenIDs[ability.ChannelId] {
				hidden[pair] = true
				hiddenGroups[ability.Group] = true
			} else {
				visible[pair] = true
				visibleGroups[ability.Group] = true
			}
		}
		for group := range hiddenGroups {
			if !visibleGroups[group] {
				delete(usable, group)
			}
		}
	}
	result := make([]model.Pricing, 0, len(pricing))
	for _, row := range pricing {
		row.EnableGroup = append([]string(nil), row.EnableGroup...)
		groups := row.EnableGroup[:0]
		for _, group := range row.EnableGroup {
			pair := association{group, row.ModelName}
			if hidden[pair] && !visible[pair] {
				continue
			}
			if _, ok := usable[group]; !ok && group != "all" {
				continue
			}
			groups = append(groups, group)
		}
		row.EnableGroup = groups
		if len(groups) == 0 {
			continue
		}
		row.GroupPricing = nil
		originalGroupPrices := row.GroupPrices
		row.GroupPrices = make(map[string]float64)
		for _, group := range groups {
			if price, ok := display[group][row.ModelName]; ok {
				if row.GroupPricing == nil {
					row.GroupPricing = map[string]model.PublicGroupDisplayPrice{}
				}
				row.GroupPricing[group] = price
			}
			// Copy only visible fixed overrides; do not mutate the cached map.
			if price, ok := originalGroupPrices[group]; ok {
				row.GroupPrices[group] = price
			}
		}
		if common.StringsContains(groups, "all") {
			for group, price := range originalGroupPrices {
				if _, ok := usable[group]; ok {
					row.GroupPrices[group] = price
				}
			}
		}
		result = append(result, row)
	}
	return result, nil
}
