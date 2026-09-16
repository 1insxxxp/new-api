package controller

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting"
	"github.com/QuantumNous/new-api/setting/billing_setting"
	"github.com/QuantumNous/new-api/setting/config"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupPublicGroupSyncTest(t *testing.T) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", strings.ReplaceAll(t.Name(), "/", "_"))
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.Channel{}, &model.Ability{}, &model.Option{}, &model.Model{}, &model.Vendor{}))

	originalDB, originalLogDB := model.DB, model.LOG_DB
	originalMemoryCache := common.MemoryCacheEnabled
	originalOptionMap := common.OptionMap
	originalModelPrice := ratio_setting.ModelPrice2JSONString()
	originalModelRatio := ratio_setting.ModelRatio2JSONString()
	originalCompletionRatio := ratio_setting.CompletionRatio2JSONString()
	originalCacheRatio := ratio_setting.CacheRatio2JSONString()
	originalCreateCacheRatio := ratio_setting.CreateCacheRatio2JSONString()
	originalGroupRatio := ratio_setting.GroupRatio2JSONString()
	originalGroupModelPrice := ratio_setting.GroupModelPrice2JSONString()
	originalUsableGroups := setting.UserUsableGroups2JSONString()
	originalBillingModes, err := json.Marshal(billing_setting.GetBillingModeCopy())
	require.NoError(t, err)
	originalSecret, secretWasSet := os.LookupEnv("PUBLIC_GROUP_SYNC_SECRET")
	model.DB, model.LOG_DB = db, db
	common.MemoryCacheEnabled = false
	common.OptionMap = make(map[string]string)
	require.NoError(t, os.Setenv("PUBLIC_GROUP_SYNC_SECRET", "test-secret"))
	t.Cleanup(func() {
		model.InvalidatePricingCache()
		model.DB, model.LOG_DB = originalDB, originalLogDB
		common.MemoryCacheEnabled = originalMemoryCache
		common.OptionMap = originalOptionMap
		_ = ratio_setting.UpdateModelPriceByJSONString(originalModelPrice)
		_ = ratio_setting.UpdateModelRatioByJSONString(originalModelRatio)
		_ = ratio_setting.UpdateCompletionRatioByJSONString(originalCompletionRatio)
		_ = ratio_setting.UpdateCacheRatioByJSONString(originalCacheRatio)
		_ = ratio_setting.UpdateCreateCacheRatioByJSONString(originalCreateCacheRatio)
		_ = ratio_setting.UpdateGroupRatioByJSONString(originalGroupRatio)
		_ = ratio_setting.UpdateGroupModelPriceByJSONString(originalGroupModelPrice)
		_ = setting.UpdateUserUsableGroupsByJSONString(originalUsableGroups)
		_ = config.UpdateConfigFromMap(config.GlobalConfig.Get("billing_setting"), map[string]string{"billing_mode": string(originalBillingModes)})
		if secretWasSet {
			_ = os.Setenv("PUBLIC_GROUP_SYNC_SECRET", originalSecret)
		} else {
			_ = os.Unsetenv("PUBLIC_GROUP_SYNC_SECRET")
		}
		sqlDB, _ := db.DB()
		if sqlDB != nil {
			_ = sqlDB.Close()
		}
	})
}

