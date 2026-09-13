package ratio_setting

import (
	"encoding/json"
	"sync"

	"github.com/samber/lo"
)

// GroupModelPrice stores fixed pricing overrides for one model in one group.
// The group is a lookup context for the model plaza; it is never part of the
// model identifier sent to an upstream API.
type GroupModelPrice struct {
	Price float64 `json:"price"`
}

var (
	groupModelPriceMu sync.RWMutex
	groupModelPrices  = make(map[string]map[string]float64)
)

func GroupModelPrice2JSONString() string {
	groupModelPriceMu.RLock()
	defer groupModelPriceMu.RUnlock()
	b, _ := json.Marshal(groupModelPrices)
	return string(b)
}

func UpdateGroupModelPriceByJSONString(value string) error {
	updated := make(map[string]map[string]float64)
	if err := json.Unmarshal([]byte(value), &updated); err != nil {
		return err
	}
	groupModelPriceMu.Lock()
	groupModelPrices = updated
	groupModelPriceMu.Unlock()
	InvalidateExposedDataCache()
	return nil
}

func GetGroupModelPriceForModel(model string) map[string]float64 {
	groupModelPriceMu.RLock()
	defer groupModelPriceMu.RUnlock()
	var result map[string]float64
	for group, models := range groupModelPrices {
		if price, ok := models[model]; ok {
			if result == nil {
				result = make(map[string]float64)
			}
			result[group] = price
		}
	}
	return result
}

func GetGroupModelPriceCopy() map[string]map[string]float64 {
	groupModelPriceMu.RLock()
	defer groupModelPriceMu.RUnlock()
	result := make(map[string]map[string]float64, len(groupModelPrices))
	for group, models := range groupModelPrices {
		result[group] = lo.Assign(models)
	}
	return result
}
