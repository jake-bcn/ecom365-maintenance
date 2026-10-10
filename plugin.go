package ecom365_maintenance

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"regexp"
	"strings"
)

// Config 插件配置
type Config struct {
	IsEnable                   bool     `json:"isEnable,omitempty"`
	MaintenanceHosts           []string `json:"maintenanceHosts,omitempty"`
	IsWholeSiteMaintenance     bool     `json:"isWholeSiteMaintenance,omitempty"`
	FrontendMaintenancePath    []string `json:"frontendMaintenancePage,omitempty"`
	MerchantMaintenance        []string `json:"merchantMaintenance,omitempty"`
	AdminApiMaintenancePath    []string `json:"apiMaintenance,omitempty"`
	MerchantApiMaintenancePath []string `json:"merchantApiMaintenance,omitempty"`
	IsDebug                    bool     `json:"isDebug,omitempty"`
	IsFrontendBaseMaintenance  bool     `json:"isFrontendBaseMaintenance,omitempty"`
	ServiceMaintenancePath     []string `json:"serviceMaintenance,omitempty"`
	IgnoreCookieName           string   `json:"ignoreCookieName,omitempty"`
	IsCmsClientEnable          bool     `json:"isCmsClientEnable,omitempty"`
	CmsClientHost              string   `json:"cmsClientHost,omitempty"`
}

// CreateConfig 创建默认配置
func CreateConfig() *Config {
	return &Config{
		IsEnable:                   false,
		MaintenanceHosts:           []string{},
		IsWholeSiteMaintenance:     false,
		FrontendMaintenancePath:    []string{},
		MerchantMaintenance:        []string{},
		AdminApiMaintenancePath:    []string{},
		MerchantApiMaintenancePath: []string{},
		ServiceMaintenancePath:     []string{},
		IsFrontendBaseMaintenance:  false,
		IsDebug:                    false,
		IgnoreCookieName:           "",
		IsCmsClientEnable:          false,
		CmsClientHost:              "",
	}
}

// MaintenancePlugin 日志插件结构体
type MaintenancePlugin struct {
	next   http.Handler
	name   string
	config *Config
}

// New 创建新的插件实例
func New(ctx context.Context, next http.Handler, config *Config, name string) (http.Handler, error) {
	return &MaintenancePlugin{
		next:   next,
		name:   name,
		config: config,
	}, nil
}

// ServeHTTP 处理HTTP请求
func (p *MaintenancePlugin) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	path := req.URL.Path
	isMaintenance := false

	if p.config.IgnoreCookieName != "" {
		if cookie, err := req.Cookie(p.config.IgnoreCookieName); err == nil && cookie.Value != "" {
			isMaintenance = false
			p.next.ServeHTTP(rw, req)
			return
		}
	}

	if p.config.IsEnable {
		if isMaintenanceHost(req.Host, p.config.MaintenanceHosts) {
			isMaintenance = true
		} else if p.config.IsWholeSiteMaintenance {
			isMaintenance = true
		} else if p.config.IsCmsClientEnable && req.Host == p.config.CmsClientHost {
			// cms client
			isMaintenance = true
		} else {
			// /api/merchant/demo/order/order/dashboard/statistics
			// 判斷是否是  merchant api /api/merchant/{merchant_code}/{service_code}
			// 正則判斷 isMerchantAPi,  提取 merchant code 和 service code
			isMerchantAPI, merchantCode, serviceCode := isMerchantAPI(path)
			fmt.Println(isMerchantAPI, merchantCode, serviceCode, path, isInArray(p.config.MerchantMaintenance, merchantCode))
			if isMerchantAPI {
				// 判斷是否 在  MerchantMaintenance 中， 如果是， 就設定 isMaintenance = true
				if isInArray(p.config.MerchantMaintenance, merchantCode) {
					isMaintenance = true
				} else if isInArray(p.config.ServiceMaintenancePath, serviceCode) {
					isMaintenance = true
				} else if isInArray(p.config.MerchantApiMaintenancePath, merchantCode+"/"+serviceCode) {
					isMaintenance = true
				}
			} else {
				isAdminAPI, adminServiceCode := isAdminAPI(path)
				if isAdminAPI {
					if isInArray(p.config.AdminApiMaintenancePath, adminServiceCode) {
						isMaintenance = true
					}
				} else {
					// 如果 base 前端 維護模式 開啟， 所有前端都會進入維護模式
					if p.config.IsFrontendBaseMaintenance {
						isMaintenance = true
					} else {
						isFrontendAPI, frontendServiceCode := isFrontendAPI(path)
						if isFrontendAPI {
							if isInArray(p.config.FrontendMaintenancePath, frontendServiceCode) {
								isMaintenance = true
							}
						}
					}

				}
			}
		}
	}
	if isMaintenance {
		maintenancePage := getMaintenancePage()
		rw.Header().Set("x-ecom365-maintenance", "true")
		rw.WriteHeader(http.StatusServiceUnavailable)
		rw.Write([]byte(maintenancePage))
		log.Printf("[DEBUG] %s %s in maintenance mode", req.Method, path)
		return
	}
	if CreateConfig().IsDebug {
		log.Printf("[DEBUG] %s %s pass", req.Method, path)
	}
	p.next.ServeHTTP(rw, req)
}