func TestReceivePublicGroupSyncSnapshotPersistsChannelsPricesAndGroups(t *testing.T) {
	setupPublicGroupSyncTest(t)
	db := model.DB

	inputPrice := 0.000004
	outputPrice := 0.000012
	imagePrice := 0.04
	envelope := publicGroupSyncEnvelope{Version: PublicGroupSyncSnapshotVersion, Snapshots: []PublicGroupSyncRequest{{
		Version:       PublicGroupSyncSnapshotVersion,
		GroupID:       42,
		GroupName:     " synced-group\t",
		PublicEnabled: true,
		GroupRatio:    2,
		Models:        []string{"sync-model", "sync-image-model"},
		ModelMapping:  map[string]string{"sync-model": "upstream-model"},
		ModelPricing: map[string]PublicGroupSyncModel{
			"sync-model": {
				Platform: "openai", DisplayName: "sync-model", UpstreamModel: "upstream-model", BillingMode: "token",
				InputPrice: &inputPrice, OutputPrice: &outputPrice,
			},
			"sync-image-model": {
				Platform: "openai", DisplayName: "sync-image-model", BillingMode: "image",
				PerRequestPrice: &imagePrice,
			}},
	}}}
	body, err := json.Marshal(envelope)
	require.NoError(t, err)
	legacyTag := "sub-public-group:42"
	require.NoError(t, db.Create(&model.Channel{
		Key: "real-upstream-key", Name: "old-name", Group: "old-name", Models: "sync-model",
		Tag: &legacyTag, Status: common.ChannelStatusEnabled,
	}).Error)
	require.NoError(t, db.Create(&model.Channel{
		Key: "sub-public-group-sync", Name: "mirror-name", Group: "mirror-name", Models: "sync-model",
		Tag: &legacyTag, Status: common.ChannelStatusEnabled,
	}).Error)
	status := postPublicGroupSyncTestRequest(t, body, "test-secret")
	require.Equal(t, http.StatusOK, status)

	var channel model.Channel
	require.NoError(t, db.Where("tag = ?", "sub-public-group:42").First(&channel).Error)
	require.Equal(t, "synced-group", channel.Name)
	require.Equal(t, "synced-group", channel.Group)
	require.Equal(t, common.ChannelStatusEnabled, channel.Status)
	require.Contains(t, channel.Models, "sync-image-model")
	var syncedAbility model.Ability
	require.NoError(t, db.Where("model = ?", "sync-image-model").First(&syncedAbility).Error)
	require.Equal(t, "synced-group", syncedAbility.Group)
	var duplicate model.Channel
	require.NoError(t, db.Where("key = ?", "sub-public-group-sync").First(&duplicate).Error)
	require.Equal(t, common.ChannelStatusManuallyDisabled, duplicate.Status)

	var ratioOption, groupOption, usableOption model.Option
	require.NoError(t, db.First(&ratioOption, "key = ?", "ModelRatio").Error)
	require.NoError(t, db.First(&groupOption, "key = ?", "GroupRatio").Error)
	require.NoError(t, db.First(&usableOption, "key = ?", "UserUsableGroups").Error)
	require.Contains(t, ratioOption.Value, "sync-model")
	require.NotContains(t, ratioOption.Value, "sync-image-model")
	require.Contains(t, groupOption.Value, "synced-group")
	require.Contains(t, usableOption.Value, "synced-group")
	var billingModeOption model.Option
	require.NoError(t, db.First(&billingModeOption, "key = ?", "billing_setting.billing_mode").Error)
	require.Contains(t, billingModeOption.Value, "sync-image-model")
	require.Contains(t, billingModeOption.Value, "image")
	var groupModelPriceOption model.Option
	require.NoError(t, db.First(&groupModelPriceOption, "key = ?", "GroupModelPrice").Error)
	var groupModelPrices map[string]map[string]float64
	require.NoError(t, json.Unmarshal([]byte(groupModelPriceOption.Value), &groupModelPrices))
	require.Equal(t, imagePrice, groupModelPrices["synced-group"]["sync-image-model"])

	emptyBody, err := json.Marshal(publicGroupSyncEnvelope{Version: PublicGroupSyncSnapshotVersion, Snapshots: []PublicGroupSyncRequest{}})
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, postPublicGroupSyncTestRequest(t, emptyBody, "test-secret"))
	require.NoError(t, db.First(&channel, channel.Id).Error)
	require.Equal(t, common.ChannelStatusManuallyDisabled, channel.Status)
	require.NoError(t, db.First(&groupOption, "key = ?", "GroupRatio").Error)
	require.NotContains(t, groupOption.Value, "synced-group")
	require.NoError(t, db.First(&billingModeOption, "key = ?", "billing_setting.billing_mode").Error)
	require.NotContains(t, billingModeOption.Value, "sync-image-model")
	require.NoError(t, db.First(&groupModelPriceOption, "key = ?", "GroupModelPrice").Error)
	require.NotContains(t, groupModelPriceOption.Value, "sync-image-model")
}

