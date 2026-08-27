package hosting

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
)

type IssuedToken struct {
	Token  *model.HostingAgentToken
	Secret string
}

func IssueAgentToken(agentId int, name, allowIPs string) (*IssuedToken, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return nil, err
	}
	secret := constant.HostingTokenPrefix + hex.EncodeToString(raw)
	prefix := secret
	if len(prefix) > 12 {
		prefix = prefix[:12]
	}
	row := &model.HostingAgentToken{
		AgentId:     agentId,
		Name:        strings.TrimSpace(name),
		TokenHash:   common.GenerateHMAC(secret),
		TokenPrefix: prefix,
		AllowIPs:    strings.TrimSpace(allowIPs),
		Status:      common.UserStatusEnabled,
	}
	if err := row.Insert(); err != nil {
		return nil, err
	}
	return &IssuedToken{Token: row, Secret: secret}, nil
}

func RotateAgentToken(agentId, tokenId int, allowIPs string) (*IssuedToken, error) {
	existing, err := model.GetHostingTokenById(tokenId, agentId)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(allowIPs) == "" {
		allowIPs = existing.AllowIPs
	}
	if err := model.RevokeHostingToken(tokenId, agentId); err != nil {
		return nil, err
	}
	name := "rotated"
	if existing.Name != "" {
		name = existing.Name
	}
	return IssueAgentToken(agentId, name, allowIPs)
}

func AuthenticateAgentToken(secret, clientIP string) (*model.HostingAgent, *model.HostingAgentToken, error) {
	secret = strings.TrimSpace(secret)
	if strings.HasPrefix(strings.ToLower(secret), "bearer ") {
		secret = strings.TrimSpace(secret[7:])
	}
	if !strings.HasPrefix(secret, constant.HostingTokenPrefix) {
		return nil, nil, fmt.Errorf("invalid hosting token")
	}
	hash := common.GenerateHMAC(secret)
	token, err := model.GetHostingTokenByHash(hash)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid hosting token")
	}
	if !ipAllowed(token.AllowIPs, clientIP) {
		return nil, nil, fmt.Errorf("hosting token IP is not allowed")
	}
	agent, err := model.GetHostingAgentById(token.AgentId)
	if err != nil {
		return nil, nil, err
	}
	if !agent.Enabled {
		return nil, nil, fmt.Errorf("hosting agent is disabled")
	}
	model.TouchHostingToken(token.Id)
	return agent, token, nil
}

func ipAllowed(allowIPs, clientIP string) bool {
	allowIPs = strings.TrimSpace(allowIPs)
	if allowIPs == "" {
		return true
	}
	ip := net.ParseIP(clientIP)
	for _, part := range strings.Split(allowIPs, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		if strings.Contains(part, "/") {
			_, cidr, err := net.ParseCIDR(part)
			if err == nil && ip != nil && cidr.Contains(ip) {
				return true
			}
			continue
		}
		if part == clientIP {
			return true
		}
	}
	return false
}
