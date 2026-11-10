package controller

import (
	"fmt"
	"maps"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting"
	"github.com/QuantumNous/new-api/setting/billing_setting"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Compare persisted routes/options and live billing settings, not only the
// display response: accepting a quote must not change ordinary sync behavior.
type publicGroupSyncState struct {
	Channels      []model.Channel
	Abilities     []model.Ability
	Options       map[string]string
	CachedOptions map[string]string
	Billing       map[string]any
}

func capturePublicGroupSyncState(t *testing.T) publicGroupSyncState {
	t.Helper()
	state := publicGroupSyncState{Options: map[string]string{}}
	require.NoError(t, model.DB.Order("id").Find(&state.Channels).Error)
	require.NoError(t, model.DB.Order("channel_id, model").Find(&state.Abilities).Error)
	var options []model.Option
	require.NoError(t, model.DB.Find(&options).Error)
	for _, option := range options {
		state.Options[option.Key] = option.Value
	}
	common.OptionMapRWMutex.RLock()
	state.CachedOptions = maps.Clone(common.OptionMap)
	common.OptionMapRWMutex.RUnlock()
	state.Billing = map[string]any{
		"prices": ratio_setting.GetModelPriceMap(), "ratios": ratio_setting.GetModelRatioCopy(),
		"completion": ratio_setting.GetCompletionRatioCopy(), "cache": ratio_setting.GetCacheRatioCopy(),
		"create_cache": ratio_setting.GetCreateCacheRatioCopy(), "groups": ratio_setting.GetGroupRatioCopy(),
		"group_prices": ratio_setting.GetGroupModelPriceCopy(), "modes": billing_setting.GetBillingModeCopy(),
		"usable": setting.GetUserUsableGroupsCopy(),
	}
	return state
}

func TestPublicGroupExplicitDisplayPricingPreservesBillingAndClearsOnLegacySync(t *testing.T) {
	setupPublicGroupSyncTest(t)
	t.Setenv("PUBLIC_GROUP_SYNC_HIDDEN_CHANNEL_IDS", "")
	body := []byte(`{"version":2,"snapshots":[
		{"version":2,"group_id":71,"group_name":"token-group","group_ratio":2,"public_enabled":true,
		 "models":["shared","zero","missing","free"],"model_mapping":{"shared":"upstream"},
		 "model_pricing":{"shared":{"platform":"openai","billing_mode":"token","input_price":0.000004,"output_price":0.000012,"cache_read_price":0.000001,"cache_write_price":0.000006}},
		 "display_pricing":{
			"shared":{"platform":"anthropic","billing_mode":"token","source_group_id":71,"source_model":"shared","source_group_ratio":2,"input_price":0.000003,"output_price":0.000009,"cache_read_price":0.000001,"cache_write_price":0.000004},
			"zero":{"billing_mode":"token","source_group_id":71,"source_model":"zero","source_group_ratio":2,"input_price":0,"output_price":0.000004,"image_input_price":0,"image_output_price":0.000009,"cache_read_price":0},
			"missing":{"billing_mode":"token","source_group_id":71,"source_model":"missing","source_group_ratio":2},
			"free":{"billing_mode":"token","source_group_id":71,"source_model":"free","source_group_ratio":0,"input_price":0.000003,"output_price":0.000009}}},
		{"version":2,"group_id":128,"group_name":"image-independent","group_ratio":1.5,"public_enabled":true,"models":["shared"],
		 "model_pricing":{"shared":{"billing_mode":"image","per_request_price":0.04}},
		 "display_pricing":{"shared":{"billing_mode":"image","source_group_id":128,"source_model":"shared","source_group_ratio":1,"per_request_price":0.2}}},
		{"version":2,"group_id":129,"group_name":"image-group-rate","group_ratio":1.5,"public_enabled":true,"models":["shared"],
		 "model_pricing":{"shared":{"billing_mode":"image","per_request_price":0.07}},
		 "display_pricing":{"shared":{"billing_mode":"image","source_group_id":129,"source_model":"shared","source_group_ratio":1.5,"per_request_price":0.2}}}
	]}`)
	// The same ordinary payload without quotes is the billing/routing baseline.
	var legacy map[string]any
	require.NoError(t, common.Unmarshal(body, &legacy))
	for _, snapshot := range legacy["snapshots"].([]any) {
		delete(snapshot.(map[string]any), "display_pricing")
	}
	legacyBody, err := common.Marshal(legacy)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, postPublicGroupSyncTestRequest(t, legacyBody, "test-secret"))
	baseline := capturePublicGroupSyncState(t)
	for range 2 {
		require.Equal(t, http.StatusOK, postPublicGroupSyncTestRequest(t, body, "test-secret"))
		state := capturePublicGroupSyncState(t)
		var overlay map[string]map[string]model.PublicGroupDisplayPrice
		require.NoError(t, common.UnmarshalJsonStr(state.Options[publicGroupDisplayPricingOption], &overlay))
		require.Len(t, overlay, 3)
		// Every other persisted option (including the ownership inventory), live
		// billing map, channel, mapping, key and ability is exactly unchanged.
		state.Options[publicGroupDisplayPricingOption] = baseline.Options[publicGroupDisplayPricingOption]
		state.CachedOptions[publicGroupDisplayPricingOption] = baseline.CachedOptions[publicGroupDisplayPricingOption]
		assert.Equal(t, baseline, state)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		GetPricing(c)
		require.Equal(t, http.StatusOK, w.Code)
		var response struct {
			Data       []model.Pricing    `json:"data"`
			GroupRatio map[string]float64 `json:"group_ratio"`
		}
		require.NoError(t, common.Unmarshal(w.Body.Bytes(), &response))
		require.Len(t, response.Data, 4)
		rows := map[string]model.Pricing{}
		for _, row := range response.Data {
			rows[row.ModelName] = row
			for group, quote := range row.GroupPricing {
				assert.Equal(t, overlay[group][row.ModelName], quote)
			}
		}
		shared := rows["shared"].GroupPricing
		require.Len(t, shared, 3)
		assert.Equal(t, "token", shared["token-group"].BillingMode)
		require.NotNil(t, shared["token-group"].InputPrice)
		assert.InDelta(t, 0.000006, *shared["token-group"].InputPrice, 1e-15)
		require.NotNil(t, shared["token-group"].OutputPrice)
		assert.InDelta(t, 0.000018, *shared["token-group"].OutputPrice, 1e-15)
		require.NotNil(t, shared["token-group"].CacheReadPrice)
		assert.InDelta(t, 0.000002, *shared["token-group"].CacheReadPrice, 1e-15)
		require.NotNil(t, shared["token-group"].CacheWritePrice)
		assert.InDelta(t, 0.000008, *shared["token-group"].CacheWritePrice, 1e-15)
		for group, want := range map[string]float64{"image-independent": 0.2, "image-group-rate": 0.3} {
			assert.Equal(t, "image", shared[group].BillingMode)
			require.NotNil(t, shared[group].PerRequestPrice)
			assert.InDelta(t, want, *shared[group].PerRequestPrice, 1e-15)
			assert.Nil(t, shared[group].InputPrice)
			assert.Equal(t, 1.5, response.GroupRatio[group])
			assert.Equal(t, 1.5, ratio_setting.GetGroupRatio(group))
		}
		assert.Equal(t, 2.0, response.GroupRatio["token-group"])
		zero, output, imageOutput := 0.0, 0.000008, 0.000018
		assert.Equal(t, model.PublicGroupDisplayPrice{BillingMode: "token", InputPrice: &zero,
			OutputPrice: &output, ImageInputPrice: &zero, ImageOutputPrice: &imageOutput, CacheReadPrice: &zero}, rows["zero"].GroupPricing["token-group"])
		assert.Equal(t, model.PublicGroupDisplayPrice{BillingMode: "token"}, rows["missing"].GroupPricing["token-group"])
		assert.Equal(t, model.PublicGroupDisplayPrice{BillingMode: "token", InputPrice: &zero, OutputPrice: &zero}, rows["free"].GroupPricing["token-group"])
		for _, row := range model.GetPricing() {
			assert.Empty(t, row.GroupPricing, "response projection must not mutate the shared pricing cache")
		}
	}

	// Omitted, null and empty display maps remain compatible with both legacy
	// wire versions and remove any previous quote on the next full snapshot.
	for _, version := range []int{1, 2} {
		for _, empty := range []string{"omitted", "null", "empty"} {
			require.Equal(t, http.StatusOK, postPublicGroupSyncTestRequest(t, body, "test-secret"))
			legacy["version"] = version
			for _, raw := range legacy["snapshots"].([]any) {
				snapshot := raw.(map[string]any)
				snapshot["version"] = version
				delete(snapshot, "display_pricing")
				if empty == "null" {
					snapshot["display_pricing"] = nil
				} else if empty == "empty" {
					snapshot["display_pricing"] = map[string]any{}
				}
			}
			legacyBody, err = common.Marshal(legacy)
			require.NoError(t, err)
			require.Equal(t, http.StatusOK, postPublicGroupSyncTestRequest(t, legacyBody, "test-secret"))
			assert.Equal(t, baseline, capturePublicGroupSyncState(t), "version=%d map=%s", version, empty)
		}
	}
}

