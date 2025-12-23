//go:build wireinject

package app

import "github.com/google/wire"

var twentyQProviderSet = wire.NewSet(
	newTwentyQDataRedis,
	newTwentyQMQValkey,
	newTwentyQRestClient,
	newTwentyQMessageProvider,
	newTwentyQDB,
	newTwentyQRepository,
	newTwentyQStatsRecorder,
	newTwentyQStores,
	newTwentyQRiddleService,
	newTwentyQAdminServices,
	newTwentyQMQPipeline,
	newTwentyQHTTPMux,
	newTwentyQHTTPServer,
	newTwentyQServerApp,
)
