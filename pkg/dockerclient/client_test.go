package dockerclient

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"sync/atomic"
	"testing"
	"time"

	"github.com/docker/docker/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type streamErrorConn struct {
	net.Conn
	err error
}

func (c streamErrorConn) Read(p []byte) (int, error) {
	n, err := c.Conn.Read(p)
	if errors.Is(err, io.EOF) && c.err != nil {
		return n, c.err
	}
	return n, err
}

func TestExecCommand(t *testing.T) {
	streamErr := errors.New("exec stream interrupted")
	for _, tc := range []struct {
		name          string
		exitCode      int
		running       bool
		inspectStatus int
		streamErr     error
		wantError     string
	}{
		{name: "successful exit", inspectStatus: http.StatusOK},
		{name: "nonzero exit", exitCode: 23, inspectStatus: http.StatusOK, wantError: "exit code 23"},
		{name: "still running", running: true, inspectStatus: http.StatusOK, wantError: "still running"},
		{name: "inspect failure", inspectStatus: http.StatusInternalServerError, wantError: "inspect failed"},
		{name: "stream failure", streamErr: streamErr, inspectStatus: http.StatusOK, wantError: streamErr.Error()},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var inspections atomic.Int32
			dial := func(ctx context.Context, network, address string) (net.Conn, error) {
				local, remote := net.Pipe()
				_ = local.SetDeadline(time.Now().Add(5 * time.Second))
				_ = remote.SetDeadline(time.Now().Add(5 * time.Second))
				t.Cleanup(func() { _ = local.Close(); _ = remote.Close() })
				go func() {
					defer remote.Close()
					req, err := http.ReadRequest(bufio.NewReader(remote))
					if !assert.NoError(t, err) {
						return
					}
					_, err = io.Copy(io.Discard, req.Body)
					assert.NoError(t, err)
					_ = req.Body.Close()
					status, body := http.StatusCreated, `{"Id":"test-exec"}`
					switch req.URL.Path {
					case "/v1.47/containers/test-container/exec":
						assert.Equal(t, http.MethodPost, req.Method)
					case "/v1.47/exec/test-exec/start":
						_, err = io.WriteString(remote, "HTTP/1.1 101 Switching Protocols\r\nConnection: Upgrade\r\nUpgrade: tcp\r\n\r\nhook output")
						assert.NoError(t, err)
						return
					case "/v1.47/exec/test-exec/json":
						inspections.Add(1)
						status = tc.inspectStatus
						body = fmt.Sprintf(`{"Running":%t,"ExitCode":%d}`, tc.running, tc.exitCode)
						if status != http.StatusOK {
							body = `{"message":"inspect failed"}`
						}
					default:
						t.Errorf("unexpected Docker API request: %s", req.URL.Path)
						return
					}
					_, err = fmt.Fprintf(remote, "HTTP/1.1 %d %s\r\nContent-Type: application/json\r\nContent-Length: %d\r\nConnection: close\r\n\r\n%s", status, http.StatusText(status), len(body), body)
					assert.NoError(t, err)
				}()
				return streamErrorConn{Conn: local, err: tc.streamErr}, nil
			}
			cli, err := client.NewClientWithOpts(client.WithHost("tcp://docker.invalid:2375"), client.WithVersion("1.47"), client.WithDialContext(dial))
			require.NoError(t, err)
			t.Cleanup(func() { _ = cli.Close() })
			c := &DefaultClient{cli: cli}
			err = c.ExecCommand(context.Background(), "test-container", []string{"sh", "-c", "synthetic hook"})
			if tc.wantError == "" {
				assert.NoError(t, err)
			} else {
				assert.ErrorContains(t, err, tc.wantError)
				assert.NotContains(t, fmt.Sprint(err), "synthetic hook")
			}
			if tc.streamErr != nil {
				assert.ErrorIs(t, err, tc.streamErr)
				assert.Zero(t, inspections.Load())
			} else {
				assert.EqualValues(t, 1, inspections.Load())
			}
		})
	}
}
