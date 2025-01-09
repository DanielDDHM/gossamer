package statementdistribution

import (
	"context"

	parachainutil "github.com/ChainSafe/gossamer/dot/parachain/util"
)

func CreateStatementDistribution() (*StatementDistribution, chan parachainutil.NetworkBridgeTxMessage) {
	mockAggregator := parachainutil.NewReputationAggregator(func(rep parachainutil.UnifiedReputationChange) bool {
		return rep.Type == parachainutil.Malicious
	})

	subSystemToOverseer := make(chan parachainutil.NetworkBridgeTxMessage, 10)

	return &StatementDistribution{
		reputationAggregator: *mockAggregator,
		SubSystemToOverseer:  subSystemToOverseer,
	}, subSystemToOverseer
}

func CreateChannels() (chan any, chan any, chan any, chan any) {
	return make(chan any, 10), make(chan any, 10), make(chan any, 10), make(chan any, 10)
}

func CreateContext() (context.Context, context.CancelFunc) {
	return context.WithCancel(context.Background())
}
