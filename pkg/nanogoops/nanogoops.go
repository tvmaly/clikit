package nanogoops

import (
	"context"
	"encoding/json"

	clierrors "github.com/tvmaly/clikit/pkg/errors"
	"github.com/tvmaly/clikit/pkg/ops"
)

// Tool is intentionally shaped like Nanogo's current core/tools.Tool interface
// without importing Nanogo into clikit.
type Tool struct {
	op ops.Operation
}

func Adapt(op ops.Operation) Tool {
	return Tool{op: op}
}

func (t Tool) Name() string {
	return t.op.FullName()
}

func (t Tool) Schema() json.RawMessage {
	return t.op.InputSchema
}

func (t Tool) Call(ctx context.Context, args json.RawMessage) (string, error) {
	result, err := t.op.Call(ctx, args)
	if err != nil {
		payload := clierrors.CLIError{
			Message:    err.Error(),
			Suggestion: "check the tool schema and arguments before retrying",
			Retry:      result != nil && result.Retry,
			ExitCode:   1,
		}
		if result != nil && result.ExitCode != 0 {
			payload.ExitCode = result.ExitCode
		}
		data, marshalErr := json.Marshal(payload)
		if marshalErr != nil {
			return "", err
		}
		return string(data), err
	}
	if result == nil {
		return "", nil
	}
	return string(result.Body), nil
}
