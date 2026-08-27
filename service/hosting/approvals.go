package hosting

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service/authz"
)

type ApprovalDecision struct {
	Approve bool   `json:"approve"`
	Note    string `json:"note"`
}

func maybeQueueToolApproval(ctx *ToolContext, tool *ToolSpec, args map[string]any) (any, bool, error) {
	if ctx == nil || ctx.SkipReview || tool == nil || !tool.Mutating {
		return nil, false, nil
	}
	if tool.Name == "handoff_incident" || tool.Name == "sleep_until_hook" || tool.Name == "request_user_access" {
		return nil, false, nil
	}
	risk := ToolRiskLevel(tool.Name, tool.Mutating)
	if tool.Name == "update_system_setting" {
		risk = SettingWriteRisk(argString(args, "key"))
	}
	auto := actorCanAutoExecute(ctx) && AutoReviewAllows(ctx.Agent.AutoReviewMode, risk)
	if auto {
		return nil, false, nil
	}
	payload, err := common.Marshal(map[string]any{"tool": tool.Name, "args": args})
	if err != nil {
		return nil, false, err
	}
	requester := ctx.ActorUserID
	if requester <= 0 {
		requester = ctx.UserID
	}
	item := &model.HostingApproval{
		AgentId:         ctx.Agent.Id,
		RequesterUserId: requester,
		Kind:            constant.HostingApprovalToolAction,
		Status:          constant.HostingApprovalPending,
		ToolName:        tool.Name,
		Risk:            risk,
		Summary:         toolApprovalSummary(tool.Name, args),
		PayloadJSON:     string(payload),
		Reason:          "Waiting for a human to confirm this change.",
		AutoReviewRule:  ctx.Agent.AutoReviewMode,
	}
	if err := item.Insert(); err != nil {
		return nil, false, err
	}
	if ctx.FromRunner && !ctx.Interactive {
		return nil, true, handoffErr("needs_approval", "tool "+tool.Name+" needs approval #"+fmt.Sprintf("%d", item.Id))
	}
	return map[string]any{
		"pending_approval": true,
		"approval_id":      item.Id,
		"tool":             tool.Name,
		"risk":             risk,
		"summary":          item.Summary,
		"message":          "This change is waiting for approval. Nothing was changed yet.",
	}, true, nil
}

func toolApprovalSummary(name string, args map[string]any) string {
	id := argInt(args, "id")
	if id > 0 {
		return fmt.Sprintf("%s #%d", name, id)
	}
	return name
}

func ListApprovalsForActor(agentId, actorUserId, actorRole int, status string) ([]*model.HostingApproval, error) {
	requester := 0
	if actorRole < common.RoleAdminUser {
		requester = actorUserId
	}
	return model.ListHostingApprovals(agentId, requester, status, 80)
}

func DecideApproval(id, reviewerId, reviewerRole int, decision ApprovalDecision) (*model.HostingApproval, error) {
	if reviewerRole < common.RoleAdminUser {
		return nil, fmt.Errorf("only an admin can review approvals")
	}
	item, err := model.GetHostingApprovalById(id)
	if err != nil {
		return nil, err
	}
	if item.Status != constant.HostingApprovalPending {
		return nil, fmt.Errorf("approval is no longer pending")
	}
	now := common.GetTimestamp()
	updates := map[string]any{
		"reviewer_user_id": reviewerId,
		"review_note":      strings.TrimSpace(decision.Note),
		"decided_at":       now,
	}
	if !decision.Approve {
		updates["status"] = constant.HostingApprovalDenied
		if err := model.DB.Model(item).Updates(updates).Error; err != nil {
			return nil, err
		}
		item.Status = constant.HostingApprovalDenied
		item.ReviewNote = strings.TrimSpace(decision.Note)
		item.ReviewerUserId = reviewerId
		item.DecidedAt = now
		return item, nil
	}
	if err := applyApprovedItem(item); err != nil {
		return nil, err
	}
	updates["status"] = constant.HostingApprovalExecuted
	updates["executed_at"] = now
	if err := model.DB.Model(item).Updates(updates).Error; err != nil {
		return nil, err
	}
	item.Status = constant.HostingApprovalExecuted
	item.ReviewNote = strings.TrimSpace(decision.Note)
	item.ReviewerUserId = reviewerId
	item.DecidedAt = now
	item.ExecutedAt = now
	return item, nil
}

