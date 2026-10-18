package vm

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"time"
)

// QMP controls the QEMU process itself, as QGA reaches inside the guest: a
// greeting, a capabilities handshake, then one JSON command per line, with
// asynchronous events interleaved among the replies.
type QMP struct {
	conn    net.Conn
	r       *bufio.Reader
	Timeout time.Duration // per reply; zero waits forever
}

type qmpReply struct {
	QMP    json.RawMessage `json:"QMP"`
	Event  string          `json:"event"`
	Return json.RawMessage `json:"return"`
	Error  *struct {
		Class string `json:"class"`
		Desc  string `json:"desc"`
	} `json:"error"`
}

// NewQMP reads the greeting on conn and negotiates capabilities.
func NewQMP(conn net.Conn) (*QMP, error) {
	q := &QMP{conn: conn, r: bufio.NewReader(conn), Timeout: 30 * time.Second}
	greet, err := q.read()
	if err != nil {
		return nil, fmt.Errorf("QMP greeting: %w", err)
	}
	if greet.QMP == nil {
		return nil, fmt.Errorf("QMP greeting: no QMP banner")
	}
	if err := q.Execute("qmp_capabilities", nil, nil); err != nil {
		return nil, err
	}
	return q, nil
}

func (q *QMP) read() (qmpReply, error) {
	if q.Timeout > 0 {
		_ = q.conn.SetDeadline(time.Now().Add(q.Timeout))
	}
	line, err := q.r.ReadBytes('\n')
	if err != nil {
		return qmpReply{}, err
	}
	var rep qmpReply
	if err := json.Unmarshal(line, &rep); err != nil {
		return qmpReply{}, fmt.Errorf("QMP answered %q: %w", line, err)
	}
	return rep, nil
}

// Execute runs cmd and decodes its return into ret, when ret is not nil.
// Events that arrive before the reply are skipped.
func (q *QMP) Execute(cmd string, args, ret any) error {
	req := map[string]any{"execute": cmd}
	if args != nil {
		req["arguments"] = args
	}
	b, err := json.Marshal(req)
	if err != nil {
		return err
	}
	if q.Timeout > 0 {
		_ = q.conn.SetDeadline(time.Now().Add(q.Timeout))
	}
	if _, err := q.conn.Write(append(b, '\n')); err != nil {
		return fmt.Errorf("%s: %w", cmd, err)
	}
	for {
		rep, err := q.read()
		if err != nil {
			return fmt.Errorf("%s: %w", cmd, err)
		}
		switch {
		case rep.Event != "":
			continue
		case rep.Error != nil:
			return fmt.Errorf("%s: %s: %s", cmd, rep.Error.Class, rep.Error.Desc)
		case ret != nil:
			return json.Unmarshal(rep.Return, ret)
		default:
			return nil
		}
	}
}

// PressKey presses and releases one key, named by its QEMU qcode ("ret").
func (q *QMP) PressKey(qcode string) error {
	return q.Execute("send-key", map[string]any{"keys": []map[string]string{{"type": "qcode", "data": qcode}}}, nil)
}

// Screenshot writes the guest's screen to path as a PNG: the only view of a
// headless install that stopped.
func (q *QMP) Screenshot(path string) error {
	return q.Execute("screendump", map[string]string{"filename": path, "format": "png"}, nil)
}

// Status is the run state QEMU reports ("running", "shutdown", ...).
func (q *QMP) Status() (string, error) {
	var s struct {
		Status string `json:"status"`
	}
	err := q.Execute("query-status", nil, &s)
	return s.Status, err
}

// Quit stops QEMU at once, as pulling the plug would.
func (q *QMP) Quit() error { return q.Execute("quit", nil, nil) }

func (q *QMP) Close() error { return q.conn.Close() }
