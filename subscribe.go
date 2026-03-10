package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

// ============================================================
// Subscription link generation — XHTTP + REALITY / CDN
// ============================================================

// SubscribeLink represents a single subscription node (URI) and its Base64 encoding.
type SubscribeLink struct {
	Name   string `json:"name"`   // The display name of the node
	URI    string `json:"uri"`    // The raw VLESS URL
	Base64 string `json:"base64"` // The Base64 encoded string of the URI
}

// generateSubscribeLinks parses the Xray configuration to find VLESS inbounds,
// searches for the target email's UUID, and generates subscription links
// formatted specifically for Reality or CDN setups.
func generateSubscribeLinks(email string) ([]SubscribeLink, error) {
	if config.XrayConfigPath == "" {
		return nil, fmt.Errorf("未配置 xray_config_path")
	}

	cfg, err := readXrayConfig()
	if err != nil {
		return nil, err
	}

	// Find UUID for this email
	userUUID := ""
	inbounds, ok := cfg["inbounds"].([]interface{})
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

	var links []SubscribeLink

	// Generate links for each VLESS inbound
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

		// Extract settings for encryption (VCN)
		// We no longer extract server 'decryption' because it cannot be converted to client 'encryption' (due to private/public key differences).
		// Instead, we use config.CDNEncryption for CDN inbound.

		// Extract path from xhttpSettings
		path := ""
		mode := ""
		if xhttpSettings, ok := streamSettings["xhttpSettings"].(map[string]interface{}); ok {
			path, _ = xhttpSettings["path"].(string)
			mode, _ = xhttpSettings["mode"].(string)
		}

		params := url.Values{}
		params.Set("type", network)

		// XHTTP mode (if specified)
		if mode != "" {
			params.Set("mode", mode)
		}

		var linkName string
		var addr string

		// Port: 尝试读取配置，默认为 443
		port := config.NodePort
		if port == "" {
			port = "443"
		}

		// FP: 尝试读取配置，默认为 random
		fp := config.NodeFP
		if fp == "" {
			fp = "random"
		}

		// ALPN: 尝试读取配置，如果不为空则附加
		alpn := config.NodeALPN

		switch security {
		case "reality":
			// ===== XHTTP + REALITY =====
			addr = config.RealityAddr
			if addr == "" {
				addr = "YOUR_SERVER_IP"
			}
			params.Set("security", "reality")
			// XHTTP does not use flow
			params.Set("encryption", "none")

			if realitySettings, ok := streamSettings["realitySettings"].(map[string]interface{}); ok {
				// SNI from serverNames
				if serverNames, ok := realitySettings["serverNames"].([]interface{}); ok && len(serverNames) > 0 {
					sni, _ := serverNames[0].(string)
					params.Set("sni", sni)
				}
				// Short ID
				if shortIds, ok := realitySettings["shortIds"].([]interface{}); ok && len(shortIds) > 0 {
					sid, _ := shortIds[0].(string)
					if sid != "" {
						params.Set("sid", sid)
					}
				}
			}

			// Public key — from config or error
			if config.RealityPublicKey != "" {
				params.Set("pbk", config.RealityPublicKey)
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
			// ===== XHTTP + CDN (security=none means TLS terminated by nginx/CDN) =====
			addr = config.CDNDomain
			if addr == "" {
				addr = "YOUR_CDN_DOMAIN"
			}
			// Client connects over TLS to CDN
			params.Set("security", "tls")
			params.Set("sni", config.CDNDomain)

			// VCN encryption: must be configured by user in panel config.json
			// Server's decryption contains private keys, so we CANNOT auto-generate
			// client encryption from it. User must provide the matching client encryption.
			if config.CDNEncryption != "" {
				params.Set("encryption", config.CDNEncryption)
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
			addr = config.RealityAddr
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

		// Build VLESS URI
		encodedParams := strings.ReplaceAll(params.Encode(), "%2C", ",")
		finalRemark := fmt.Sprintf("%s-%s", email, linkName)
		uri := fmt.Sprintf("vless://%s@%s:%s?%s#%s",
			userUUID, addr, port, encodedParams, url.PathEscape(finalRemark))

		links = append(links, SubscribeLink{
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

// handleSubscribe API: GET /admin/api/subscribe?email=...
// Generates Xray subscription links (Vless URIs + Base64 bundle) for the specified user.
func handleSubscribe(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	email := r.URL.Query().Get("email")
	if email == "" {
		json.NewEncoder(w).Encode(map[string]string{"error": "email 参数必须"})
		return
	}

	links, err := generateSubscribeLinks(email)
	if err != nil {
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	// Combined base64 (all URIs concatenated)
	var allURIs []string
	for _, l := range links {
		allURIs = append(allURIs, l.URI)
	}
	combinedBase64 := base64.StdEncoding.EncodeToString([]byte(strings.Join(allURIs, "\n")))

	json.NewEncoder(w).Encode(map[string]interface{}{
		"links":           links,
		"combined_base64": combinedBase64,
	})
}