func isMaintenanceHost(requestHost string, maintenanceHosts []string) bool {
	normalizedRequestHost := normalizeHostname(requestHost)
	if normalizedRequestHost == "" {
		return false
	}

	for _, maintenanceHost := range maintenanceHosts {
		if normalizedRequestHost == normalizeHostname(maintenanceHost) {
			return true
		}
	}

	return false
}

func normalizeHostname(host string) string {
	host = strings.TrimSpace(strings.ToLower(host))
	if host == "" {
		return ""
	}

	if parsedHost, _, err := net.SplitHostPort(host); err == nil {
		host = parsedHost
	} else {
		host = strings.TrimPrefix(host, "[")
		host = strings.TrimSuffix(host, "]")
	}

	return strings.TrimSuffix(host, ".")
}

// responseWriter 自定义ResponseWriter用于捕获状态码
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

// WriteHeader 重写WriteHeader方法以捕获状态码
func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// 正则表达式匹配 /api/merchant/{merchant_code}/{service_code}
var merchantAPIPattern = regexp.MustCompile(`^/api/merchant/([^/]+)/([^/]+)`)

func isMerchantAPI(path string) (bool, string, string) {
	matches := merchantAPIPattern.FindStringSubmatch(path)
	if len(matches) == 3 {
		return true, matches[1], matches[2] // merchant_code, service_code
	}
	return false, "", ""
}

// 正则表达式匹配 /api/{admin_service_code}, 而且  admin_service_code 不能為merchant

var adminAPIPattern = regexp.MustCompile(`^/api/([^/]+)`)

func isAdminAPI(path string) (bool, string) {
	matches := adminAPIPattern.FindStringSubmatch(path)
	if len(matches) == 2 {
		adminServiceCode := matches[1]
		if adminServiceCode != "merchant" {
			return true, adminServiceCode
		}
	}
	return false, ""
}

// /merchant/{frontend_service}/index/*
// 正则表达式匹配 /merchant/{frontend_service}/index/*
var frontendAPIPattern = regexp.MustCompile(`^/merchant/([^/]+)/.*`)

func isFrontendAPI(path string) (bool, string) {
	matches := frontendAPIPattern.FindStringSubmatch(path)
	if len(matches) == 2 {
		frontendServiceCode := matches[1]
		return true, frontendServiceCode
	}
	return false, ""
}

func isInArray(arr []string, str string) bool {
	for _, v := range arr {
		if v == str {
			return true
		}
	}
	return false
}
