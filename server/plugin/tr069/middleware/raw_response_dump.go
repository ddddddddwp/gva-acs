package middleware

import (
	"fmt"
	"os"
	"time"

	"github.com/ddddddddwp/gva-acs/server/plugin/tr069/config"
	"github.com/ddddddddwp/gva-acs/server/plugin/tr069/trace"
	"github.com/gin-gonic/gin"
)

type RawResponseDumpConfig struct {
	MaxBytes      int
	DumpToConsole bool
}

// RawResponseDump keeps its config parameter for source compatibility; each
// request derives all behavior from one atomic runtime snapshot.
func RawResponseDump(_ RawResponseDumpConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		settings := config.CurrentRuntime().Settings
		if !rawDumpEnabled(settings) {
			c.Next()
			return
		}
		maxBytes := dumpMaxBytes(settings)
		reqID, _ := c.Get("requestId")
		requestID, _ := reqID.(string)

		capture := newResponseCaptureWriter(c.Writer, maxBytes)
		c.Writer = capture

		start := time.Now()
		c.Next()
		elapsed := time.Since(start)

		logBody := sanitizedCWMPLogCopy(capture.body.Bytes())
		respDump := dumpResponse(c.Writer.Status(), c.Writer.Header(), logBody, requestID, elapsed, maxBytes)
		if settings.DumpRaw {
			_, _ = fmt.Fprintln(os.Stdout, respDump)
		}
		if settings.InfoLogEnable {
			writeInfoLog(respDump, settings)
		}
		if requestID != "" {
			trace.Default.Add(requestID, trace.Entry{
				At:      time.Now(),
				Stage:   "raw.response",
				Message: truncateBytes(logBody, maxBytes),
				Fields: map[string]string{
					"status": fmt.Sprintf("%d", c.Writer.Status()),
				},
			})
		}
	}
}