func TestPublicGroupPartialDisplayPricingPreservesLegacyAndUserGroupRatios(t *testing.T) {
	setupPublicGroupSyncTest(t)
	t.Setenv("PUBLIC_GROUP_SYNC_HIDDEN_CHANNEL_IDS", "")
	previousRedis := common.RedisEnabled
	previousUserRatios := ratio_setting.GroupGroupRatio2JSONString()
	common.RedisEnabled = false
	t.Cleanup(func() {
		common.RedisEnabled = previousRedis
		require.NoError(t, ratio_setting.UpdateGroupGroupRatioByJSONString(previousUserRatios))
	})
	require.NoError(t, model.DB.AutoMigrate(&model.User{}))
	user := model.User{Username: "pricing-viewer", Group: "vip"}
	require.NoError(t, model.DB.Create(&user).Error)
	require.NoError(t, ratio_setting.UpdateGroupGroupRatioByJSONString(`{"vip":{"ordinary":3,"category":4}}`))
	body := []byte(`{"version":2,"snapshots":[
		{"version":2,"group_id":71,"group_name":"ordinary","group_ratio":2,"public_enabled":true,"models":["quoted","legacy-only"],
		 "model_pricing":{"quoted":{"billing_mode":"token","input_price":0.000004},"legacy-only":{"billing_mode":"token","input_price":0.000004}},
		 "display_pricing":{"quoted":{"billing_mode":"token","source_group_id":71,"source_model":"quoted","source_group_ratio":2,"input_price":0.000003}}},
		{"version":2,"group_id":200,"group_name":"category","group_ratio":1,"public_enabled":true,"models":["quoted"],
		 "model_pricing":{"quoted":{"billing_mode":"token","source_group_id":99,"source_model":"alias","source_group_ratio":3,"input_price":0.000005}}}
	]}`)
	require.Equal(t, http.StatusOK, postPublicGroupSyncTestRequest(t, body, "test-secret"))
	before := capturePublicGroupSyncState(t)
	for _, signedIn := range []bool{false, true} {
		t.Run(fmt.Sprintf("signed_in=%t", signedIn), func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			ordinaryRatio, categoryRatio := 2.0, 1.0
			if signedIn {
				c.Set("id", user.Id)
				ordinaryRatio, categoryRatio = 3, 4
			}
			GetPricing(c)
			require.Equal(t, http.StatusOK, w.Code)
			var response struct {
				Data       []model.Pricing    `json:"data"`
				GroupRatio map[string]float64 `json:"group_ratio"`
			}
			require.NoError(t, common.Unmarshal(w.Body.Bytes(), &response))
			require.Len(t, response.Data, 2)
			assert.Equal(t, ordinaryRatio, response.GroupRatio["ordinary"])
			assert.Equal(t, categoryRatio, response.GroupRatio["category"])
			for _, row := range response.Data {
				if row.ModelName == "legacy-only" {
					assert.Empty(t, row.GroupPricing)
					assert.Equal(t, 2.0, row.ModelRatio)
					assert.Equal(t, 0, row.QuotaType)
					continue
				}
				require.Len(t, row.GroupPricing, 2)
				for group, want := range map[string]float64{"ordinary": 0.000006, "category": 0.000015} {
					quote := row.GroupPricing[group]
					assert.Equal(t, "token", quote.BillingMode)
					require.NotNil(t, quote.InputPrice)
					assert.InDelta(t, want, *quote.InputPrice, 1e-15, "effective quote must not use the response group ratio")
				}
			}
			assert.Equal(t, before, capturePublicGroupSyncState(t))
		})
	}
}

