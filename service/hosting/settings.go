package hosting

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
)

type settingSpec struct {
	Key         string
	Title       string
	Description string
	Category    string
	Risk        string
}

var stewardSettingCatalog = []settingSpec{
	{Key: "SystemName", Title: "Site name", Description: "The name shown on the login page and browser tab.", Category: "site", Risk: constant.HostingToolRiskLow},
	{Key: "Logo", Title: "Logo URL", Description: "Image address for the site logo.", Category: "site", Risk: constant.HostingToolRiskLow},
	{Key: "Footer", Title: "Footer text", Description: "Short text at the bottom of pages.", Category: "site", Risk: constant.HostingToolRiskLow},
	{Key: "About", Title: "About page", Description: "Content for the About page.", Category: "site", Risk: constant.HostingToolRiskLow},
	{Key: "HomePageContent", Title: "Home page content", Description: "Extra content on the public home page.", Category: "site", Risk: constant.HostingToolRiskLow},
	{Key: "Notice", Title: "Notice", Description: "Announcement shown to users after they sign in.", Category: "site", Risk: constant.HostingToolRiskLow},
	{Key: "ServerAddress", Title: "Public site URL", Description: "The address people use to open this site, such as https://api.example.com.", Category: "site", Risk: constant.HostingToolRiskMedium},
	{Key: "RegisterEnabled", Title: "Allow new sign-ups", Description: "Whether new people can create accounts.", Category: "access", Risk: constant.HostingToolRiskMedium},
	{Key: "PasswordLoginEnabled", Title: "Password login", Description: "Allow signing in with a username and password.", Category: "access", Risk: constant.HostingToolRiskMedium},
	{Key: "PasswordRegisterEnabled", Title: "Password sign-up", Description: "Allow creating accounts with a username and password.", Category: "access", Risk: constant.HostingToolRiskMedium},
	{Key: "EmailVerificationEnabled", Title: "Require email verification", Description: "New accounts must confirm their email.", Category: "access", Risk: constant.HostingToolRiskMedium},
	{Key: "QuotaForNewUser", Title: "Quota for new users", Description: "How much quota a brand-new account receives.", Category: "billing", Risk: constant.HostingToolRiskMedium},
	{Key: "QuotaRemindThreshold", Title: "Low-quota reminder", Description: "Warn a user when remaining quota falls to this number.", Category: "billing", Risk: constant.HostingToolRiskLow},
	{Key: "DisplayInCurrencyEnabled", Title: "Show money instead of quota", Description: "Display usage as currency on the console.", Category: "billing", Risk: constant.HostingToolRiskLow},
	{Key: "USDExchangeRate", Title: "USD exchange rate", Description: "How many quota units equal one US dollar.", Category: "billing", Risk: constant.HostingToolRiskMedium},
	{Key: "MinTopUp", Title: "Minimum top-up", Description: "Smallest amount a user can add to their wallet.", Category: "billing", Risk: constant.HostingToolRiskMedium},
	{Key: "AutomaticDisableChannelEnabled", Title: "Auto-disable bad channels", Description: "Turn a channel off automatically when it keeps failing.", Category: "operations", Risk: constant.HostingToolRiskMedium},
	{Key: "AutomaticEnableChannelEnabled", Title: "Auto-enable recovered channels", Description: "Turn a channel back on when it looks healthy again.", Category: "operations", Risk: constant.HostingToolRiskMedium},
	{Key: "RetryTimes", Title: "Retry count", Description: "How many times to retry a failed model request.", Category: "operations", Risk: constant.HostingToolRiskLow},
	{Key: "LogConsumeEnabled", Title: "Record usage logs", Description: "Save consume logs for each API call.", Category: "operations", Risk: constant.HostingToolRiskLow},
	{Key: "DrawingEnabled", Title: "Image tasks", Description: "Allow image-generation task APIs.", Category: "features", Risk: constant.HostingToolRiskLow},
	{Key: "TaskEnabled", Title: "Async tasks", Description: "Allow video and other long-running task APIs.", Category: "features", Risk: constant.HostingToolRiskLow},
	{Key: "DefaultCollapseSidebar", Title: "Collapse sidebar by default", Description: "Start with a narrow sidebar for new visitors.", Category: "site", Risk: constant.HostingToolRiskLow},
}

func settingSpecByKey(key string) (settingSpec, bool) {
	for _, spec := range stewardSettingCatalog {
		if spec.Key == key {
			return spec, true
		}
	}
	return settingSpec{}, false
}

func IsSensitiveOptionKey(key string) bool {
	k := strings.TrimSpace(key)
	if k == "" {
		return false
	}
	lower := strings.ToLower(k)
	if strings.Contains(lower, "secret") || strings.Contains(lower, "private") || strings.Contains(lower, "password") || strings.Contains(lower, "token") || strings.Contains(lower, "api_key") {
		return true
	}
	return strings.HasSuffix(k, "Key") || strings.HasSuffix(k, "Token") || strings.HasSuffix(k, "Secret") || strings.HasSuffix(k, "Cert")
}

