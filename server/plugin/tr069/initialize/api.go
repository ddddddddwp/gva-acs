package initialize

import (
	"context"
	model "github.com/ddddddddwp/gva-acs/server/model/system"
	"github.com/ddddddddwp/gva-acs/server/plugin/plugin-tool/utils"
)

func Api(ctx context.Context) {
	entities := []model.SysApi{
		// Device APIs
		{Path: "/tr069/device/list", Description: "Get Device List", ApiGroup: "TR069", Method: "GET"},
		{Path: "/tr069/device", Description: "Create Device", ApiGroup: "TR069", Method: "POST"},
		{Path: "/tr069/device/:deviceId", Description: "Delete Device", ApiGroup: "TR069", Method: "DELETE"},

		// FAP APIs
		{Path: "/tr069/fap/:deviceId", Description: "Get FAP Info", ApiGroup: "TR069", Method: "GET"},
		{Path: "/tr069/fap/:deviceId/sync", Description: "Sync FAP Info", ApiGroup: "TR069", Method: "POST"},
		{Path: "/tr069/fap/:deviceId", Description: "Configure FAP", ApiGroup: "TR069", Method: "PUT"},

		// Debug APIs
		{Path: "/tr069/debug/trace/:requestId", Description: "Get Trace", ApiGroup: "TR069", Method: "GET"},

		// Command APIs
		{Path: "/tr069/command/:deviceId/getRPCMethods", Description: "Sync RPC Methods", ApiGroup: "TR069", Method: "POST"},
		{Path: "/tr069/command/:deviceId/getParameterValues", Description: "Get Parameter Values", ApiGroup: "TR069", Method: "POST"},
		{Path: "/tr069/command/:deviceId/setParameterValues", Description: "Set Parameter Values", ApiGroup: "TR069", Method: "POST"},

		// DataModel APIs
		{Path: "/tr069/datamodel/:deviceId/sync", Description: "Full Sync Datamodel", ApiGroup: "TR069", Method: "POST"},
		{Path: "/tr069/datamodel/:deviceId/list", Description: "Get Datamodel List", ApiGroup: "TR069", Method: "GET"},
		{Path: "/tr069/datamodel/:deviceId/structure", Description: "Get Datamodel Structure", ApiGroup: "TR069", Method: "GET"},

		// Alarm APIs
		{Path: "/tr069/alarm/list", Description: "Get Alarm List", ApiGroup: "TR069", Method: "GET"},
	}
	utils.RegisterApis(entities...)
}
