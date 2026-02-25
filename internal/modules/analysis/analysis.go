package analysis

import "context"

type Provider interface {
	FileResume(ctx context.Context, input string) (string, error)
}
