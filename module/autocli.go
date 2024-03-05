package module

import (
	autocliv1 "cosmossdk.io/api/cosmos/autocli/v1"

	tiablobv1 "github.com/rollchains/tiablob/api/v1"
)

// AutoCLIOptions implements the autocli.HasAutoCLIConfig interface.
func (am AppModule) AutoCLIOptions() *autocliv1.ModuleOptions {
	return &autocliv1.ModuleOptions{
		Query: &autocliv1.ServiceCommandDescriptor{
			Service: tiablobv1.Query_ServiceDesc.ServiceName,
		},
		Tx: &autocliv1.ServiceCommandDescriptor{
			Service: tiablobv1.Msg_ServiceDesc.ServiceName,
		},
	}
}