func SettingWriteRisk(key string) string {
	if IsSensitiveOptionKey(key) {
		return constant.HostingToolRiskHigh
	}
	if spec, ok := settingSpecByKey(key); ok {
		return spec.Risk
	}
	lower := strings.ToLower(key)
	if strings.Contains(lower, "ratio") || strings.Contains(lower, "price") || strings.Contains(lower, "payment") || strings.Contains(lower, "stripe") || strings.Contains(lower, "epay") || strings.Contains(lower, "waffo") || strings.Contains(lower, "creem") {
		return constant.HostingToolRiskHigh
	}
	return constant.HostingToolRiskMedium
}

func optionValue(key string) (string, bool) {
	common.OptionMapRWMutex.RLock()
	defer common.OptionMapRWMutex.RUnlock()
	value, ok := common.OptionMap[key]
	return common.Interface2String(value), ok
}

func redactOptionValue(key, value string) string {
	if IsSensitiveOptionKey(key) {
		if strings.TrimSpace(value) == "" {
			return ""
		}
		return "(hidden)"
	}
	return value
}

func publicSettingView(spec settingSpec, value string, present bool) map[string]any {
	view := map[string]any{
		"key":          spec.Key,
		"title":        spec.Title,
		"description":  spec.Description,
		"category":     spec.Category,
		"risk":         spec.Risk,
		"sensitive":    IsSensitiveOptionKey(spec.Key),
		"value":        redactOptionValue(spec.Key, value),
		"is_set":       present && strings.TrimSpace(value) != "",
		"needs_review": spec.Risk == constant.HostingToolRiskHigh,
	}
	return view
}

func toolListSystemSettings(_ *ToolContext, args map[string]any) (any, error) {
	keyword := strings.ToLower(strings.TrimSpace(argString(args, "keyword")))
	category := strings.TrimSpace(argString(args, "category"))
	out := make([]map[string]any, 0, len(stewardSettingCatalog))
	for _, spec := range stewardSettingCatalog {
		if category != "" && spec.Category != category {
			continue
		}
		if keyword != "" {
			blob := strings.ToLower(spec.Key + " " + spec.Title + " " + spec.Description)
			if !strings.Contains(blob, keyword) {
				continue
			}
		}
		value, present := optionValue(spec.Key)
		out = append(out, publicSettingView(spec, value, present))
	}
	if keyword != "" && len(out) == 0 {
		common.OptionMapRWMutex.RLock()
		for key, raw := range common.OptionMap {
			if !strings.Contains(strings.ToLower(key), keyword) {
				continue
			}
			if IsSensitiveOptionKey(key) {
				continue
			}
			spec := settingSpec{Key: key, Title: key, Description: "System setting " + key, Category: "other", Risk: SettingWriteRisk(key)}
			out = append(out, publicSettingView(spec, common.Interface2String(raw), true))
			if len(out) >= 40 {
				break
			}
		}
		common.OptionMapRWMutex.RUnlock()
	}
	return map[string]any{"settings": out, "count": len(out)}, nil
}

func toolGetSetupChecklist(ctx *ToolContext, _ map[string]any) (any, error) {
	var channelCount int64
	if model.DB != nil {
		_ = model.DB.Model(&model.Channel{}).Count(&channelCount).Error
	}
	pending := int64(0)
	if ctx != nil && ctx.Agent != nil {
		pending, _ = model.CountPendingHostingApprovals(ctx.Agent.Id)
	}
	siteName, _ := optionValue("SystemName")
	register, _ := optionValue("RegisterEnabled")
	ready := ctx != nil && BrainIsConfigured(ctx.Agent)
	practice := ctx != nil && ctx.Agent != nil && ctx.Agent.DryRun
	next := make([]string, 0, 4)
	if strings.TrimSpace(siteName) == "" || strings.EqualFold(siteName, "New API") {
		next = append(next, "Set a site name people will recognize.")
	}
	if !ready {
		next = append(next, "Choose an AI model in Steward Settings so the steward can talk.")
	}
	if channelCount == 0 {
		next = append(next, "Add at least one model channel so API calls have somewhere to go.")
	}
	if practice {
		next = append(next, "Practice mode is on: the steward will not apply real changes yet.")
	}
	if len(next) == 0 {
		next = append(next, "Basic setup looks complete. Ask the steward if you want to change a specific option.")
	}
	return map[string]any{
		"site_name":           siteName,
		"register_enabled":    register,
		"channel_count":       channelCount,
		"steward_model_ready": ready,
		"practice_mode":       practice,
		"pending_approvals":   pending,
		"next_steps":          next,
	}, nil
}

func validateSettingWrite(key, value string) error {
	key = strings.TrimSpace(key)
	if key == "" {
		return fmt.Errorf("setting key is required")
	}
	if utf8.RuneCountInString(key) > 191 {
		return fmt.Errorf("setting key is too long")
	}
	if utf8.RuneCountInString(value) > constant.MaxHostingOptionValueRunes {
		return fmt.Errorf("setting value is too long")
	}
	return nil
}

func toolUpdateSystemSetting(_ *ToolContext, args map[string]any) (any, error) {
	key := strings.TrimSpace(argString(args, "key"))
	value := argString(args, "value")
	if err := validateSettingWrite(key, value); err != nil {
		return nil, err
	}
	if err := model.UpdateOption(key, value); err != nil {
		return nil, err
	}
	spec, ok := settingSpecByKey(key)
	title := key
	if ok {
		title = spec.Title
	}
	return map[string]any{
		"updated": true,
		"key":     key,
		"title":   title,
		"value":   redactOptionValue(key, value),
		"risk":    SettingWriteRisk(key),
	}, nil
}
