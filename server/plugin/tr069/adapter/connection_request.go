package adapter

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/ddddddddwp/gva-acs/server/global"
	"github.com/ddddddddwp/gva-acs/server/plugin/tr069/infolog"
	"github.com/ddddddddwp/gva-acs/server/plugin/tr069/model"
	"go.uber.org/zap"
)

type ConnectionRequestResult struct {
	URL        string
	StatusCode int
	Elapsed    time.Duration
	Err        error
}

type ConnectionRequestConfig struct {
	Timeout time.Duration
	Retries int
}

func TriggerConnectionRequest(ctx context.Context, deviceID uint, cfg ConnectionRequestConfig) ConnectionRequestResult {
	if cfg.Timeout <= 0 {
		cfg.Timeout = 5 * time.Second
	}
	if cfg.Retries < 0 {
		cfg.Retries = 0
	}

	connURL, user, pass, ok := connectionRequestTarget(ctx, deviceID)
	if !ok {
		return ConnectionRequestResult{Err: fmt.Errorf("missing connection request url for deviceId=%d", deviceID)}
	}

	var lastErr error
	for attempt := 0; attempt <= cfg.Retries; attempt++ {
		start := time.Now()
		status, err := doConnectionRequest(ctx, connURL, user, pass, cfg.Timeout)
		elapsed := time.Since(start)
		res := ConnectionRequestResult{
			URL:        connURL,
			StatusCode: status,
			Elapsed:    elapsed,
			Err:        err,
		}
		writeConnectionRequestLog(res, deviceID, attempt)
		if err == nil {
			return res
		}
		lastErr = err
	}
	return ConnectionRequestResult{URL: connURL, Err: lastErr}
}

func connectionRequestTarget(ctx context.Context, deviceID uint) (connURL string, user string, pass string, ok bool) {
	if global.GVA_DB == nil {
		return "", "", "", false
	}
	var d model.Device
	if err := global.GVA_DB.WithContext(ctx).Select("id", "connection_req_url").First(&d, deviceID).Error; err != nil {
		return "", "", "", false
	}
	if strings.TrimSpace(d.ConnectionReqURL) == "" {
		return "", "", "", false
	}
	connURL = strings.TrimSpace(d.ConnectionReqURL)

	u, err := url.Parse(connURL)
	if err == nil && u != nil && u.User != nil {
		user = u.User.Username()
		pass, _ = u.User.Password()
	}
	if user == "" {
		if u2, p2, ok := loadConnectionRequestCreds(ctx, deviceID); ok {
			user, pass = u2, p2
		}
	}
	return connURL, user, pass, true
}

func loadConnectionRequestCreds(ctx context.Context, deviceID uint) (string, string, bool) {
	if global.GVA_DB == nil {
		return "", "", false
	}
	var rows []model.DataModelValue
	if err := global.GVA_DB.WithContext(ctx).
		Select("name", "value_json").
		Where("device_id = ? AND name IN ?", deviceID, []string{
			"Device.ManagementServer.ConnectionRequestUsername",
			"Device.ManagementServer.ConnectionRequestPassword",
		}).Find(&rows).Error; err != nil {
		return "", "", false
	}
	var user string
	var pass string
	for _, r := range rows {
		var v string
		_ = json.Unmarshal(r.ValueJSON, &v)
		switch r.Name {
		case "Device.ManagementServer.ConnectionRequestUsername":
			user = v
		case "Device.ManagementServer.ConnectionRequestPassword":
			pass = v
		}
	}
	if user == "" {
		return "", "", false
	}
	return user, pass, true
}

func doConnectionRequest(ctx context.Context, connURL string, user string, pass string, timeout time.Duration) (int, error) {
	cctx := ctx
	if cctx == nil {
		cctx = context.Background()
	}
	cctx, cancel := context.WithTimeout(cctx, timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(cctx, http.MethodGet, connURL, nil)
	if err != nil {
		return 0, err
	}
	req.Header.Set("User-Agent", "gva-tr069-acs/connection-request")
	if user != "" {
		req.SetBasicAuth(user, pass)
	}

	client := &http.Client{
		Timeout: timeout,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{MinVersion: tls.VersionTLS12},
		},
	}
	resp, err := client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)
	return resp.StatusCode, nil
}

func writeConnectionRequestLog(res ConnectionRequestResult, deviceID uint, attempt int) {
	status := "OK"
	if res.Err != nil {
		status = "ERR"
	}
	s := fmt.Sprintf("----- TR069 CONNECTION REQUEST BEGIN -----\nstatus: %s\ndeviceId: %d\nattempt: %d\nurl: %s\nhttpStatus: %d\nelapsed: %s\nerror: %v\n----- TR069 CONNECTION REQUEST END -----",
		status, deviceID, attempt, res.URL, res.StatusCode, res.Elapsed.String(), res.Err)
	_, _ = fmt.Fprintln(os.Stdout, s)
	infolog.Write(s)
	if res.Err != nil && global.GVA_LOG != nil {
		global.GVA_LOG.Warn("TR069 connection request failed", zap.Uint("deviceId", deviceID), zap.String("url", res.URL), zap.Int("attempt", attempt), zap.Error(res.Err))
	}
}
