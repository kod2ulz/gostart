package storage

import (
	"context"
	"fmt"
	"log/slog"
	"regexp"
	"runtime"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/tracelog"
	"github.com/kod2ulz/gostart/logr"
)

// pgxLoggerAdapter implements the pgx/v5 tracelog.Logger interface.
// It adapts pgx log messages to our structured logr.Logger.
type pgxLoggerAdapter struct {
	log        *logr.Logger
	callerSkip int // Number of stack frames to skip to find the real caller
}

// Log translates a pgx log call to a logr log call.
func (p *pgxLoggerAdapter) Log(ctx context.Context, level tracelog.LogLevel, msg string, data map[string]interface{}) {
	var slogLevel slog.Level
	switch level {
	case tracelog.LogLevelTrace, tracelog.LogLevelDebug:
		slogLevel = slog.LevelDebug
	case tracelog.LogLevelInfo:
		slogLevel = slog.LevelInfo
	case tracelog.LogLevelWarn:
		slogLevel = slog.LevelWarn
	case tracelog.LogLevelError:
		slogLevel = slog.LevelError
	default:
		slogLevel = slog.LevelInfo // Default to Info
	}

	if !p.log.Enabled(ctx, slogLevel) {
		return
	}

	// Parse commandTag to extract rows affected
	if commandTag, ok := data["commandTag"].(string); ok {
		if rowsAffected := extractRowsAffected(commandTag); rowsAffected >= 0 {
			data["rowsAffected"] = rowsAffected
		}
	}

	// Get caller PC (program counter) for correct source attribution
	// Walk the stack to find the first non-pgx frame
	pc := p.findUserCaller()

	// Build slog.Attr from data with camelCase keys
	attrs := make([]slog.Attr, 0, len(data))

	// Add source information at the beginning so it appears early in logs
	if pc != 0 {
		fn := runtime.FuncForPC(pc)
		if fn != nil {
			file, line := fn.FileLine(pc)
			// Use Group for nested attributes
			attrs = append(attrs, slog.Group("source",
				slog.String("function", fn.Name()),
				slog.String("file", file),
				slog.Int("line", line),
			))
		}
	}

	for k, v := range data {
		attrs = append(attrs, slog.Any(toCamelCase(k), v))
	}

	// Create the log record WITHOUT a PC (we're handling source manually)
	rec := slog.NewRecord(time.Now(), slogLevel, msg, 0)
	rec.AddAttrs(attrs...)

	// Log using the handler directly
	_ = p.log.Handler().Handle(ctx, rec)
}

// findUserCaller walks the stack to find the first frame that's not pgx internal
func (p *pgxLoggerAdapter) findUserCaller() uintptr {
	// Get a larger stack to walk through
	pcs := make([]uintptr, 20)
	n := runtime.Callers(2, pcs) // Skip: runtime.Callers, findUserCaller
	if n == 0 {
		return 0
	}

	// Walk through the stack to find the first non-pgx frame
	for i := 0; i < n; i++ {
		fn := runtime.FuncForPC(pcs[i])
		if fn == nil {
			continue
		}

		funcName := fn.Name()
		// Skip pgx internal functions
		if strings.Contains(funcName, "github.com/jackc/pgx") ||
		   strings.Contains(funcName, "database/sql") ||
		   strings.Contains(funcName, "runtime.") {
			continue
		}

		// Found a user frame!
		return pcs[i]
	}

	// If we couldn't find a user frame, return the first frame after the logger
	return pcs[0]
}

// toCamelCase converts snake_case to camelCase
func toCamelCase(s string) string {
	parts := strings.Split(s, "_")
	for i := 1; i < len(parts); i++ {
		if len(parts[i]) > 0 {
			parts[i] = strings.ToUpper(parts[i][:1]) + parts[i][1:]
		}
	}
	return strings.Join(parts, "")
}

// extractRowsAffected parses the PostgreSQL command tag to extract rows affected
// Command tag formats:
// - "SELECT 0" -> 0 rows
// - "INSERT 0 1" -> 1 row (format: INSERT oid rows)
// - "UPDATE 1" -> 1 row
// - "DELETE 5" -> 5 rows
// - "MOVE 1" -> 1 row (cursor)
// - "FETCH 1" -> 1 row
// - "COPY 100" -> 100 rows
func extractRowsAffected(commandTag string) int64 {
	// Handle INSERT with oid: "INSERT 0 1" -> 1 row
	reInsert := regexp.MustCompile(`^INSERT \d+ (\d+)$`)
	if matches := reInsert.FindStringSubmatch(commandTag); len(matches) > 1 {
		var rows int64
		fmt.Sscanf(matches[1], "%d", &rows)
		return rows
	}

	// Handle other commands: "SELECT 0", "UPDATE 1", "DELETE 5", etc.
	reOther := regexp.MustCompile(`^(SELECT|UPDATE|DELETE|MOVE|FETCH|COPY) (\d+)$`)
	if matches := reOther.FindStringSubmatch(commandTag); len(matches) > 2 {
		var rows int64
		fmt.Sscanf(matches[2], "%d", &rows)
		return rows
	}

	// Special cases that don't return row counts
	// BEGIN, COMMIT, ROLLBACK, etc.
	switch commandTag {
	case "BEGIN", "COMMIT", "ROLLBACK":
		return 0
	}

	return -1 // Unknown format
}

// NewPgxLogger creates a tracer for pgx/v5 that is compatible with the application's logr logger.
func NewPgxLogger(log *logr.Logger) *tracelog.TraceLog {
	return &tracelog.TraceLog{
		Logger:   &pgxLoggerAdapter{log: log},
		LogLevel: tracelog.LogLevelInfo, // You can make this configurable
	}
}