func TestPublicGroupExplicitDisplayPricingRejectsInvalidMapBeforeAnyWrites(t *testing.T) {
	valid := `{"billing_mode":"token","source_group_id":71,"source_model":"shared","source_group_ratio":2,"input_price":0.000003}`
	type testCase struct {
		name, quote, snapshotFields string
		field, value                string
		version                     int
	}
	tests := []testCase{
		{name: "version one", quote: valid, version: 1},
		{name: "model not in snapshot", quote: valid, snapshotFields: `"models":["other"]`},
		{name: "inherited overlap", quote: valid, snapshotFields: `"model_pricing":{"shared":` + valid + `}`},
		{name: "disabled still validated", quote: `{}`, snapshotFields: `"public_enabled":false`},
		{name: "no source triple", quote: `{"billing_mode":"token","input_price":0.1}`},
	}
	for field, values := range map[string][]string{
		"source_group_id":    {`null`, `0`, `-1`, `72`},
		"source_model":       {`null`, `""`, `" "`, `"alias"`, `" shared "`},
		"source_group_ratio": {`null`, `-1`, `1e309`, `NaN`, `Infinity`},
		"billing_mode":       {`null`, `""`, `"unknown"`},
	} {
		for _, value := range values {
			tests = append(tests, testCase{name: field + "=" + value, field: field, value: value})
		}
	}
	for _, field := range []string{"input_price", "output_price", "image_input_price", "image_output_price", "per_request_price", "cache_read_price", "cache_write_price"} {
		for _, value := range []string{`-0.1`, `1e308`, `1e309`} {
			tests = append(tests, testCase{name: field + "=" + value, field: field, value: value})
		}
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setupPublicGroupSyncTest(t)
			if tt.field != "" {
				fields := map[string]string{"billing_mode": `"token"`, "source_group_id": `71`,
					"source_model": `"shared"`, "source_group_ratio": `2`, "input_price": `0.000003`}
				fields[tt.field] = tt.value
				var entries []string
				for name, value := range fields {
					entries = append(entries, fmt.Sprintf("%q:%s", name, value))
				}
				tt.quote = "{" + strings.Join(entries, ",") + "}"
			}
			seed := []byte(`{"version":2,"snapshots":[{"version":2,"group_id":99,"group_name":"retained","group_ratio":1,"public_enabled":true,"models":["retained"],"model_pricing":{"retained":{"source_group_id":71,"source_model":"source","source_group_ratio":2,"input_price":0.000004}}}]}`)
			require.Equal(t, http.StatusOK, postPublicGroupSyncTestRequest(t, seed, "test-secret"))
			before := capturePublicGroupSyncState(t)
			version := tt.version
			if version == 0 {
				version = 2
			}
			extra := ""
			if tt.snapshotFields != "" {
				extra = "," + tt.snapshotFields
			}
			// A valid earlier snapshot would update an existing route and insert
			// a new one. The later invalid map must prevent both operations.
			body := []byte(fmt.Sprintf(`{"version":%d,"snapshots":[
				{"version":%d,"group_id":99,"group_name":"changed","group_ratio":3,"models":["changed"],"model_pricing":{"changed":{"input_price":0.3}}},
				{"version":%d,"group_id":100,"group_name":"new","group_ratio":1,"models":["new"]},
				{"version":%d,"group_id":71,"group_name":"ordinary","group_ratio":1,"public_enabled":true,"models":["shared","valid"],"model_pricing":{},
				 "display_pricing":{"valid":{"billing_mode":"token","source_group_id":71,"source_model":"valid","source_group_ratio":1,"input_price":0},"shared":%s}%s}
			]}`, version, version, version, version, tt.quote, extra))
			require.Equal(t, http.StatusBadRequest, postPublicGroupSyncTestRequest(t, body, "test-secret"))
			assert.Equal(t, before, capturePublicGroupSyncState(t))
		})
	}
}

