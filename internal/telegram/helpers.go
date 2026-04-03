package telegram

import (
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"log"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"xray-panel/internal/types"
	"xray-panel/internal/xray"
)

// ===== Xray 服务控制 =====

var xrayMu sync.Mutex

// XrayServiceCmd 执行 systemctl 命令
func XrayServiceCmd(action string) (string, error) {
	xrayMu.Lock()
	defer xrayMu.Unlock()
	out, err := exec.Command("systemctl", action, "xray").CombinedOutput()
	result := strings.TrimSpace(string(out))

	if action == "reload" && err != nil {
		log.Printf("[Admin] Xray reload 失败，自动降级为 restart。原错误: %v, 输出: %s", err, result)
		out, err = exec.Command("systemctl", "restart", "xray").CombinedOutput()
		result = strings.TrimSpace(string(out))
	}
	return result, err
}

// XrayServiceStatus 获取 Xray 服务状态
func XrayServiceStatus() string {
	out, _ := exec.Command("systemctl", "is-active", "xray").Output()
	return strings.TrimSpace(string(out))
}

// ===== 证书解析 =====

// ParseCertificate 读取并解析 PEM 证书
func ParseCertificate(certPath string) (*types.CertInfo, error) {
	data, err := os.ReadFile(certPath)
	if err != nil {
		return nil, fmt.Errorf("读取证书失败: %v", err)
	}
	block, _ := pem.Decode(data)
	if block == nil {
		return nil, fmt.Errorf("证书 PEM 解码失败")
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("解析证书失败: %v", err)
	}
	now := time.Now()
	daysLeft := int(cert.NotAfter.Sub(now).Hours() / 24)
	tz, _ := time.LoadLocation("Asia/Shanghai")
	if tz == nil {
		tz = time.FixedZone("CST", 8*3600)
	}
	return &types.CertInfo{
		Subject:    cert.Subject.CommonName,
		Issuer:     cert.Issuer.CommonName,
		NotBefore:  cert.NotBefore.In(tz).Format("2006-01-02 15:04:05"),
		NotAfter:   cert.NotAfter.In(tz).Format("2006-01-02 15:04:05"),
		DaysLeft:   daysLeft,
		DNSNames:   cert.DNSNames,
		IsExpired:  now.After(cert.NotAfter),
		IsExpiring: daysLeft < 30 && daysLeft >= 0,
	}, nil
}

// ===== 订阅链接生成 =====

// GenerateSubscribeLinks 为指定用户生成订阅链接
func GenerateSubscribeLinks(cfg *types.Config, email string) ([]types.SubscribeLink, error) {
	if cfg.XrayConfigPath == "" {
		return nil, fmt.Errorf("未配置 xray_config_path")
	}

	xrayCfg, err := xray.ReadXrayConfig(cfg.XrayConfigPath)
	if err != nil {
		return nil, err
	}

	userUUID := ""
	inbounds, ok := xrayCfg["inbounds"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("配置中无 inbounds")
	}

	for _, ib := range inbounds {
		ibMap, ok := ib.(map[string]interface{})
		if !ok {
			continue
		}
		protocol, _ := ibMap["protocol"].(string)
		if protocol != "vless" {
			continue
		}
		settings, ok := ibMap["settings"].(map[string]interface{})
		if !ok {
			continue
		}
		clients, ok := settings["clients"].([]interface{})
		if !ok {
			continue
		}
		for _, c := range clients {
			cMap, _ := c.(map[string]interface{})
			if e, _ := cMap["email"].(string); e == email {
				userUUID, _ = cMap["id"].(string)
				break
			}
		}
		if userUUID != "" {
			break
		}
	}

	if userUUID == "" {
		return nil, fmt.Errorf("用户 %s 不存在", email)
	}

	var links []types.SubscribeLink

	for _, ib := range inbounds {
		ibMap, ok := ib.(map[string]interface{})
		if !ok {
			continue
		}
		protocol, _ := ibMap["protocol"].(string)
		if protocol != "vless" {
			continue
		}
		tag, _ := ibMap["tag"].(string)
		streamSettings, ok := ibMap["streamSettings"].(map[string]interface{})
		if !ok {
			continue
		}
		network, _ := streamSettings["network"].(string)
		security, _ := streamSettings["security"].(string)

		path := ""
		mode := ""
		if xhttpSettings, ok := streamSettings["xhttpSettings"].(map[string]interface{}); ok {
			path, _ = xhttpSettings["path"].(string)
			mode, _ = xhttpSettings["mode"].(string)
		}

		params := url.Values{}
		params.Set("type", network)
		if mode != "" {
			params.Set("mode", mode)
		}

		var linkName string
		var addr string

		port := cfg.NodePort
		if port == "" {
			port = "443"
		}
		fp := cfg.NodeFP
		if fp == "" {
			fp = "random"
		}
		alpn := cfg.NodeALPN

		switch security {
		case "reality":
			addr = cfg.RealityAddr
			if addr == "" {
				addr = "YOUR_SERVER_IP"
			}
			params.Set("security", "reality")
			params.Set("encryption", "none")
			if realitySettings, ok := streamSettings["realitySettings"].(map[string]interface{}); ok {
				if serverNames, ok := realitySettings["serverNames"].([]interface{}); ok && len(serverNames) > 0 {
					sni, _ := serverNames[0].(string)
					params.Set("sni", sni)
				}
				if shortIds, ok := realitySettings["shortIds"].([]interface{}); ok && len(shortIds) > 0 {
					sid, _ := shortIds[0].(string)
					if sid != "" {
						params.Set("sid", sid)
					}
				}
			}
			if cfg.RealityPublicKey != "" {
				params.Set("pbk", cfg.RealityPublicKey)
			}
			params.Set("fp", fp)
			if alpn != "" {
				params.Set("alpn", alpn)
			}
			if path != "" {
				params.Set("path", path)
			}
			linkName = tag + " (REALITY)"

		case "none", "":
			addr = cfg.CDNDomain
			if addr == "" {
				addr = "YOUR_CDN_DOMAIN"
			}
			params.Set("security", "tls")
			params.Set("sni", cfg.CDNDomain)
			if cfg.CDNEncryption != "" {
				params.Set("encryption", cfg.CDNEncryption)
			} else {
				params.Set("encryption", "none")
			}
			if path != "" {
				params.Set("path", path)
			}
			params.Set("fp", fp)
			if alpn != "" {
				params.Set("alpn", alpn)
			}
			linkName = tag + " (CDN)"

		default:
			addr = cfg.RealityAddr
			if addr == "" {
				addr = "YOUR_SERVER_IP"
			}
			params.Set("security", security)
			params.Set("encryption", "none")
			params.Set("fp", fp)
			if alpn != "" {
				params.Set("alpn", alpn)
			}
			if path != "" {
				params.Set("path", path)
			}
			linkName = tag
		}

		encodedParams := strings.ReplaceAll(params.Encode(), "%2C", ",")
		finalRemark := fmt.Sprintf("%s-%s", email, linkName)
		uri := fmt.Sprintf("vless://%s@%s:%s?%s#%s",
			userUUID, addr, port, encodedParams, url.PathEscape(finalRemark))

		links = append(links, types.SubscribeLink{
			Name:   linkName,
			URI:    uri,
			Base64: base64.StdEncoding.EncodeToString([]byte(uri)),
		})
	}

	if len(links) == 0 {
		return nil, fmt.Errorf("未找到 VLESS 入站")
	}
	return links, nil
}
