package handler

import (
	"context"
	"github.com/ddddddddwp/gva-acs/server/plugin/tr069/lib/factory"
	"github.com/ddddddddwp/gva-acs/server/plugin/tr069/service"
	"github.com/gin-gonic/gin"
	"io"
	"net/http"
)

var cwmpService = new(service.CWMPService)

func CWMPHandler(c *gin.Context) {
	// 1. Read Body
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.Status(http.StatusBadRequest)
		return
	}

	// 2. Parse using SDK
	p := factory.NewParser()
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

	// 4. Build Response using SDK
	b := factory.NewBuilder()
	respBytes, err := b.BuildMessage(context.Background(), respMsg)
	if err != nil {
		c.Status(http.StatusInternalServerError)
		return
	}

	// 5. Send Response
	c.Data(http.StatusOK, "text/xml", respBytes)
}
