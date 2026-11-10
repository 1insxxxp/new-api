package controller

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPublicGroupInheritedPricingNeverChangesGlobalBilling(t *testing.T) {
	setupPublicGroupSyncTest(t)
	require.NoError(t, ratio_setting.UpdateModelRatioByJSONString(`{"existing-token":7}`))
	require.NoError(t, ratio_setting.UpdateModelPriceByJSONString(`{"existing-fixed":0.8}`))
	require.NoError(t, ratio_setting.UpdateCompletionRatioByJSONString(`{"existing-token":3}`))
	require.NoError(t, ratio_setting.UpdateCacheRatioByJSONString(`{"existing-token":0.2}`))
	require.NoError(t, ratio_setting.UpdateCreateCacheRatioByJSONString(`{"existing-token":1.5}`))
	// An existing price need not have an enabled independent channel to be preserved.
	body := []byte(`{"version":2,"snapshots":[{"version":2,"group_id":200,"group_name":"category","group_ratio":1,"public_enabled":true,"models":["existing-token","existing-fixed","free"],"model_pricing":{
		"existing-token":{"billing_mode":"token","source_group_id":71,"source_model":"alias","source_group_ratio":2,"input_price":0.000003,"output_price":0.000009,"cache_read_price":0.000001},
		"existing-fixed":{"billing_mode":"per_request","source_group_id":128,"source_model":"fixed","source_group_ratio":3,"per_request_price":0.2},
		"free":{"billing_mode":"token","source_group_id":71,"source_model":"zero","source_group_ratio":0,"input_price":0.000003,"output_price":0.000009}
	}}]}`)
	for range 2 {
		require.Equal(t, http.StatusOK, postPublicGroupSyncTestRequest(t, body, "test-secret"))
		assert.Equal(t, map[string]float64{"existing-token": 7}, ratio_setting.GetModelRatioCopy())
		assert.Equal(t, map[string]float64{"existing-fixed": 0.8}, ratio_setting.GetModelPriceMap())
		assert.Equal(t, map[string]float64{"existing-token": 3}, ratio_setting.GetCompletionRatioCopy())
		assert.Equal(t, map[string]float64{"existing-token": 0.2}, ratio_setting.GetCacheRatioCopy())
		assert.Equal(t, map[string]float64{"existing-token": 1.5}, ratio_setting.GetCreateCacheRatioCopy())
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	GetPricing(c)
	var response struct {
		Data []struct {
			ModelName    string `json:"model_name"`
			GroupPricing map[string]struct {
				InputPrice      *float64 `json:"input_price"`
				OutputPrice     *float64 `json:"output_price"`
				PerRequestPrice *float64 `json:"per_request_price"`
			} `json:"group_pricing"`
		} `json:"data"`
	}
	require.NoError(t, common.Unmarshal(w.Body.Bytes(), &response))
	require.Len(t, response.Data, 3)
	for _, row := range response.Data {
		price, ok := row.GroupPricing["category"]
		require.True(t, ok)
		switch row.ModelName {
		case "existing-token":
			require.NotNil(t, price.InputPrice)
			assert.InDelta(t, 0.000006, *price.InputPrice, 1e-15)
			require.NotNil(t, price.OutputPrice)
			assert.InDelta(t, 0.000018, *price.OutputPrice, 1e-15)
		case "existing-fixed":
			require.NotNil(t, price.PerRequestPrice)
			assert.InDelta(t, 0.6, *price.PerRequestPrice, 1e-15)
		case "free":
			require.NotNil(t, price.InputPrice)
			assert.Zero(t, *price.InputPrice)
		}
	}
	for range 2 {
		require.Equal(t, http.StatusOK, postPublicGroupSyncTestRequest(t, []byte(`{"version":2,"snapshots":[]}`), "test-secret"))
		assert.Equal(t, map[string]float64{"existing-token": 7}, ratio_setting.GetModelRatioCopy())
		assert.Equal(t, map[string]float64{"existing-fixed": 0.8}, ratio_setting.GetModelPriceMap())
	}
}

func TestPublicGroupInheritedPricingRetainsFormerOrdinaryBilling(t *testing.T) {
	setupPublicGroupSyncTest(t)
	ordinary := []byte(`{"version":2,"snapshots":[{"version":2,"group_id":71,"group_name":"ordinary","group_ratio":1,"models":["token","fixed"],"model_pricing":{
		"token":{"input_price":0.000004,"output_price":0.000012,"cache_read_price":0.000001,"cache_write_price":0.000006},
		"fixed":{"billing_mode":"per_request","per_request_price":0.8}
	}}]}`)
	require.Equal(t, http.StatusOK, postPublicGroupSyncTestRequest(t, ordinary, "test-secret"))
	ratios := ratio_setting.GetModelRatioCopy()
	prices := ratio_setting.GetModelPriceMap()
	completion := ratio_setting.GetCompletionRatioCopy()
	cache := ratio_setting.GetCacheRatioCopy()
	createCache := ratio_setting.GetCreateCacheRatioCopy()
	inherited := []byte(`{"version":2,"snapshots":[{"version":2,"group_id":200,"group_name":"category","group_ratio":1,"models":["token","fixed"],"model_pricing":{
		"token":{"source_group_id":71,"source_model":"token","source_group_ratio":2,"input_price":0.000003,"output_price":0.000009},
		"fixed":{"source_group_id":128,"source_model":"fixed","source_group_ratio":3,"billing_mode":"per_request","per_request_price":0.2}
	}}]}`)
	for _, body := range [][]byte{inherited, inherited, []byte(`{"version":2,"snapshots":[]}`)} {
		require.Equal(t, http.StatusOK, postPublicGroupSyncTestRequest(t, body, "test-secret"))
		assert.Equal(t, ratios, ratio_setting.GetModelRatioCopy())
		assert.Equal(t, prices, ratio_setting.GetModelPriceMap())
		assert.Equal(t, completion, ratio_setting.GetCompletionRatioCopy())
		assert.Equal(t, cache, ratio_setting.GetCacheRatioCopy())
		assert.Equal(t, createCache, ratio_setting.GetCreateCacheRatioCopy())
	}
}

func TestPublicGroupInheritedImageDoesNotOverrideOrdinaryTokenPricing(t *testing.T) {
	setupPublicGroupSyncTest(t)
	body := []byte(`{"version":2,"snapshots":[
		{"version":2,"group_id":71,"group_name":"normal","group_ratio":2,"public_enabled":true,"models":["shared"],"model_pricing":{"shared":{"billing_mode":"token","input_price":0.000004,"output_price":0.000012}}},
		{"version":2,"group_id":200,"group_name":"category","group_ratio":1,"public_enabled":true,"models":["shared"],"model_pricing":{"shared":{"billing_mode":"image","source_group_id":99,"source_model":"image","source_group_ratio":3,"per_request_price":0.2}}}
	]}`)
	require.Equal(t, http.StatusOK, postPublicGroupSyncTestRequest(t, body, "test-secret"))
	ratio, ok, _ := ratio_setting.GetModelRatio("shared")
	assert.True(t, ok)
	assert.Equal(t, 2.0, ratio)
	_, priced := ratio_setting.GetModelPrice("shared", false)
	assert.False(t, priced)
	rows := model.GetPricing()
	require.Len(t, rows, 1)
	assert.Equal(t, 0, rows[0].QuotaType)
	assert.Empty(t, rows[0].BillingMode)
}

func TestPublicGroupInheritedPricingRejectsIncompleteProvenanceBeforeWrites(t *testing.T) {
	setupPublicGroupSyncTest(t)
	for _, pricing := range []string{
		`{"source_group_id":71,"source_model":"model","input_price":0.1}`,
		`{"source_group_id":71,"source_group_ratio":1,"input_price":0.1}`,
		`{"source_group_id":71,"source_model":"model","source_group_ratio":-1,"input_price":0.1}`,
		`{"source_group_id":71,"source_model":"model","source_group_ratio":1e300,"input_price":1e300}`,
	} {
		body := []byte(`{"version":2,"snapshots":[{"version":2,"group_id":200,"group_name":"category","group_ratio":1,"models":["model"],"model_pricing":{"model":` + pricing + `}}]}`)
		assert.Equal(t, http.StatusBadRequest, postPublicGroupSyncTestRequest(t, body, "test-secret"))
	}
	var count int64
	require.NoError(t, model.DB.Model(&model.Channel{}).Count(&count).Error)
	assert.Zero(t, count)
}

func TestPublicGroupSyncVersionTwoRequiresInheritedPricingAndAcceptsOrdinaryLegacy(t *testing.T) {
	setupPublicGroupSyncTest(t)
	legacy := []byte(`{"version":1,"snapshots":[{"version":1,"group_id":71,"group_name":"legacy","group_ratio":1,"models":["legacy-model"],"model_pricing":{"legacy-model":{"input_price":0.000002}}}]}`)
	require.Equal(t, http.StatusOK, postPublicGroupSyncTestRequest(t, legacy, "test-secret"))
	ratio, ok, _ := ratio_setting.GetModelRatio("legacy-model")
	assert.True(t, ok)
	assert.Equal(t, 1.0, ratio)
	for _, body := range []string{
		`{"version":1,"snapshots":[{"version":1,"group_id":200,"group_name":"category","group_ratio":1,"models":["legacy-model"],"model_pricing":{"legacy-model":{"source_group_id":71,"source_model":"source","source_group_ratio":2,"input_price":0.1}}}]}`,
		`{"version":2,"snapshots":[{"version":1,"group_id":200,"group_name":"category","group_ratio":1}]}`,
		`{"version":1,"snapshots":[{"version":2,"group_id":200,"group_name":"category","group_ratio":1}]}`,
	} {
		assert.Equal(t, http.StatusBadRequest, postPublicGroupSyncTestRequest(t, []byte(body), "test-secret"))
	}
	ratio, _, _ = ratio_setting.GetModelRatio("legacy-model")
	assert.Equal(t, 1.0, ratio)
	var count int64
	require.NoError(t, model.DB.Model(&model.Channel{}).Count(&count).Error)
	assert.Equal(t, int64(1), count)
}

func TestPublicGroupPricingHidesOnlySelectedChannelAssociations(t *testing.T) {
	setupPublicGroupSyncTest(t)
	t.Setenv("PUBLIC_GROUP_SYNC_HIDDEN_CHANNEL_IDS", "20,30")
	require.NoError(t, setting.UpdateUserUsableGroupsByJSONString(`{"shared-group":"shared-group","legacy":"legacy"}`))
	require.NoError(t, ratio_setting.UpdateGroupRatioByJSONString(`{"shared-group":1,"legacy":2}`))
	for _, channel := range []*model.Channel{
		{Id: 20, Key: "retained-key", Name: "hidden", Group: "shared-group", Models: "shared,hidden-only", Status: common.ChannelStatusEnabled},
		{Id: 21, Key: "visible-key", Name: "visible", Group: "shared-group", Models: "shared,visible-only", Status: common.ChannelStatusEnabled},
		{Id: 30, Key: "legacy-key", Name: "legacy", Group: "legacy", Models: "legacy-only", Status: common.ChannelStatusEnabled},
	} {
		require.NoError(t, channel.Insert())
	}
	model.InvalidatePricingCache()
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	GetPricing(c)
	require.Equal(t, http.StatusOK, w.Code)
	var response struct {
		Data   []model.Pricing    `json:"data"`
		Usable map[string]string  `json:"usable_group"`
		Ratios map[string]float64 `json:"group_ratio"`
		Order  []string           `json:"group_order"`
	}
	require.NoError(t, common.Unmarshal(w.Body.Bytes(), &response))
	var names []string
	for _, row := range response.Data {
		names = append(names, row.ModelName)
	}
	assert.ElementsMatch(t, []string{"shared", "visible-only"}, names)
	assert.Equal(t, map[string]string{"shared-group": "shared-group"}, response.Usable)
	assert.Equal(t, map[string]float64{"shared-group": 1}, response.Ratios)
	assert.Equal(t, []string{"shared-group"}, response.Order)
	// Public model cache and relay abilities still contain the original route.
	assert.Len(t, model.GetPricing(), 4)
	var retainedAbility model.Ability
	require.NoError(t, model.DB.Where(&model.Ability{Group: "legacy", Model: "legacy-only", ChannelId: 30}).First(&retainedAbility).Error)
	assert.True(t, retainedAbility.Enabled)
	assert.Contains(t, setting.GetUserUsableGroupsCopy(), "legacy")
	assert.Equal(t, 2.0, ratio_setting.GetGroupRatio("legacy"))
	var channel model.Channel
	require.NoError(t, model.DB.First(&channel, 20).Error)
	assert.Equal(t, "retained-key", channel.Key)
	assert.Equal(t, "shared,hidden-only", channel.Models)
	assert.Equal(t, common.ChannelStatusEnabled, channel.Status)
}

func TestPublicGroupDisplayPricePreservesIndependentZeroAndImagePrices(t *testing.T) {
	zero, output, imageOutput, rate := 0.0, 0.000004, 0.000009, 2.0
	p := PublicGroupSyncModel{SourceGroupID: 71, SourceModel: "source", SourceGroupRatio: &rate,
		InputPrice: &zero, OutputPrice: &output, ImageInputPrice: &zero, ImageOutputPrice: &imageOutput}
	quote, err := p.displayPrice()
	require.NoError(t, err)
	require.NotNil(t, quote.InputPrice)
	assert.Zero(t, *quote.InputPrice)
	require.NotNil(t, quote.OutputPrice)
	assert.Equal(t, 0.000008, *quote.OutputPrice)
	require.NotNil(t, quote.ImageInputPrice)
	assert.Zero(t, *quote.ImageInputPrice)
	require.NotNil(t, quote.ImageOutputPrice)
	assert.Equal(t, 0.000018, *quote.ImageOutputPrice)
	p.SourceGroupRatio = &zero
	quote, err = p.displayPrice()
	require.NoError(t, err)
	require.NotNil(t, quote.OutputPrice)
	assert.Zero(t, *quote.OutputPrice)
	require.NotNil(t, quote.ImageOutputPrice)
	assert.Zero(t, *quote.ImageOutputPrice)
}