func TestReceivePublicGroupSyncPreservesTokenAndRequestPricingAcrossGroups(t *testing.T) {
	for _, reverse := range []bool{false, true} {
		t.Run(fmt.Sprintf("request_first=%t", reverse), func(t *testing.T) {
			setupPublicGroupSyncTest(t)
			input, output, cache, requestPrice := 0.000003, 0.000009, 0.0000001, 0.15
			tokenGroup := PublicGroupSyncRequest{
				Version: PublicGroupSyncSnapshotVersion, GroupID: 71, GroupName: "token-group",
				PublicEnabled: true, GroupRatio: 1, Models: []string{"shared-model"},
				ModelPricing: map[string]PublicGroupSyncModel{"shared-model": {
					Platform: "openai", BillingMode: "token", InputPrice: &input,
					OutputPrice: &output, CacheReadPrice: &cache,
				}},
			}
			requestGroup := PublicGroupSyncRequest{
				Version: PublicGroupSyncSnapshotVersion, GroupID: 128, GroupName: "request-group",
				PublicEnabled: true, GroupRatio: 1, Models: []string{"shared-model"},
				ModelPricing: map[string]PublicGroupSyncModel{"shared-model": {
					Platform: "openai", BillingMode: "per_request", PerRequestPrice: &requestPrice,
				}},
			}
			snapshots := []PublicGroupSyncRequest{tokenGroup, requestGroup}
			if reverse {
				snapshots[0], snapshots[1] = snapshots[1], snapshots[0]
			}
			body, err := common.Marshal(publicGroupSyncEnvelope{Version: PublicGroupSyncSnapshotVersion, Snapshots: snapshots})
			require.NoError(t, err)
			for range 2 {
				require.Equal(t, http.StatusOK, postPublicGroupSyncTestRequest(t, body, "test-secret"))
				pricing := model.GetPricing()
				require.Len(t, pricing, 1)
				assert.Equal(t, 0, pricing[0].QuotaType)
				assert.InDelta(t, 1.5, pricing[0].ModelRatio, 1e-10)
				assert.InDelta(t, 3, pricing[0].CompletionRatio, 1e-10)
				require.NotNil(t, pricing[0].CacheRatio)
				assert.InDelta(t, 1.0/30, *pricing[0].CacheRatio, 1e-10)
				assert.Equal(t, map[string]float64{"request-group": 0.15}, pricing[0].GroupPrices)
				assert.ElementsMatch(t, []string{"token-group", "request-group"}, pricing[0].EnableGroup)
			}

			// Removing the token group must restore request-only pricing.
			body, err = common.Marshal(publicGroupSyncEnvelope{Version: PublicGroupSyncSnapshotVersion, Snapshots: []PublicGroupSyncRequest{requestGroup}})
			require.NoError(t, err)
			require.Equal(t, http.StatusOK, postPublicGroupSyncTestRequest(t, body, "test-secret"))
			pricing := model.GetPricing()
			require.Len(t, pricing, 1)
			assert.Equal(t, 1, pricing[0].QuotaType)
			assert.Equal(t, 0.15, pricing[0].ModelPrice)
			assert.Equal(t, []string{"request-group"}, pricing[0].EnableGroup)
		})
	}
}

func TestSelectPublicGroupSyncChannelPrefersExistingConfiguredChannel(t *testing.T) {
	legacyKey := "real-upstream-key"
	mirrorKey := "sub-public-group-sync"
	legacy := &model.Channel{Id: 17, Key: legacyKey, Status: common.ChannelStatusEnabled}
	mirror := &model.Channel{Id: 36, Key: mirrorKey, Status: common.ChannelStatusEnabled}

	canonical, duplicates := selectPublicGroupSyncChannel([]*model.Channel{mirror, legacy})

	require.Same(t, legacy, canonical)
	require.Len(t, duplicates, 1)
	require.Same(t, mirror, duplicates[0])
}

