package adapter

import "github.com/ddddddddwp/gva-acs/server/global"

func RedisAvailable() bool { return global.GVA_REDIS != nil }
