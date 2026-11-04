package controller

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting"
	"github.com/QuantumNous/new-api/setting/billing_setting"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
	"github.com/gin-gonic/gin"
)

type publicGroupSyncEnvelope struct {
	Version   int                      `json:"version"`
	Snapshots []PublicGroupSyncRequest `json:"snapshots"`
}

const publicGroupSyncTagPrefix = "sub-public-group:"

func publicGroupSyncSignature(secret, timestamp string, body []byte) string {
	h := hmac.New(sha256.New, []byte(secret))
	h.Write([]byte(timestamp))
	h.Write([]byte("\n"))
	h.Write(body)
	return hex.EncodeToString(h.Sum(nil))
}

func ReceivePublicGroupSyncSnapshot(c *gin.Context) {
	secret := strings.TrimSpace(os.Getenv("PUBLIC_GROUP_SYNC_SECRET"))
	if secret == "" {
		c.Status(http.StatusNotFound)
		return
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 4<<20)
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(400, gin.H{"error": "invalid body"})
		return
	}
	ts := c.GetHeader("X-Public-Group-Sync-Timestamp")
	sig := c.GetHeader("X-Public-Group-Sync-Signature")
	stamp, err := strconv.ParseInt(ts, 10, 64)
	if err != nil || time.Now().Unix()-stamp > 300 || stamp-time.Now().Unix() > 300 || !hmac.Equal([]byte(strings.ToLower(sig)), []byte(publicGroupSyncSignature(secret, ts, body))) {
		c.JSON(401, gin.H{"error": "invalid signature"})
		return
	}
	var envelope publicGroupSyncEnvelope
	if err := json.Unmarshal(body, &envelope); err != nil || (envelope.Version != 1 && envelope.Version != PublicGroupSyncSnapshotVersion) {
		c.JSON(400, gin.H{"error": "invalid snapshot"})
		return
	}
	seen := map[string]bool{}
	allChannels, err := model.GetAllChannels(0, 0, true, false)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "list channels: " + err.Error()})
		return
	}
	oldSyncedModels := make(map[string]struct{})
	protectedModels := make(map[string]struct{})
	oldSyncedGroups := make(map[string]struct{})
	protectedGroups := make(map[string]struct{})
	for _, existing := range allChannels {
		isSynced := strings.HasPrefix(existing.GetTag(), publicGroupSyncTagPrefix)
		for _, group := range strings.Split(existing.Group, ",") {
			group = strings.TrimSpace(group)
			if group == "" {
				continue
			}
			if isSynced {
				oldSyncedGroups[group] = struct{}{}
			} else if existing.Status == common.ChannelStatusEnabled {
				protectedGroups[group] = struct{}{}
			}
		}
		for _, name := range strings.Split(existing.Models, ",") {
			name = strings.TrimSpace(name)
			if name == "" {
				continue
			}
			if isSynced {
				oldSyncedModels[name] = struct{}{}
			} else if existing.Status == common.ChannelStatusEnabled {
				protectedModels[name] = struct{}{}
			}
		}
	}
	// Only ordinary snapshots own global billing keys. Channel.Models also
	// contains display-only inherited models, so it cannot be used for cleanup
	// after this receiver has persisted its first explicit ownership inventory.
	common.OptionMapRWMutex.RLock()
	previousBillingModels, hasBillingInventory := common.OptionMap[publicGroupBillingModelsOption]
	common.OptionMapRWMutex.RUnlock()
	if hasBillingInventory {
		var names []string
		if err := common.UnmarshalJsonStr(previousBillingModels, &names); err != nil {
			c.JSON(500, gin.H{"error": "invalid previous billing inventory"})
			return
		}
		oldSyncedModels = make(map[string]struct{}, len(names))
		for _, name := range names {
			oldSyncedModels[name] = struct{}{}
		}
	}
	newSyncedModels := make(map[string]struct{})
	newSyncedPricedModels := make(map[string]struct{})
	// An inherited entry never authorizes deleting an existing billing key,
	// including when its ordinary source disappeared from this snapshot.
	// Drop old ownership so later removal of the display entry is also harmless.
	for _, snapshot := range envelope.Snapshots {
		for name, pricing := range snapshot.ModelPricing {
			if pricing.hasSourcePricing() {
				delete(oldSyncedModels, name)
			}
		}
	}
	// A model can be exposed by more than one public group. Image pricing has
	// precedence over token/request pricing so a later ordinary group cannot
	// downgrade an image model back to the default billing mode.
	imageModels := make(map[string]struct{})
	tokenModels := make(map[string]struct{})
	for _, snapshot := range envelope.Snapshots {
		for name, pricing := range snapshot.ModelPricing {
			if pricing.hasSourcePricing() {
				continue
			}
			mode := strings.ToLower(strings.TrimSpace(pricing.BillingMode))
			if mode == billing_setting.BillingModeImage {
				imageModels[name] = struct{}{}
			}
			if (mode == "" || mode == "token") && pricing.InputPrice != nil {
				tokenModels[name] = struct{}{}
			}
		}
	}
	prices := ratio_setting.GetModelPriceMap()
	ratios := ratio_setting.GetModelRatioCopy()
	completionRatios := ratio_setting.GetCompletionRatioCopy()
	cacheRatios := ratio_setting.GetCacheRatioCopy()
	createCacheRatios := ratio_setting.GetCreateCacheRatioCopy()
	groupRatios := ratio_setting.GetGroupRatioCopy()
	billingModes := billing_setting.GetBillingModeCopy()
	usableGroups := setting.GetUserUsableGroupsCopy()
	groupModelPrices := make(map[string]map[string]float64)
	displayPricing := make(map[string]map[string]model.PublicGroupDisplayPrice)
	newSyncedGroups := make(map[string]struct{})
	for _, snapshot := range envelope.Snapshots {
		if snapshot.GroupID == 0 || snapshot.Version != envelope.Version {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid group_id"})
			return
		}
		if strings.TrimSpace(snapshot.GroupName) == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "empty group name"})
			return
		}
		for name, pricing := range snapshot.ModelPricing {
			mode := strings.ToLower(strings.TrimSpace(pricing.BillingMode))
			if mode != "" && mode != "token" && mode != "per_request" && mode != "image" && mode != "video" {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid billing mode for " + name})
				return
			}
			if pricing.hasSourcePricing() {
				quote, err := pricing.displayPrice()
				if err != nil || envelope.Version != PublicGroupSyncSnapshotVersion || snapshot.GroupRatio != 1 || !common.StringsContains(snapshot.Models, name) {
					c.JSON(400, gin.H{"error": "invalid source pricing for " + name})
					return
				}
				if snapshot.PublicEnabled {
					group := strings.TrimSpace(snapshot.GroupName)
					if displayPricing[group] == nil {
						displayPricing[group] = make(map[string]model.PublicGroupDisplayPrice)
					}
					displayPricing[group][name] = quote
				}
			}
		}
	}
	for _, snapshot := range envelope.Snapshots {
		for _, name := range snapshot.Models {
			if strings.TrimSpace(name) != "" {
				if pricing, priced := snapshot.ModelPricing[name]; !priced || pricing.hasSourcePricing() {
					continue
				}
				newSyncedPricedModels[name] = struct{}{}
			}
		}
		tag := publicGroupSyncTagPrefix + strconv.FormatInt(snapshot.GroupID, 10)
		seen[tag] = true
		groupName := strings.TrimSpace(snapshot.GroupName)
		newSyncedGroups[groupName] = struct{}{}
		groupRatios[groupName] = snapshot.GroupRatio
		usableGroups[groupName] = groupName
		for name, pricing := range snapshot.ModelPricing {
			if pricing.hasSourcePricing() {
				continue
			}
			mode := strings.ToLower(strings.TrimSpace(pricing.BillingMode))
			if (mode == billing_setting.BillingModeImage || mode == "per_request") && pricing.PerRequestPrice != nil {
				if groupModelPrices[groupName] == nil {
					groupModelPrices[groupName] = make(map[string]float64)
				}
				groupModelPrices[groupName][name] = *pricing.PerRequestPrice
			}
		}
		for _, name := range snapshot.Models {
			if strings.TrimSpace(name) != "" && !snapshot.ModelPricing[name].hasSourcePricing() {
				newSyncedModels[name] = struct{}{}
			}
		}
		mapping, _ := json.Marshal(snapshot.ModelMapping)
		ch := &model.Channel{Key: "sub-public-group-sync", Name: groupName, Group: groupName, Models: strings.Join(snapshot.Models, ","), ModelMapping: stringPtr(string(mapping)), Tag: stringPtr(tag), Type: publicGroupSyncChannelType(snapshot), Status: common.ChannelStatusManuallyDisabled}
		if snapshot.PublicEnabled && len(snapshot.Models) > 0 {
			ch.Status = common.ChannelStatusEnabled
		}
		found, findErr := model.GetChannelsByTag(tag, false, true)
		if findErr != nil {
			c.JSON(500, gin.H{"error": findErr.Error()})
			return
		}
		if len(found) > 0 {
			canonical, duplicates := selectPublicGroupSyncChannel(found)
			ch.Id = canonical.Id
			ch.Key = canonical.Key
			if err := ch.Update(); err != nil {
				c.JSON(500, gin.H{"error": err.Error()})
				return
			}
			for _, duplicate := range duplicates {
				duplicate.Status = common.ChannelStatusManuallyDisabled
				if err := duplicate.Update(); err != nil {
					c.JSON(500, gin.H{"error": "disable duplicate channel: " + err.Error()})
					return
				}
			}
		} else if err := ch.Insert(); err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		for name, pricing := range snapshot.ModelPricing {
			if pricing.hasSourcePricing() {
				continue
			}
			mode := strings.ToLower(strings.TrimSpace(pricing.BillingMode))
			if mode == "" {
				mode = "token"
			}
			if mode != billing_setting.BillingModeImage {
				if _, image := imageModels[name]; image {
					continue
				}
			}
			if mode == billing_setting.BillingModeImage {
				if pricing.PerRequestPrice != nil {
					prices[name] = *pricing.PerRequestPrice
				} else {
					delete(prices, name)
				}
				billingModes[name] = billing_setting.BillingModeImage
				delete(ratios, name)
				delete(completionRatios, name)
				delete(cacheRatios, name)
				delete(createCacheRatios, name)
				continue
			}
			if mode == "per_request" || mode == "video" {
				// Keep token prices for shared models; fixed prices are stored per
				// group above so snapshot order cannot change their billing mode.
				if _, token := tokenModels[name]; token {
					continue
				}
				if pricing.PerRequestPrice != nil {
					prices[name] = *pricing.PerRequestPrice
				} else {
					delete(prices, name)
				}
				delete(ratios, name)
				delete(completionRatios, name)
				delete(cacheRatios, name)
				delete(createCacheRatios, name)
				delete(billingModes, name)
				continue
			}
			delete(billingModes, name)
			delete(prices, name)
			if pricing.InputPrice != nil {
				// New API's token ratio unit is $0.002 per 1K input tokens;
				// Sub stores token prices as USD per token.
				ratios[name] = *pricing.InputPrice * 500000
			} else {
				delete(ratios, name)
			}
			if pricing.InputPrice != nil && pricing.OutputPrice != nil && *pricing.InputPrice > 0 {
				completionRatios[name] = *pricing.OutputPrice / *pricing.InputPrice
			} else {
				delete(completionRatios, name)
			}
			if pricing.CacheReadPrice != nil && pricing.InputPrice != nil && *pricing.InputPrice > 0 {
				cacheRatios[name] = *pricing.CacheReadPrice / *pricing.InputPrice
			} else {
				delete(cacheRatios, name)
			}
			if pricing.CacheWritePrice != nil && pricing.InputPrice != nil && *pricing.InputPrice > 0 {
				createCacheRatios[name] = *pricing.CacheWritePrice / *pricing.InputPrice
			} else {
				delete(createCacheRatios, name)
			}
		}
	}
	for name := range oldSyncedModels {
		if _, stillSynced := newSyncedModels[name]; stillSynced {
			continue
		}
		if _, protected := protectedModels[name]; protected {
			continue
		}
		delete(prices, name)
		delete(ratios, name)
		delete(completionRatios, name)
		delete(cacheRatios, name)
		delete(createCacheRatios, name)
		delete(billingModes, name)
	}
	for name := range newSyncedModels {
		if _, priced := newSyncedPricedModels[name]; priced {
			continue
		}
		if _, protected := protectedModels[name]; protected {
			continue
		}
		delete(prices, name)
		delete(ratios, name)
		delete(completionRatios, name)
		delete(cacheRatios, name)
		delete(createCacheRatios, name)
	}
	for group := range oldSyncedGroups {
		if _, stillSynced := newSyncedGroups[group]; stillSynced {
			continue
		}
		if _, protected := protectedGroups[group]; protected {
			continue
		}
		delete(groupRatios, group)
		delete(usableGroups, group)
	}
	for _, ch := range allChannels {
		if !strings.HasPrefix(ch.GetTag(), publicGroupSyncTagPrefix) {
			continue
		}
		if !seen[ch.GetTag()] && ch.Status != common.ChannelStatusManuallyDisabled {
			ch.Status = common.ChannelStatusManuallyDisabled
			if err := ch.Update(); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "disable stale channel: " + err.Error()})
				return
			}
		}
	}
	billingModesJSON, err := json.Marshal(billingModes)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "marshal billing modes: " + err.Error()})
		return
	}
	options := make(map[string]string, 8)
	// Keep display ordering separate from the pricing application order.
	orderedSnapshots := append([]PublicGroupSyncRequest(nil), envelope.Snapshots...)
	sort.SliceStable(orderedSnapshots, func(i, j int) bool {
		if orderedSnapshots[i].SortOrder != orderedSnapshots[j].SortOrder {
			return orderedSnapshots[i].SortOrder < orderedSnapshots[j].SortOrder
		}
		return orderedSnapshots[i].GroupID < orderedSnapshots[j].GroupID
	})
	groupOrder := make([]string, 0, len(orderedSnapshots))
	for _, snapshot := range orderedSnapshots {
		groupOrder = append(groupOrder, strings.TrimSpace(snapshot.GroupName))
	}
	billingModels := make([]string, 0, len(newSyncedModels))
	for name := range newSyncedModels {
		billingModels = append(billingModels, name)
	}
	sort.Strings(billingModels)
	for key, value := range map[string]any{
		publicGroupDisplayPricingOption: displayPricing,
		publicGroupBillingModelsOption:  billingModels,
		"PublicGroupOrder":              groupOrder,
		"ModelPrice":                    prices,
		"ModelRatio":                    ratios,
		"CompletionRatio":               completionRatios,
		"CacheRatio":                    cacheRatios,
		"CreateCacheRatio":              createCacheRatios,
		"GroupRatio":                    groupRatios,
		"UserUsableGroups":              usableGroups,
		"GroupModelPrice":               groupModelPrices,
	} {
		b, marshalErr := json.Marshal(value)
		if marshalErr != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "marshal option " + key + ": " + marshalErr.Error()})
			return
		}
		options[key] = string(b)
	}
	options["billing_setting.billing_mode"] = string(billingModesJSON)
	if err := model.UpdateOptionsBulk(options); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "persist pricing options: " + err.Error()})
		return
	}
	model.InitChannelCache()
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func selectPublicGroupSyncChannel(channels []*model.Channel) (*model.Channel, []*model.Channel) {
	ordered := append([]*model.Channel(nil), channels...)
	sort.SliceStable(ordered, func(i, j int) bool {
		iConfigured := strings.TrimSpace(ordered[i].Key) != "sub-public-group-sync"
		jConfigured := strings.TrimSpace(ordered[j].Key) != "sub-public-group-sync"
		if iConfigured != jConfigured {
			return iConfigured
		}
		return ordered[i].Id < ordered[j].Id
	})
	return ordered[0], ordered[1:]
}

func publicGroupSyncChannelType(snapshot PublicGroupSyncRequest) int {
	for _, pricing := range snapshot.ModelPricing {
		switch strings.ToLower(pricing.Platform) {
		case "anthropic", "claude":
			return constant.ChannelTypeAnthropic
		case "gemini", "google":
			return constant.ChannelTypeGemini
		case "openai":
			return constant.ChannelTypeOpenAI
		}
	}
	return constant.ChannelTypeOpenAI
}

func stringPtr(s string) *string { return &s }