func TestPublicGroupSyncUpdatesPricingGroupOrder(t *testing.T) {
	setupPublicGroupSyncTest(t)
	require.NoError(t, setting.UpdateUserUsableGroupsByJSONString(`{}`))
	for _, test := range []struct {
		name, snapshots string
		want            []string
	}{
		{"configured order and ID tie break", `[
			{"version":1,"group_id":3,"group_name":"alpha","sort_order":20},
			{"version":1,"group_id":8,"group_name":"beta","sort_order":10},
			{"version":1,"group_id":2,"group_name":" zeta ","sort_order":10}]`, []string{"zeta", "beta", "alpha"}},
		{"reorder rename and remove", `[
			{"version":1,"group_id":3,"group_name":"alpha","sort_order":0},
			{"version":1,"group_id":8,"group_name":"renamed","sort_order":5}]`, []string{"alpha", "renamed"}},
		{"legacy snapshot uses ID order", `[
			{"version":1,"group_id":8,"group_name":"alpha"},
			{"version":1,"group_id":3,"group_name":"zeta"}]`, []string{"zeta", "alpha"}},
	} {
		t.Run(test.name, func(t *testing.T) {
			body := []byte(`{"version":1,"snapshots":` + test.snapshots + `}`)
			require.Equal(t, http.StatusOK, postPublicGroupSyncTestRequest(t, body, "test-secret"))
			recorder := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(recorder)
			ctx.Request = httptest.NewRequest(http.MethodGet, "/api/pricing", nil)
			GetPricing(ctx)
			require.Equal(t, http.StatusOK, recorder.Code)
			var response struct {
				GroupOrder []string `json:"group_order"`
			}
			require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
			assert.Equal(t, test.want, response.GroupOrder)
		})
	}
}

func TestPricingGroupOrderOmitsUnavailableAndAppendsLocalGroups(t *testing.T) {
	setupPublicGroupSyncTest(t)
	require.NoError(t, model.UpdateOptionsBulk(map[string]string{
		"PublicGroupOrder": `["private","zeta","zeta","alpha"]`,
	}))
	assert.Equal(t, []string{"zeta", "alpha", "local"}, getPricingGroupOrder(map[string]string{
		"alpha": "alpha", "local": "local", "zeta": "zeta",
	}))
	assert.Empty(t, getPricingGroupOrder(map[string]string{}))
}