func TestPublicGroupExplicitDisplayPricingCoexistsWithInheritedQuotes(t *testing.T) {
	for _, mode := range []string{"token", "image", "per_request", " VIDEO "} {
		for _, enabled := range []bool{true, false} {
			t.Run(fmt.Sprintf("%s/enabled=%t", mode, enabled), func(t *testing.T) {
				setupPublicGroupSyncTest(t)
				body := []byte(fmt.Sprintf(`{"version":2,"snapshots":[{"version":2,"group_id":71,"group_name":"mixed","group_ratio":1,"public_enabled":%t,"models":["inherited","ordinary"],
					"model_pricing":{"inherited":{"billing_mode":"token","source_group_id":99,"source_model":"alias","source_group_ratio":3,"input_price":0.000002}},
					"display_pricing":{"ordinary":{"billing_mode":%q,"source_group_id":71,"source_model":"ordinary","source_group_ratio":2,"per_request_price":0.2}}}]}`, enabled, mode))
				require.Equal(t, http.StatusOK, postPublicGroupSyncTestRequest(t, body, "test-secret"))
				state := capturePublicGroupSyncState(t)
				var overlay map[string]map[string]model.PublicGroupDisplayPrice
				require.NoError(t, common.UnmarshalJsonStr(state.Options[publicGroupDisplayPricingOption], &overlay))
				if !enabled {
					assert.Empty(t, overlay)
					return
				}
				require.Len(t, overlay["mixed"], 2)
				wantMode := mode
				if mode == " VIDEO " {
					wantMode = "video"
				}
				price, input := 0.4, 0.000006
				assert.Equal(t, model.PublicGroupDisplayPrice{BillingMode: wantMode, PerRequestPrice: &price}, overlay["mixed"]["ordinary"])
				assert.Equal(t, model.PublicGroupDisplayPrice{BillingMode: "token", InputPrice: &input}, overlay["mixed"]["inherited"])
			})
		}
	}
}

