package discoveryx

import "github.com/google/wire"

var ProviderSet = wire.NewSet(
	NewKubernetesRegistrar,
)
