package main

import (
	"encoding/json"
	"net/http"
)

// isAuthed 判断当前 HTTP 请求的 Cookie 中是否带有合法的认证信息。
// 若 config.Password 为空，则视为不启用密码保护，所有请求均视为已认证。
func isAuthed(r *http.Request) bool {
	if config.Password == "" {
		return true
	}
	cookie, err := r.Cookie("auth")
	return err == nil && cookie.Value == authToken
}

// requireAuth 页面路由级别的认证中间件：拦截未授权请求并重定向到登录页
func requireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !isAuthed(r) {
			w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate")
			w.Header().Set("CDN-Cache-Control", "no-store")
			// 将当前用户试图访问的页面路径带在 redirect 参数里
			http.Redirect(w, r, "/login?redirect="+r.URL.Path, http.StatusFound)
			return
		}
		next(w, r)
	}
}

// requireAuthAPI API 级别的认证中间件：拦截未授权请求并返回 401 JSON
func requireAuthAPI(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !isAuthed(r) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(401)
			json.NewEncoder(w).Encode(map[string]string{"error": "未登录"})
			return
		}
		next(w, r)
	}
}

// requirePublicOrAuth 动态页面级别的认证：如果 public_dashboard 为 false 且未登录，重定向到登录页
func requirePublicOrAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !config.PublicDashboard && !isAuthed(r) {
			w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate")
			w.Header().Set("CDN-Cache-Control", "no-store")
			http.Redirect(w, r, "/login?redirect="+r.URL.Path, http.StatusFound)
			return
		}
		next(w, r)
	}
}

// requirePublicOrAuthAPI 动态 API 级别的认证：如果 public_dashboard 为 false 且未登录，返回 401
func requirePublicOrAuthAPI(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !config.PublicDashboard && !isAuthed(r) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(401)
			json.NewEncoder(w).Encode(map[string]string{"error": "未登录或面板未公开"})
			return
		}
		next(w, r)
	}
}

// requirePanelRequest 防护中间件：仅放行来自面板前端 (携带 X-Panel = 1 头) 的请求
// 防止非面板发起的恶意扫描和接口调用
func requirePanelRequest(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Panel") != "1" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(403)
			json.NewEncoder(w).Encode(map[string]string{"error": "forbidden"})
			return
		}
		next(w, r)
	}
}
