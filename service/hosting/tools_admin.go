package hosting

import (
	"fmt"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
)

func rejectProtectedUser(target *model.User, agentUserId int) error {
	if target == nil {
		return fmt.Errorf("user not found")
	}
	if target.Role >= common.RoleRootUser {
		return fmt.Errorf("refusing to modify a root user")
	}
	if target.Id == agentUserId {
		return fmt.Errorf("refusing to modify the hosting agent user")
	}
	if target.IsAgentAccount() {
		return fmt.Errorf("refusing to modify a hosting agent account")
	}
	return nil
}

func publicToken(token *model.Token) map[string]any {
	if token == nil {
		return nil
	}
	allow := ""
	if token.AllowIps != nil {
		allow = *token.AllowIps
	}
	return map[string]any{
		"id":              token.Id,
		"user_id":         token.UserId,
		"name":            token.Name,
		"status":          token.Status,
		"remain_quota":    token.RemainQuota,
		"unlimited_quota": token.UnlimitedQuota,
		"group":           token.Group,
		"expired_time":    token.ExpiredTime,
		"used_quota":      token.UsedQuota,
		"allow_ips":       allow,
		"key_prefix":      model.MaskTokenKey(token.Key),
	}
}

func toolListTokens(_ *ToolContext, args map[string]any) (any, error) {
	userId := argInt(args, "user_id")
	if userId <= 0 {
		return nil, fmt.Errorf("user_id is required")
	}
	page := argInt(args, "page")
	size := argInt(args, "page_size")
	if page <= 0 {
		page = 1
	}
	if size <= 0 || size > 100 {
		size = 20
	}
	tokens, err := model.GetAllUserTokens(userId, (page-1)*size, size)
	if err != nil {
		return nil, err
	}
	items := make([]map[string]any, 0, len(tokens))
	for _, token := range tokens {
		items = append(items, publicToken(token))
	}
	return map[string]any{"items": items, "page": page}, nil
}

func toolGetToken(_ *ToolContext, args map[string]any) (any, error) {
	token, err := model.GetTokenByIds(argInt(args, "id"), argInt(args, "user_id"))
	if err != nil {
		return nil, err
	}
	return publicToken(token), nil
}

func toolCreateToken(_ *ToolContext, args map[string]any) (any, error) {
	userId := argInt(args, "user_id")
	user, err := model.GetUserById(userId, false)
	if err != nil {
		return nil, err
	}
	if user.IsAgentAccount() {
		return nil, fmt.Errorf("refusing to create tokens for a hosting agent")
	}
	name := strings.TrimSpace(argString(args, "name"))
	if name == "" {
		return nil, fmt.Errorf("name is required")
	}
	key, err := common.GenerateKey()
	if err != nil {
		return nil, err
	}
	unlimited, _ := argBool(args, "unlimited_quota")
	token := &model.Token{
		UserId:         userId,
		Name:           name,
		Key:            key,
		CreatedTime:    common.GetTimestamp(),
		AccessedTime:   common.GetTimestamp(),
		ExpiredTime:    int64(argInt(args, "expired_time")),
		RemainQuota:    argInt(args, "remain_quota"),
		UnlimitedQuota: unlimited,
		Group:          argString(args, "group"),
		Status:         common.TokenStatusEnabled,
	}
	if token.ExpiredTime == 0 {
		token.ExpiredTime = -1
	}
	if err := token.Insert(); err != nil {
		return nil, err
	}
	return publicToken(token), nil
}

func toolUpdateToken(_ *ToolContext, args map[string]any) (any, error) {
	token, err := model.GetTokenByIds(argInt(args, "id"), argInt(args, "user_id"))
	if err != nil {
		return nil, err
	}
	if name := strings.TrimSpace(argString(args, "name")); name != "" {
		token.Name = name
	}
	if status := argInt(args, "status"); status != 0 {
		token.Status = status
	}
	if _, ok := args["remain_quota"]; ok {
		token.RemainQuota = argInt(args, "remain_quota")
	}
	if group := argString(args, "group"); group != "" {
		token.Group = group
	}
	if ips := argString(args, "allow_ips"); ips != "" {
		token.AllowIps = &ips
	}
	if err := token.Update(); err != nil {
		return nil, err
	}
	return publicToken(token), nil
}

