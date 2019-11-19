package bytepoolzap

import (
	"fmt"

	"github.com/bufbuild/buf/internal/pkg/bytepool"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// LogListStats logs list stats and unrecycled elements.
func LogListStats(logger *zap.Logger, level zapcore.Level, segList *bytepool.SegList) {
	if checkedEntry := logger.Check(level, "bytepool"); checkedEntry != nil {
		var unrecycled uint64
		var fields []zap.Field
		for _, listStats := range segList.ListStats() {
			fields = append(
				fields,
				zap.Any(
					fmt.Sprintf("list_stats_%d", int(listStats.ListSize)),
					listStats,
				),
			)
			unrecycled += listStats.TotalUnrecycled
		}
		if unrecycled != 0 {
			fields = append(fields, zap.Uint64("unrecycled", unrecycled))
		}
		if len(fields) > 0 {
			checkedEntry.Write(fields...)
		}
	}
}
