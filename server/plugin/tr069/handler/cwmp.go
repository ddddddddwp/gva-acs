package handler

import (
	"context"
	"io"
	"net/http"

	"github.com/ddddddddwp/gva-acs/server/plugin/tr069/service"
	"github.com/ddddddddwp/tr069-core-only/factory"
	tr069 "github.com/ddddddddwp/tr069-core-only/interface"
	"github.com/gin-gonic/gin"
)

var cwmpService = new(service.CWMPService)

func CWMPHandler(c *gin.Context) {
	// 1. Read Body
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.Status(http.StatusBadRequest)
		return
	}

	// 2. Parse using Real SDK
	p := factory.NewParser(tr069.WithStrictMode(false))
	msg, err := p.ParseMessage(context.Background(), body)
	if err != nil {
		c.Status(http.StatusBadRequest)
		return
	}

	// 3. Process Business Logic
	respMsg, err := cwmpService.HandleMessage(msg, c.ClientIP())
	if err != nil {
		c.Status(http.StatusInternalServerError)
		return
	}

	// 4. Build Response using Real SDK
	b := factory.NewBuilder()
	respBytes, err := b.BuildMessage(context.Background(), respMsg)
	if err != nil {
		c.Status(http.StatusInternalServerError)
		return
	}

	// 5. Send Response
	c.Data(http.StatusOK, "text/xml", respBytes)
}