func TestPublicGroupExplicitDisplayPricingWireRoundTripPreservesOrdinaryPricing(t *testing.T) {
	body := `{"version":2,"group_id":71,"group_name":"ordinary","group_ratio":1.5,"models":["shared"],"model_pricing":{"shared":{"billing_mode":"token","input_price":0,"output_price":0.000004}},"display_pricing":{"shared":{"platform":"","display_name":"","upstream_model":"","billing_mode":"image","source_group_id":71,"source_model":"shared","source_group_ratio":1,"per_request_price":0.2}}}`
	var snapshot PublicGroupSyncRequest
	require.NoError(t, common.UnmarshalJsonStr(body, &snapshot))
	encoded, err := common.Marshal(snapshot)
	require.NoError(t, err)
	var before, after map[string]any
	require.NoError(t, common.UnmarshalJsonStr(body, &before))
	require.NoError(t, common.Unmarshal(encoded, &after))
	assert.Equal(t, before["display_pricing"], after["display_pricing"])
	assert.Equal(t, before["group_ratio"], after["group_ratio"])
	// Existing DTO defaults are preserved; adding a display map must not
	// change any bytes produced for the ordinary model_pricing field.
	delete(before, "display_pricing")
	ordinary, err := common.Marshal(before)
	require.NoError(t, err)
	var legacy PublicGroupSyncRequest
	require.NoError(t, common.Unmarshal(ordinary, &legacy))
	want, err := common.Marshal(legacy.ModelPricing)
	require.NoError(t, err)
	got, err := common.Marshal(snapshot.ModelPricing)
	require.NoError(t, err)
	assert.Equal(t, string(want), string(got))
	encoded, err = common.Marshal(legacy)
	require.NoError(t, err)
	assert.NotContains(t, string(encoded), `"display_pricing"`)
}
