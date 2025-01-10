package statementdistribution

// import (
// 	parachainutil "github.com/ChainSafe/gossamer/dot/parachain/util"
// )

// func createStatementDistribution() (*StatementDistribution, chan parachainutil.NetworkBridgeTxMessage) {
// 	mockAggregator := parachainutil.NewReputationAggregator(func(rep parachainutil.UnifiedReputationChange) bool {
// 		return rep.Type == parachainutil.Malicious
// 	})

// 	subSystemToOverseer := make(chan parachainutil.NetworkBridgeTxMessage, 10)

// 	return &StatementDistribution{
// 		reputationAggregator: mockAggregator,
// 		SubSystemToOverseer:  subSystemToOverseer,
// 	}, subSystemToOverseer
// }

// func createChannels() (chan any, chan any, chan any, chan any) {
// 	return make(chan any, 10), make(chan any, 10), make(chan any, 10), make(chan any, 10)
// }
