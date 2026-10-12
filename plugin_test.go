package ecom365_maintenance

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestMaintenancePageUsesGaxasBranding(t *testing.T) {
	page := getMaintenancePage()

	for _, expected := range []string{
		"data:image/svg+xml;base64,",
		"class=\"brand_logo\"",
		"alt=\"Gaxas.ai\"",
		"href=\"https://www.gaxas.ai/\"",
		">Gaxas.ai</a>",
	} {
		if !strings.Contains(page, expected) {
			t.Fatalf("maintenance page does not contain %q", expected)
		}
	}

	if strings.Contains(page, "e-Com365.com") {
		t.Fatal("maintenance page still contains the old footer branding")
	}
}

func TestHostMaintenanceReturns503ForConfiguredHost(t *testing.T) {
	config := CreateConfig()
	config.IsEnable = true
	config.MaintenanceHosts = []string{"365staging.gaxas.ai"}

	next := http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		rw.WriteHeader(http.StatusOK)
	})
	plugin, err := New(context.Background(), next, config, "test")
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "https://365staging.gaxas.ai/any/path", nil)
	recorder := httptest.NewRecorder()
	plugin.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusServiceUnavailable)
	}
	if recorder.Header().Get("x-ecom365-maintenance") != "true" {
		t.Fatal("x-ecom365-maintenance header was not set")
	}
}

func TestHostMaintenanceNormalizesHostnameAndPort(t *testing.T) {
	config := CreateConfig()
	config.IsEnable = true
	config.MaintenanceHosts = []string{"365staging.gaxas.ai"}

	next := http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		rw.WriteHeader(http.StatusOK)
	})
	plugin, err := New(context.Background(), next, config, "test")
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "https://unused.example/any/path", nil)
	req.Host = "365STAGING.GAXAS.AI:443"
	recorder := httptest.NewRecorder()
	plugin.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusServiceUnavailable)
	}
}

func TestHostMaintenanceDoesNotAffectOtherHost(t *testing.T) {
	config := CreateConfig()
	config.IsEnable = true
	config.MaintenanceHosts = []string{"365staging.gaxas.ai"}

	next := http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		rw.WriteHeader(http.StatusOK)
	})
	plugin, err := New(context.Background(), next, config, "test")
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "https://other.example/any/path", nil)
	recorder := httptest.NewRecorder()
	plugin.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}
}

func TestDisabledPluginIgnoresMaintenanceHosts(t *testing.T) {
	config := CreateConfig()
	config.IsEnable = false
	config.MaintenanceHosts = []string{"365staging.gaxas.ai"}

	next := http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		rw.WriteHeader(http.StatusOK)
	})
	plugin, err := New(context.Background(), next, config, "test")
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "https://365staging.gaxas.ai/any/path", nil)
	recorder := httptest.NewRecorder()
	plugin.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}
}
