package middleware

import (
	"fmt"
	"os"
	"time"

	tr069Global "github.com/ddddddddwp/gva-acs/server/plugin/tr069/global"
	"github.com/ddddddddwp/gva-acs/server/plugin/tr069/trace"
	"github.com/gin-gonic/gin"
)

type RawResponseDumpConfig struct {
	MaxBytes      int
	DumpToConsole bool
}

func RawResponseDump(cfg RawResponseDumpConfig) gin.HandlerFunc {
	maxBytes := cfg.MaxBytes
	if maxBytes <= 0 {
		maxBytes = 64 * 1024
	}
	return func(c *gin.Context) {
		reqID, _ := c.Get("requestId")
		requestID, _ := reqID.(string)

		capture := newResponseCaptureWriter(c.Writer, maxBytes)
		c.Writer = capture

		start := time.Now()
		c.Next()
		elapsed := time.Since(start)

		respDump := dumpResponse(c.Writer.Status(), c.Writer.Header(), capture.body.Bytes(), requestID, elapsed, maxBytes)
		if cfg.DumpToConsole {
			_, _ = fmt.Fprintln(os.Stdout, respDump)
		}
		if tr069Global.GlobalConfig != nil && tr069Global.GlobalConfig.InfoLogEnable {
			writeInfoLog(respDump)
		}
		if requestID != "" {
			trace.Default.Add(requestID, trace.Entry{
				At:      time.Now(),
				Stage:   "raw.response",
				Message: truncateBytes(capture.body.Bytes(), maxBytes),
				Fields: map[string]string{
					"status": fmt.Sprintf("%d", c.Writer.Status()),
				},
			})
		}
	}
}
