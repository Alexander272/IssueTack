package mattermost

import (
	"bytes"
	"net"
	"strings"

	"github.com/Alexander272/IssueTrack/backend/internal/config"
	"github.com/Alexander272/IssueTrack/backend/internal/models"
	"github.com/Alexander272/IssueTrack/backend/internal/models/response"
	"github.com/Alexander272/IssueTrack/backend/pkg/logger"
	"github.com/gin-gonic/gin"
)

// ipRange — замкнутый диапазон IP [lo, hi] (включая границы).
type ipRange struct {
	lo net.IP
	hi net.IP
}

// sourceGuard проверяет, что входящий запрос пришёл от доверенного источника
// Mattermost (сам сервер MM или стоящий перед ним прокси). Источник определяется
// по TCP peer-адресу (RemoteAddr), а не по ClientIP/X-Forwarded-For, чтобы нельзя
// было подделать через доверенные прокси (router.SetTrustedProxies).
type sourceGuard struct {
	ips    []net.IP
	nets   []*net.IPNet
	ranges []ipRange
}

// newSourceGuard собирает allowlist из конфига. Форматы записей:
// одиночный IP ("10.0.0.1"), CIDR ("10.0.0.0/24"), диапазон ("10.0.0.5-10.0.0.20").
func newSourceGuard(cfg config.MattermostConfig) *sourceGuard {
	var g sourceGuard

	for _, raw := range cfg.AllowedServerIPs {
		entry := strings.TrimSpace(raw)
		if entry == "" {
			continue
		}

		switch {
		case strings.Contains(entry, "/"):
			_, network, err := net.ParseCIDR(entry)
			if err != nil {
				logger.Warn("invalid CIDR in mattermost allowed_server_ips", logger.StringAttr("value", entry))
				continue
			}
			g.nets = append(g.nets, network)

		case strings.Contains(entry, "-"):
			parts := strings.SplitN(entry, "-", 2)
			if len(parts) != 2 {
				logger.Warn("invalid IP range in mattermost allowed_server_ips", logger.StringAttr("value", entry))
				continue
			}
			lo := net.ParseIP(strings.TrimSpace(parts[0]))
			hi := net.ParseIP(strings.TrimSpace(parts[1]))
			if lo == nil || hi == nil {
				logger.Warn("invalid IP range in mattermost allowed_server_ips", logger.StringAttr("value", entry))
				continue
			}
			if bytes.Compare(toComparableIP(lo), toComparableIP(hi)) > 0 {
				logger.Warn("invalid IP range in mattermost allowed_server_ips: lower bound is greater than upper", logger.StringAttr("value", entry))
				continue
			}
			g.ranges = append(g.ranges, ipRange{lo: lo, hi: hi})

		default:
			ip := net.ParseIP(entry)
			if ip == nil {
				logger.Warn("invalid IP in mattermost allowed_server_ips", logger.StringAttr("value", entry))
				continue
			}
			g.ips = append(g.ips, ip)
		}
	}

	if len(g.ips)+len(g.nets)+len(g.ranges) == 0 {
		logger.Warn("mattermost allowed_server_ips is empty: all /mattermost requests will be rejected")
	}

	return &g
}

// middleware — групповая проверка источника всех /mattermost/* запросов.
func (g *sourceGuard) middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !g.allow(c.Request.RemoteAddr) {
			logger.Warn("rejected mattermost request from untrusted source",
				logger.StringAttr("remote_addr", c.Request.RemoteAddr),
			)
			response.SendError(c, models.ErrUntrustedSource)
			return
		}
		c.Next()
	}
}

// allow проверяет IP из RemoteAddr (host:port) по allowlist.
func (g *sourceGuard) allow(remoteAddr string) bool {
	host, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		host = remoteAddr
	}

	ip := toComparableIP(net.ParseIP(host))
	if ip == nil {
		return false
	}

	for _, allowed := range g.ips {
		if ip.Equal(allowed) {
			return true
		}
	}
	for _, network := range g.nets {
		if network.Contains(ip) {
			return true
		}
	}
	for _, r := range g.ranges {
		if bytes.Compare(ip, toComparableIP(r.lo)) >= 0 && bytes.Compare(ip, toComparableIP(r.hi)) <= 0 {
			return true
		}
	}

	return false
}

// toComparableIP приводит адрес к единому байтовому представлению (IPv4 — 4 байта),
// чтобы bytes.Equal/Compare корректно работали с нормализованной (16-байтовой) формой.
func toComparableIP(ip net.IP) net.IP {
	if v4 := ip.To4(); v4 != nil {
		return v4
	}
	return ip
}
