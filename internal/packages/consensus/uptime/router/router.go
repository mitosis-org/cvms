package router

import (
	"github.com/cosmostation/cvms/internal/common"

	"github.com/cosmostation/cvms/internal/packages/consensus/uptime/api"
	"github.com/cosmostation/cvms/internal/packages/consensus/uptime/types"
)

func GetStatus(exporter *common.Exporter, p common.Packager) (types.CommonUptimeStatus, error) {
	switch p.ProtocolType {
	case "cosmos":
		if p.IsConsumerChain {
			return api.GetConsumserUptimeStatus(exporter, p.ChainID)
		}
		return api.GetUptimeStatus(exporter)
	case "mitosis":
		// mitosis chain uses cosmos-compatible uptime API but with custom validator fetching
		return api.GetMitosisUptimeStatus(exporter.CommonApp, p.ChainName)
	default:
		return types.CommonUptimeStatus{}, common.ErrOutOfSwitchCases
	}
}
