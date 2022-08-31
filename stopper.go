package threads

import (
	"context"
)

type Stoppable interface {
	Stop(context.Context)
}

type StopCombiner []Stoppable

func (s *StopCombiner) Add(stoppable Stoppable) {
	*s = append(*s, stoppable)
}

func (s StopCombiner) Stop(ctx context.Context) {
	for _, stoppable := range s {
		stoppable.Stop(ctx)
	}
}