func toolUpdateUserStatus(ctx *ToolContext, args map[string]any) (any, error) {
	user, err := model.GetUserById(argInt(args, "id"), false)
	if err != nil {
		return nil, err
	}
	if err := rejectProtectedUser(user, ctx.UserID); err != nil {
		return nil, err
	}
	status := argInt(args, "status")
	if status != common.UserStatusEnabled && status != common.UserStatusDisabled {
		return nil, fmt.Errorf("unsupported user status")
	}
	user.Status = status
	if err := user.Update(false); err != nil {
		return nil, err
	}
	return map[string]any{"id": user.Id, "status": user.Status}, nil
}

func toolAddUserQuota(ctx *ToolContext, args map[string]any) (any, error) {
	user, err := model.GetUserById(argInt(args, "id"), false)
	if err != nil {
		return nil, err
	}
	if err := rejectProtectedUser(user, ctx.UserID); err != nil {
		return nil, err
	}
	quota := argInt(args, "quota")
	if quota <= 0 {
		return nil, fmt.Errorf("quota must be positive")
	}
	if err := common.ValidateWalletQuota(quota); err != nil {
		return nil, err
	}
	if err := model.IncreaseUserQuota(user.Id, quota, true); err != nil {
		return nil, err
	}
	return map[string]any{"id": user.Id, "added": quota}, nil
}

func toolUpdateGroupRatio(_ *ToolContext, args map[string]any) (any, error) {
	name := strings.TrimSpace(argString(args, "group"))
	if name == "" {
		return nil, fmt.Errorf("group is required")
	}
	copy := ratio_setting.GetGroupRatioCopy()
	if _, ok := copy[name]; !ok {
		return nil, fmt.Errorf("group does not exist")
	}
	ratio := argFloat(args, "ratio")
	if ratio <= 0 {
		return nil, fmt.Errorf("ratio must be positive")
	}
	copy[name] = ratio
	raw, err := common.Marshal(copy)
	if err != nil {
		return nil, err
	}
	if err := model.UpdateOption("GroupRatio", string(raw)); err != nil {
		return nil, err
	}
	return map[string]any{"group": name, "ratio": ratio}, nil
}

func toolCreateRedemption(ctx *ToolContext, args map[string]any) (any, error) {
	if !operation_setting.IsPaymentComplianceConfirmed() {
		return nil, fmt.Errorf("payment compliance is not confirmed")
	}
	name := strings.TrimSpace(argString(args, "name"))
	if name == "" {
		return nil, fmt.Errorf("name is required")
	}
	quota := argInt(args, "quota")
	count := argInt(args, "count")
	if count <= 0 {
		count = 1
	}
	if count > 20 {
		return nil, fmt.Errorf("count must be at most 20")
	}
	codes := make([]string, 0, count)
	for i := 0; i < count; i++ {
		item := &model.Redemption{
			UserId:      ctx.UserID,
			Name:        name,
			Key:         common.GetUUID(),
			CreatedTime: common.GetTimestamp(),
			Quota:       quota,
			ExpiredTime: int64(argInt(args, "expired_time")),
			Status:      common.RedemptionCodeStatusEnabled,
		}
		if err := item.Insert(); err != nil {
			return nil, err
		}
		codes = append(codes, item.Key)
	}
	return map[string]any{"name": name, "count": count, "codes": codes}, nil
}

func toolUpdateRedemptionStatus(_ *ToolContext, args map[string]any) (any, error) {
	item, err := model.GetRedemptionById(argInt(args, "id"))
	if err != nil {
		return nil, err
	}
	status := argInt(args, "status")
	item.Status = status
	if err := item.SelectUpdate(); err != nil {
		return nil, err
	}
	return map[string]any{"id": item.Id, "status": item.Status}, nil
}

