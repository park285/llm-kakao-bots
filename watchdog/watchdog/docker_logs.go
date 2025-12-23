package watchdog

import (
	"bufio"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/moby/moby/client"
)

// OpenDockerLogs opens a log stream for a container.
func (w *Watchdog) OpenDockerLogs(ctx context.Context, containerName string, tail int, follow bool, timestamps bool) (io.ReadCloser, bool, error) {
	inspectCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	inspectResult, err := w.cli.ContainerInspect(inspectCtx, containerName, client.ContainerInspectOptions{})
	if err != nil {
		return nil, false, fmt.Errorf("docker inspect failed: %w", err)
	}
	isTTY := false
	if inspectResult.Container.Config != nil {
		isTTY = inspectResult.Container.Config.Tty
	}

	tailValue := "200"
	if tail > 0 {
		tailValue = fmt.Sprintf("%d", tail)
	}

	logsCtx := ctx
	var logsCancel context.CancelFunc
	if !follow {
		logsCtx, logsCancel = context.WithTimeout(ctx, 30*time.Second)
		defer logsCancel()
	}

	reader, err := w.cli.ContainerLogs(logsCtx, containerName, client.ContainerLogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Timestamps: timestamps,
		Follow:     follow,
		Tail:       tailValue,
	})
	if err != nil {
		return nil, isTTY, fmt.Errorf("docker logs failed: %w", err)
	}
	return reader, isTTY, nil
}

// WrapDockerLogsReader wraps a docker log reader to handle multiplexing if necessary.
func WrapDockerLogsReader(reader io.ReadCloser, isTTY bool) io.ReadCloser {
	if isTTY {
		return reader
	}
	return &dockerMultiplexedReader{r: reader, closeFn: reader.Close}
}

type dockerMultiplexedReader struct {
	r       io.Reader
	closeFn func() error

	buf    []byte
	offset int
}

func (r *dockerMultiplexedReader) Close() error {
	if r.closeFn == nil {
		return nil
	}
	return r.closeFn()
}

func (r *dockerMultiplexedReader) Read(p []byte) (int, error) {
	for {
		if r.offset < len(r.buf) {
			n := copy(p, r.buf[r.offset:])
			r.offset += n
			return n, nil
		}

		header := make([]byte, 8)
		_, err := io.ReadFull(r.r, header)
		if err != nil {
			if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
				return 0, io.EOF
			}
			return 0, err
		}

		frameSize := binary.BigEndian.Uint32(header[4:8])
		if frameSize == 0 {
			r.buf = nil
			r.offset = 0
			continue
		}

		payload := make([]byte, frameSize)
		if _, err := io.ReadFull(r.r, payload); err != nil {
			if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
				return 0, io.EOF
			}
			return 0, err
		}

		r.buf = payload
		r.offset = 0
	}
}

// StreamLines reads lines from the reader and calls onLine for each line.
func StreamLines(ctx context.Context, r io.Reader, onLine func(line string) error) error {
	scanner := bufio.NewScanner(r)
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 2*1024*1024)
	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		if err := onLine(scanner.Text()); err != nil {
			return err
		}
	}
	if err := scanner.Err(); err != nil {
		return err
	}
	return nil
}