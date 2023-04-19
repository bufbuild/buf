package command

import (
	"context"
	"os"

	"go.uber.org/multierr"
)

type process struct {
	osProcess *os.Process
	runner    *runner
}

func (p *process) Terminate(ctx context.Context) error {
	if err := p.osProcess.Signal(os.Interrupt); err == nil {
		// Successful interrupt sent: give the process some time.
		wait := make(chan error)
		go func() {
			_, err := p.osProcess.Wait()
			wait <- err
		}()
		select {
		case err := <-wait:
			// Process exited. We'll pass on any error.
			<-p.runner.semaphoreC
			return err
		case <-ctx.Done():
			// Timed out. Follow the no-interrupt case.
		}
	}
	// No hope on this host or process. Immediately kill and release the
	// process.
	<-p.runner.semaphoreC
	return multierr.Combine(
		p.osProcess.Kill(),
		p.osProcess.Release(),
	)
}
