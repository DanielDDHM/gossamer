package statementdistribution

// import (
// 	parachainutil "github.com/ChainSafe/gossamer/dot/parachain/util"
// )

// func createStatementDistribution() (*StatementDistribution, chan any) {
// 	mockAggregator := parachainutil.NewReputationAggregator(func(rep parachainutil.UnifiedReputationChange) bool {
// 		return rep.Type == parachainutil.Malicious
// 	})

// 	subSystemToOverseer := make(chan any, 10)

// 	return &StatementDistribution{
// 		reputationAggregator: mockAggregator,
// 		SubSystemToOverseer:  subSystemToOverseer,
// 	}, subSystemToOverseer
// }
