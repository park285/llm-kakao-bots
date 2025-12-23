//go:build wireinject

package app

import "github.com/google/wire"

var turtleSoupProviderSet = wire.NewSet(
	newTurtleSoupDataRedis,
	newTurtleSoupMQValkey,
	newTurtleSoupRestClient,
	newTurtleSoupMessageProvider,
	newTurtleSoupStores,
	newTurtleSoupInjectionGuard,
	newTurtleSoupReplyPublisher,
	newTurtleSoupServices,
	newTurtleSoupGameService,
	newTurtleSoupStreamConsumer,
	newTurtleSoupMQPipeline,
	newTurtleSoupHTTPMux,
	newTurtleSoupHTTPServer,
	newTurtleSoupServerApp,
)
