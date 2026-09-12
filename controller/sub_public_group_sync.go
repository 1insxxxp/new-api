package controller

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
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
	secret := os.Getenv("PUBLIC_GROUP_SYNC_SECRET")
	if secret == "" {
		c.Status(http.StatusNotFound)
		return
	}
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
	if err := json.Unmarshal(body, &envelope); err != nil || envelope.Version != PublicGroupSyncSnapshotVersion {
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
	for _, existing := range allChannels {
		isSynced := strings.HasPrefix(existing.GetTag(), publicGroupSyncTagPrefix)
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
	newSyncedModels := make(map[string]struct{})
	prices := ratio_setting.GetModelPriceMap()
	ratios := ratio_setting.GetModelRatioCopy()
	completionRatios := ratio_setting.GetCompletionRatioCopy()
	cacheRatios := ratio_setting.GetCacheRatioCopy()
	createCacheRatios := ratio_setting.GetCreateCacheRatioCopy()
	for _, snapshot := range envelope.Snapshots {
		if snapshot.GroupID == 0 || snapshot.Version != PublicGroupSyncSnapshotVersion {
			c.JSON(400, gin.H{"error": "invalid group_id"})
			return
		}
		for name, pricing := range snapshot.ModelPricing {
			mode := strings.ToLower(strings.TrimSpace(pricing.BillingMode))
			if mode != "" && mode != "token" && mode != "per_request" && mode != "image" && mode != "video" {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid billing mode for " + name})
				return
			}
		}
		tag := publicGroupSyncTagPrefix + strconv.FormatInt(snapshot.GroupID, 10)
		seen[tag] = true
		for _, name := range snapshot.Models {
			if strings.TrimSpace(name) != "" {
				newSyncedModels[name] = struct{}{}
			}
		}
		mapping, _ := json.Marshal(snapshot.ModelMapping)
		ch := &model.Channel{Key: "sub-public-group-sync", Name: snapshot.GroupName, Group: snapshot.GroupName, Models: strings.Join(snapshot.Models, ","), ModelMapping: stringPtr(string(mapping)), Tag: stringPtr(tag), Type: publicGroupSyncChannelType(snapshot), Status: common.ChannelStatusManuallyDisabled}
		if snapshot.PublicEnabled && len(snapshot.Models) > 0 {
			ch.Status = common.ChannelStatusEnabled
		}
		found, findErr := model.GetChannelsByTag(tag, false, true)
		if findErr != nil {
			c.JSON(500, gin.H{"error": findErr.Error()})
			return
		}
		if len(found) > 0 {
			ch.Id = found[0].Id
			ch.Key = found[0].Key
			if err := ch.Update(); err != nil {
				c.JSON(500, gin.H{"error": err.Error()})
				return
			}
		} else if err := ch.Insert(); err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		for name, pricing := range snapshot.ModelPricing {
			mode := strings.ToLower(strings.TrimSpace(pricing.BillingMode))
			if mode == "" {
				mode = "token"
			}
			if mode == "per_request" || mode == "image" || mode == "video" {
				if pricing.PerRequestPrice != nil {
					prices[name] = *pricing.PerRequestPrice
				}
				delete(ratios, name)
				delete(completionRatios, name)
				delete(cacheRatios, name)
				delete(createCacheRatios, name)
				continue
			}
			if pricing.InputPrice != nil {
				// New API's token ratio unit is $0.002 per 1K input tokens;
				// Sub stores token prices as USD per token.
				ratios[name] = *pricing.InputPrice * 500000
				delete(prices, name)
			}
			if pricing.InputPrice != nil && pricing.OutputPrice != nil && *pricing.InputPrice > 0 {
				completionRatios[name] = *pricing.OutputPrice / *pricing.InputPrice
			}
			if pricing.CacheReadPrice != nil && pricing.InputPrice != nil && *pricing.InputPrice > 0 {
				cacheRatios[name] = *pricing.CacheReadPrice / *pricing.InputPrice
			}
			if pricing.CacheWritePrice != nil && pricing.InputPrice != nil && *pricing.InputPrice > 0 {
				createCacheRatios[name] = *pricing.CacheWritePrice / *pricing.InputPrice
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
	if b, err := json.Marshal(prices); err == nil {
		_ = ratio_setting.UpdateModelPriceByJSONString(string(b))
	}
	if b, err := json.Marshal(ratios); err == nil {
		_ = ratio_setting.UpdateModelRatioByJSONString(string(b))
	}
	if b, err := json.Marshal(completionRatios); err == nil {
		_ = ratio_setting.UpdateCompletionRatioByJSONString(string(b))
	}
	if b, err := json.Marshal(cacheRatios); err == nil {
		_ = ratio_setting.UpdateCacheRatioByJSONString(string(b))
	}
	if b, err := json.Marshal(createCacheRatios); err == nil {
		_ = ratio_setting.UpdateCreateCacheRatioByJSONString(string(b))
	}
	model.InitChannelCache()
	c.JSON(http.StatusOK, gin.H{"ok": true})
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