func TestReceivePublicGroupSyncImagePricingWinsAcrossGroups(t *testing.T) {
	// This regression test covers a model that is listed by both an image group
	// and a normal group. The image billing mode must remain authoritative.
	gin.SetMode(gin.TestMode)
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", strings.ReplaceAll(t.Name(), "/", "_"))
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.Channel{}, &model.Ability{}, &model.Option{}))

	originalDB, originalLogDB := model.DB, model.LOG_DB
	originalMemoryCache := common.MemoryCacheEnabled
	originalOptionMap := common.OptionMap
	originalModelPrice := ratio_setting.ModelPrice2JSONString()
	originalModelRatio := ratio_setting.ModelRatio2JSONString()
	originalCompletionRatio := ratio_setting.CompletionRatio2JSONString()
	originalCacheRatio := ratio_setting.CacheRatio2JSONString()
	originalCreateCacheRatio := ratio_setting.CreateCacheRatio2JSONString()
	originalGroupRatio := ratio_setting.GroupRatio2JSONString()
	originalGroupModelPrice := ratio_setting.GroupModelPrice2JSONString()
	originalUsableGroups := setting.UserUsableGroups2JSONString()
	originalBillingModes, err := json.Marshal(billing_setting.GetBillingModeCopy())
	require.NoError(t, err)
	originalSecret, secretWasSet := os.LookupEnv("PUBLIC_GROUP_SYNC_SECRET")
	model.DB, model.LOG_DB = db, db
	common.MemoryCacheEnabled = false
	common.OptionMap = make(map[string]string)
	require.NoError(t, os.Setenv("PUBLIC_GROUP_SYNC_SECRET", "test-secret"))
	t.Cleanup(func() {
		model.DB, model.LOG_DB = originalDB, originalLogDB
		common.MemoryCacheEnabled = originalMemoryCache
		common.OptionMap = originalOptionMap
		_ = ratio_setting.UpdateModelPriceByJSONString(originalModelPrice)
		_ = ratio_setting.UpdateModelRatioByJSONString(originalModelRatio)
		_ = ratio_setting.UpdateCompletionRatioByJSONString(originalCompletionRatio)
		_ = ratio_setting.UpdateCacheRatioByJSONString(originalCacheRatio)
		_ = ratio_setting.UpdateCreateCacheRatioByJSONString(originalCreateCacheRatio)
		_ = ratio_setting.UpdateGroupRatioByJSONString(originalGroupRatio)
		_ = ratio_setting.UpdateGroupModelPriceByJSONString(originalGroupModelPrice)
		_ = setting.UpdateUserUsableGroupsByJSONString(originalUsableGroups)
		_ = config.UpdateConfigFromMap(config.GlobalConfig.Get("billing_setting"), map[string]string{"billing_mode": string(originalBillingModes)})
		if secretWasSet {
			_ = os.Setenv("PUBLIC_GROUP_SYNC_SECRET", originalSecret)
		} else {
			_ = os.Unsetenv("PUBLIC_GROUP_SYNC_SECRET")
		}
		sqlDB, _ := db.DB()
		if sqlDB != nil {
			_ = sqlDB.Close()
		}
	})

	imagePrice := 0.04
	image := PublicGroupSyncModel{Platform: "openai", DisplayName: "shared-image-model", BillingMode: "image", PerRequestPrice: &imagePrice}
	normal := PublicGroupSyncModel{Platform: "openai", DisplayName: "shared-image-model", BillingMode: "token"}
	envelope := publicGroupSyncEnvelope{Version: PublicGroupSyncSnapshotVersion, Snapshots: []PublicGroupSyncRequest{
		{Version: PublicGroupSyncSnapshotVersion, GroupID: 1, GroupName: "image-group", PublicEnabled: true, Models: []string{"shared-image-model"}, ModelPricing: map[string]PublicGroupSyncModel{"shared-image-model": image}},
		{Version: PublicGroupSyncSnapshotVersion, GroupID: 2, GroupName: "normal-group", PublicEnabled: true, Models: []string{"shared-image-model"}, ModelPricing: map[string]PublicGroupSyncModel{"shared-image-model": normal}},
	}}
	body, err := json.Marshal(envelope)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, postPublicGroupSyncTestRequest(t, body, "test-secret"))

	require.Equal(t, billing_setting.BillingModeImage, billing_setting.GetBillingMode("shared-image-model"))
	price, ok := ratio_setting.GetModelPrice("shared-image-model", false)
	require.True(t, ok)
	require.Equal(t, imagePrice, price)
}

func postPublicGroupSyncTestRequest(t *testing.T, body []byte, secret string) int {
	t.Helper()
	timestamp := fmt.Sprintf("%d", time.Now().Unix())
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(timestamp + "\n"))
	_, _ = mac.Write(body)
	req := httptest.NewRequest(http.MethodPost, "/api/internal/sub-public-group-sync/snapshot", bytes.NewReader(body))
	req.Header.Set("X-Public-Group-Sync-Timestamp", timestamp)
	req.Header.Set("X-Public-Group-Sync-Signature", hex.EncodeToString(mac.Sum(nil)))
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = req
	ReceivePublicGroupSyncSnapshot(ctx)
	return recorder.Code
}
