package di

import "github.com/valkey-io/valkey-go"

// DataValkeyClient 는 Wire에서 동일 타입(valkey.Client) 중복 제공 충돌을 피하기 위한 DI wrapper 타입이다.
// Data/MQ 클라이언트를 분리된 타입으로 취급해 의존성 그래프를 명확히 한다.
type DataValkeyClient struct{ valkey.Client }

// MQValkeyClient 는 MQ용 Valkey 클라이언트 DI wrapper 타입이다.
type MQValkeyClient struct{ valkey.Client }
