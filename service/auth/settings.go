package auth

import (
	"context"
	"strings"

	"github.com/cloudreve/Cloudreve/v4/application/dependency"
	"github.com/cloudreve/Cloudreve/v4/ent/setting"
	"github.com/gin-gonic/gin"
)

// seedSSOSettings lazily inserts the PRO SSO setting keys that may not exist on
// installations upgraded before they were introduced. Admin settings save
// requires these rows to already be present, so they are created on the first
// SSO-related request.
func seedSSOSettings(c *gin.Context, dep dependency.Dep) {
	db := dep.DBClient()
	for _, key := range ssoSettingKeys {
		exists, err := db.Setting.Query().Where(setting.Name(key)).Exist(c)
		if err != nil || exists {
			continue
		}

		value := "0"
		if key == "oidc_config" {
			value = "{}"
		}

		if _, err := db.Setting.Create().SetName(key).SetValue(value).Save(c); err != nil {
			dep.Logger().Warning("Failed to seed SSO setting %q: %s", key, err)
		}
	}
}

// ssoEnabledKey reports whether the provider enable flag is set to a truthy value.
func ssoEnabledKey(c context.Context, dep dependency.Dep, key string) bool {
	values, err := dep.SettingClient().Gets(c, []string{key})
	if err != nil {
		return false
	}
	switch strings.ToLower(strings.TrimSpace(values[key])) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

// getSettingString returns the value of a setting key, or the fallback when the
// key is missing.
func getSettingString(c context.Context, dep dependency.Dep, key, fallback string) string {
	values, err := dep.SettingClient().Gets(c, []string{key})
	if err != nil {
		return fallback
	}
	if v, ok := values[key]; ok {
		return v
	}
	return fallback
}