func applyApprovedItem(item *model.HostingApproval) error {
	switch item.Kind {
	case constant.HostingApprovalUserPermission:
		return applyUserPermissionPayload(item)
	default:
		return executeApprovedTool(item)
	}
}

func executeApprovedTool(item *model.HostingApproval) error {
	var payload struct {
		Tool string         `json:"tool"`
		Args map[string]any `json:"args"`
	}
	if err := common.UnmarshalJsonStr(item.PayloadJSON, &payload); err != nil {
		return err
	}
	agent, err := model.GetHostingAgentById(item.AgentId)
	if err != nil {
		return err
	}
	ctx := &ToolContext{
		Agent:       agent,
		UserID:      agent.UserId,
		Role:        common.RoleAdminUser,
		ActorUserID: item.RequesterUserId,
		ActorRole:   common.RoleAdminUser,
		SkipReview:  true,
		FromRunner:  false,
	}
	authz.MarkHostingAgentUser(agent.UserId)
	_, err = ExecuteTool(ctx, payload.Tool, payload.Args)
	return err
}

func applyUserPermissionPayload(item *model.HostingApproval) error {
	var payload userAccessPayload
	if err := common.UnmarshalJsonStr(item.PayloadJSON, &payload); err != nil {
		return err
	}
	agent, err := model.GetHostingAgentById(item.AgentId)
	if err == nil && agent != nil && agent.DryRun {
		return fmt.Errorf("practice mode is on; this access change was not applied")
	}
	return applyUserAccess(payload)
}

type userAccessPayload struct {
	RequestType  string `json:"request_type"`
	TargetUserId int    `json:"target_user_id"`
	Quota        int    `json:"quota"`
	Group        string `json:"group"`
	Note         string `json:"note"`
}

func applyUserAccess(payload userAccessPayload) error {
	if payload.TargetUserId <= 0 {
		return fmt.Errorf("target user is required")
	}
	user, err := model.GetUserById(payload.TargetUserId, true)
	if err != nil {
		return err
	}
	if err := rejectProtectedUser(user, 0); err != nil {
		return err
	}
	switch payload.RequestType {
	case "quota":
		if payload.Quota <= 0 || payload.Quota > constant.MaxHostingQuotaGrant {
			return fmt.Errorf("quota must be between 1 and %d", constant.MaxHostingQuotaGrant)
		}
		if err := common.ValidateWalletQuota(payload.Quota); err != nil {
			return err
		}
		return model.IncreaseUserQuota(user.Id, payload.Quota, true)
	case "group":
		group := strings.TrimSpace(payload.Group)
		if group == "" {
			return fmt.Errorf("group is required")
		}
		return model.DB.Model(user).Update("group", group).Error
	case "enable":
		return model.DB.Model(user).Update("status", common.UserStatusEnabled).Error
	case "admin":
		if user.Role >= common.RoleAdminUser {
			return nil
		}
		return model.DB.Model(user).Update("role", common.RoleAdminUser).Error
	default:
		return fmt.Errorf("unknown access request type")
	}
}