func toolListSubscriptionPlans(_ *ToolContext, _ map[string]any) (any, error) {
	var plans []model.SubscriptionPlan
	if err := model.DB.Order("sort_order desc, id desc").Find(&plans).Error; err != nil {
		return nil, err
	}
	return map[string]any{"items": plans}, nil
}

func toolListUserSubscriptions(_ *ToolContext, args map[string]any) (any, error) {
	userId := argInt(args, "user_id")
	items, err := model.GetAllUserSubscriptions(userId)
	if err != nil {
		return nil, err
	}
	return map[string]any{"items": items}, nil
}

func toolSetSubscriptionPlanEnabled(_ *ToolContext, args map[string]any) (any, error) {
	id := argInt(args, "id")
	if id <= 0 {
		return nil, fmt.Errorf("id is required")
	}
	enabled, ok := argBool(args, "enabled")
	if !ok {
		return nil, fmt.Errorf("enabled is required")
	}
	if err := model.DB.Model(&model.SubscriptionPlan{}).Where("id = ?", id).Update("enabled", enabled).Error; err != nil {
		return nil, err
	}
	model.InvalidateSubscriptionPlanCache(id)
	return map[string]any{"id": id, "enabled": enabled}, nil
}

func toolListVendors(_ *ToolContext, args map[string]any) (any, error) {
	page := argInt(args, "page")
	size := argInt(args, "page_size")
	if page <= 0 {
		page = 1
	}
	if size <= 0 || size > 100 {
		size = 20
	}
	items, err := model.GetAllVendors((page-1)*size, size)
	if err != nil {
		return nil, err
	}
	return map[string]any{"items": items, "page": page}, nil
}

func toolGetVendor(_ *ToolContext, args map[string]any) (any, error) {
	item, err := model.GetVendorByID(argInt(args, "id"))
	if err != nil {
		return nil, err
	}
	return item, nil
}

func toolCreateVendor(_ *ToolContext, args map[string]any) (any, error) {
	name := strings.TrimSpace(argString(args, "name"))
	if name == "" {
		return nil, fmt.Errorf("name is required")
	}
	dup, err := model.IsVendorNameDuplicated(0, name)
	if err != nil {
		return nil, err
	}
	if dup {
		return nil, fmt.Errorf("vendor name already exists")
	}
	item := &model.Vendor{
		Name:        name,
		Description: argString(args, "description"),
		Icon:        argString(args, "icon"),
		Status:      1,
	}
	if err := item.Insert(); err != nil {
		return nil, err
	}
	return item, nil
}

func toolListModelMeta(_ *ToolContext, args map[string]any) (any, error) {
	page := argInt(args, "page")
	size := argInt(args, "page_size")
	if page <= 0 {
		page = 1
	}
	if size <= 0 || size > 100 {
		size = 20
	}
	items, total, err := model.SearchModels(argString(args, "keyword"), "", "", "", (page-1)*size, size)
	if err != nil {
		return nil, err
	}
	return map[string]any{"items": items, "total": total, "page": page}, nil
}

func toolGetModelMeta(_ *ToolContext, args map[string]any) (any, error) {
	id := argInt(args, "id")
	if id <= 0 {
		return nil, fmt.Errorf("id is required")
	}
	var item model.Model
	if err := model.DB.First(&item, id).Error; err != nil {
		return nil, err
	}
	return item, nil
}

func toolListSystemTasks(_ *ToolContext, args map[string]any) (any, error) {
	limit := argInt(args, "limit")
	tasks, err := model.ListSystemTasks(limit)
	if err != nil {
		return nil, err
	}
	out := make([]map[string]any, 0, len(tasks))
	for _, task := range tasks {
		if task == nil {
			continue
		}
		out = append(out, map[string]any{
			"id":         task.ID,
			"type":       task.Type,
			"status":     task.Status,
			"updated_at": task.UpdatedAt,
			"error":      task.Error,
		})
	}
	return map[string]any{"items": out}, nil
}
