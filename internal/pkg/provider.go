package provider

import (
	"github.com/google/wire"
	"github.com/wyuhsin/web-template-go/internal/pkg/dbx"
	"github.com/wyuhsin/web-template-go/internal/pkg/discoveryx"
	"github.com/wyuhsin/web-template-go/internal/pkg/grpcx"
	"github.com/wyuhsin/web-template-go/internal/pkg/messagingx"
)

var ProviderSet = wire.NewSet(
	dbx.ProviderSet,
	discoveryx.ProviderSet,
	grpcx.ProviderSet,
	messagingx.ProviderSet,
)