func queueUserAccessRequest(ctx *ToolContext, args map[string]any) (any, error) {
	if ctx == nil || ctx.Agent == nil {
		return nil, fmt.Errorf("missing hosting agent context")
	}
	payload := userAccessPayload{
		RequestType:  strings.TrimSpace(argString(args, "request_type")),
		TargetUserId: argInt(args, "target_user_id"),
		Quota:        argInt(args, "quota"),
		Group:        strings.TrimSpace(argString(args, "group")),
		Note:         strings.TrimSpace(argString(args, "note")),
	}
	if payload.TargetUserId <= 0 {
		payload.TargetUserId = ctx.ActorUserID
	}
	if payload.TargetUserId <= 0 {
		return nil, fmt.Errorf("target user is required")
	}
	if ctx.ActorRole < common.RoleAdminUser && payload.TargetUserId != ctx.ActorUserID {
		return nil, fmt.Errorf("you can only request access for your own account")
	}
	if payload.RequestType == "" {
		payload.RequestType = "quota"
	}
	if payload.RequestType == "quota" {
		if payload.Quota <= 0 || payload.Quota > constant.MaxHostingQuotaGrant {
			return nil, fmt.Errorf("quota must be between 1 and %d", constant.MaxHostingQuotaGrant)
		}
		if utf8.RuneCountInString(payload.Note) > 500 {
			return nil, fmt.Errorf("note is too long")
		}
	}
	raw, err := common.Marshal(payload)
	if err != nil {
		return nil, err
	}
	risk := constant.HostingToolRiskHigh
	if payload.RequestType == "quota" && payload.Quota <= constant.MaxHostingAutoQuotaGrant {
		risk = constant.HostingToolRiskLow
	}
	if payload.RequestType == "enable" || payload.RequestType == "group" {
		risk = constant.HostingToolRiskMedium
	}
	item := &model.HostingApproval{
		AgentId:         ctx.Agent.Id,
		RequesterUserId: ctx.ActorUserID,
		TargetUserId:    payload.TargetUserId,
		Kind:            constant.HostingApprovalUserPermission,
		Status:          constant.HostingApprovalPending,
		ToolName:        "request_user_access",
		Risk:            risk,
		Resource:        "user",
		Action:          payload.RequestType,
		Summary:         userAccessSummary(payload),
		PayloadJSON:     string(raw),
		Reason:          payload.Note,
		AutoReviewRule:  ctx.Agent.AutoReviewMode,
	}
	if ctx.Agent.DryRun {
		return map[string]any{
			"dry_run": true,
			"summary": item.Summary,
			"message": "Practice mode: this access change was not applied.",
		}, nil
	}
	if actorCanAutoExecute(ctx) && AutoReviewAllows(ctx.Agent.AutoReviewMode, risk) && payload.RequestType != "admin" {
		if err := applyUserAccess(payload); err != nil {
			return nil, err
		}
		item.Status = constant.HostingApprovalAutoApproved
		item.DecidedAt = common.GetTimestamp()
		item.ExecutedAt = item.DecidedAt
		item.ReviewerUserId = ctx.ActorUserID
		item.AutoReviewRule = ctx.Agent.AutoReviewMode
		if err := item.Insert(); err != nil {
			return nil, err
		}
		return map[string]any{
			"auto_approved": true,
			"approval_id":   item.Id,
			"summary":       item.Summary,
		}, nil
	}
	if err := item.Insert(); err != nil {
		return nil, err
	}
	return map[string]any{
		"pending_approval": true,
		"approval_id":      item.Id,
		"summary":          item.Summary,
		"message":          "This access request is waiting for an admin to approve.",
	}, nil
}

func userAccessSummary(payload userAccessPayload) string {
	switch payload.RequestType {
	case "quota":
		return fmt.Sprintf("Add %d quota to user #%d", payload.Quota, payload.TargetUserId)
	case "group":
		return fmt.Sprintf("Move user #%d to group %s", payload.TargetUserId, payload.Group)
	case "enable":
		return fmt.Sprintf("Enable user #%d", payload.TargetUserId)
	case "admin":
		return fmt.Sprintf("Grant admin to user #%d", payload.TargetUserId)
	default:
		return "User access request"
	}
}
