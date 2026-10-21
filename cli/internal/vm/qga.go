package vm

import (
	"bufio"
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"sync/atomic"
	"time"
)

// qgaChunk is how many bytes one guest-file-write carries. UTM pushes 4 KiB
// at a time; 48 KiB keeps each line well under the agent's JSON limits while
// a source archive still crosses in a few hundred calls.
const qgaChunk = 48 << 10

// QGA talks to the QEMU guest agent inside a guest: one JSON command per
// line on the agent's virtio-serial channel, one JSON reply per line. It is
// the only way the host reaches the Windows guest, which runs no sshd
// (ADR-053).
type QGA struct {
	conn    net.Conn
	r       *bufio.Reader
	Timeout time.Duration // per reply; zero waits forever
	Poll    time.Duration // between guest-exec-status calls
	Wait    time.Duration // per Run, until the process exits; zero waits forever
}

var syncID atomic.Int64

// NewQGA wraps a connection to the agent's channel.
func NewQGA(conn net.Conn) *QGA {
	return &QGA{conn: conn, r: bufio.NewReader(conn), Timeout: time.Minute, Poll: time.Second, Wait: 5 * time.Minute}
}

type qgaReply struct {
	Return json.RawMessage `json:"return"`
	Error  *struct {
		Class string `json:"class"`
		Desc  string `json:"desc"`
	} `json:"error"`
}

func (q *QGA) send(cmd string, args any) error {
	req := map[string]any{"execute": cmd}
	if args != nil {
		req["arguments"] = args
	}
	b, err := json.Marshal(req)
	if err != nil {
		return err
	}
	q.deadline()
	_, err = q.conn.Write(append(b, '\n'))
	return err
}

func (q *QGA) deadline() {
	var t time.Time
	if q.Timeout > 0 {
		t = time.Now().Add(q.Timeout)
	}
	_ = q.conn.SetDeadline(t)
}

func (q *QGA) read() (qgaReply, error) {
	q.deadline()
	line, err := q.r.ReadBytes('\n')
	if err != nil {
		return qgaReply{}, err
	}
	var rep qgaReply
	if err := json.Unmarshal(line, &rep); err != nil {
		return qgaReply{}, fmt.Errorf("the guest agent answered %q: %w", bytes.TrimSpace(line), err)
	}
	return rep, nil
}

// Sync makes the next reply on the channel the answer to the next command:
// replies a previous client left unread are read past until the one carrying
// this call's id.
func (q *QGA) Sync() error {
	id := time.Now().UnixNano() + syncID.Add(1)
	if err := q.send("guest-sync", map[string]int64{"id": id}); err != nil {
		return fmt.Errorf("guest-sync: %w", err)
	}
	for {
		rep, err := q.read()
		if err != nil {
			return fmt.Errorf("guest-sync: %w", err)
		}
		var got int64
		if json.Unmarshal(rep.Return, &got) == nil && got == id {
			return nil
		}
	}
}

// call sends one command and decodes its return value into ret.
func (q *QGA) call(cmd string, args, ret any) error {
	if err := q.send(cmd, args); err != nil {
		return fmt.Errorf("%s: %w", cmd, err)
	}
	rep, err := q.read()
	if err != nil {
		return fmt.Errorf("%s: %w", cmd, err)
	}
	if rep.Error != nil {
		return fmt.Errorf("%s: %s: %s", cmd, rep.Error.Class, rep.Error.Desc)
	}
	if ret == nil {
		return nil
	}
	if err := json.Unmarshal(rep.Return, ret); err != nil {
		return fmt.Errorf("%s: %w", cmd, err)
	}
	return nil
}

// ExecResult is a finished guest process.
type ExecResult struct {
	ExitCode       int
	Stdout, Stderr []byte
}

// Run starts path with args in the guest, its output captured, and waits for
// it to exit. A non-zero exit is a result, not an error.
func (q *QGA) Run(path string, args []string) (ExecResult, error) {
	var started struct {
		PID int `json:"pid"`
	}
	if err := q.call("guest-exec", map[string]any{"path": path, "arg": args, "capture-output": true}, &started); err != nil {
		return ExecResult{}, err
	}
	// Timeout bounds each reply, so a live agent reporting a process that
	// never exits would otherwise keep this loop polling forever.
	deadline := time.Now().Add(q.Wait)
	for {
		var st struct {
			Exited   bool   `json:"exited"`
			ExitCode int    `json:"exitcode"`
			OutData  string `json:"out-data"`
			ErrData  string `json:"err-data"`
		}
		if err := q.call("guest-exec-status", map[string]int{"pid": started.PID}, &st); err != nil {
			return ExecResult{}, err
		}
		if st.Exited {
			out, oerr := base64.StdEncoding.DecodeString(st.OutData)
			errOut, eerr := base64.StdEncoding.DecodeString(st.ErrData)
			return ExecResult{ExitCode: st.ExitCode, Stdout: out, Stderr: errOut}, errors.Join(oerr, eerr)
		}
		if q.Wait > 0 && time.Now().After(deadline) {
			return ExecResult{}, fmt.Errorf("%s (pid %d) did not exit within %s", path, started.PID, q.Wait)
		}
		time.Sleep(q.Poll)
	}
}

// WriteFile creates or truncates path in the guest with the contents of r.
func (q *QGA) WriteFile(path string, r io.Reader) (err error) {
	h, err := q.open(path, "wb")
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, q.call("guest-file-close", map[string]int{"handle": h}, nil)) }()
	buf := make([]byte, qgaChunk)
	for {
		n, rerr := io.ReadFull(r, buf)
		if n > 0 {
			args := map[string]any{"handle": h, "buf-b64": base64.StdEncoding.EncodeToString(buf[:n])}
			if err := q.call("guest-file-write", args, nil); err != nil {
				return err
			}
		}
		if errors.Is(rerr, io.EOF) || errors.Is(rerr, io.ErrUnexpectedEOF) {
			return nil
		}
		if rerr != nil {
			return rerr
		}
	}
}

// ReadFile returns the contents of path in the guest.
func (q *QGA) ReadFile(path string) (data []byte, err error) {
	h, err := q.open(path, "rb")
	if err != nil {
		return nil, err
	}
	defer func() { err = errors.Join(err, q.call("guest-file-close", map[string]int{"handle": h}, nil)) }()
	var out bytes.Buffer
	for {
		var chunk struct {
			Buf string `json:"buf-b64"`
			EOF bool   `json:"eof"`
		}
		if err := q.call("guest-file-read", map[string]int{"handle": h, "count": qgaChunk}, &chunk); err != nil {
			return nil, err
		}
		b, derr := base64.StdEncoding.DecodeString(chunk.Buf)
		if derr != nil {
			return nil, fmt.Errorf("guest-file-read: %w", derr)
		}
		out.Write(b)
		if chunk.EOF {
			return out.Bytes(), nil
		}
	}
}

func (q *QGA) open(path, mode string) (int, error) {
	var h int
	err := q.call("guest-file-open", map[string]string{"path": path, "mode": mode}, &h)
	return h, err
}

// Shutdown asks the guest to power off. The agent sends no reply, because
// the guest goes away; the caller waits for QEMU to exit instead.
func (q *QGA) Shutdown() error {
	return q.send("guest-shutdown", map[string]string{"mode": "powerdown"})
}

func (q *QGA) Close() error { return q.conn.Close() }
