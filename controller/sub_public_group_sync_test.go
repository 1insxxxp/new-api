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
	"github.com/QuantumNous/new-api/setting/ratio_setting"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestReceivePublicGroupSyncSnapshotPersistsChannelsPricesAndGroups(t *testing.T) {
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
	originalUsableGroups := setting.UserUsableGroups2JSONString()
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
		_ = setting.UpdateUserUsableGroupsByJSONString(originalUsableGroups)
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

	inputPrice := 0.000004
	outputPrice := 0.000012
	envelope := publicGroupSyncEnvelope{Version: PublicGroupSyncSnapshotVersion, Snapshots: []PublicGroupSyncRequest{{
		Version:       PublicGroupSyncSnapshotVersion,
		GroupID:       42,
		GroupName:     "synced-group",
		PublicEnabled: true,
		GroupRatio:    2,
		Models:        []string{"sync-model"},
		ModelMapping:  map[string]string{"sync-model": "upstream-model"},
		ModelPricing: map[string]PublicGroupSyncModel{"sync-model": {
			Platform: "openai", DisplayName: "sync-model", UpstreamModel: "upstream-model", BillingMode: "token",
			InputPrice: &inputPrice, OutputPrice: &outputPrice,
		}},
	}}}
	body, err := json.Marshal(envelope)
	require.NoError(t, err)
	status := postPublicGroupSyncTestRequest(t, body, "test-secret")
	require.Equal(t, http.StatusOK, status)

	var channel model.Channel
	require.NoError(t, db.Where("tag = ?", "sub-public-group:42").First(&channel).Error)
	require.Equal(t, "synced-group", channel.Group)
	require.Equal(t, common.ChannelStatusEnabled, channel.Status)

	var ratioOption, groupOption, usableOption model.Option
	require.NoError(t, db.First(&ratioOption, "key = ?", "ModelRatio").Error)
	require.NoError(t, db.First(&groupOption, "key = ?", "GroupRatio").Error)
	require.NoError(t, db.First(&usableOption, "key = ?", "UserUsableGroups").Error)
	require.Contains(t, ratioOption.Value, "sync-model")
	require.Contains(t, groupOption.Value, "synced-group")
	require.Contains(t, usableOption.Value, "synced-group")

	emptyBody, err := json.Marshal(publicGroupSyncEnvelope{Version: PublicGroupSyncSnapshotVersion, Snapshots: []PublicGroupSyncRequest{}})
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, postPublicGroupSyncTestRequest(t, emptyBody, "test-secret"))
	require.NoError(t, db.First(&channel, channel.Id).Error)
	require.Equal(t, common.ChannelStatusManuallyDisabled, channel.Status)
	require.NoError(t, db.First(&groupOption, "key = ?", "GroupRatio").Error)
	require.NotContains(t, groupOption.Value, "synced-group")
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
