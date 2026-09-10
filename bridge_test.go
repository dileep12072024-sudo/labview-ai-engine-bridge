package main

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

func mustBuild(t *testing.T, tool string, a map[string]any) (string, string) {
	t.Helper()
	p, id, err := buildPayload(tool, a)
	if err != nil {
		t.Fatalf("%s: %v", tool, err)
	}
	return p, id
}

func TestBuildPayload(t *testing.T) {
	lay.reset()

	if p, _ := mustBuild(t, "create_vi", map[string]any{"name": "demo"}); p != "CMD:NEWVI;NAME:demo\n" {
		t.Fatalf("newvi = %q", p)
	}
	p, id := mustBuild(t, "create_vi_control", map[string]any{"control_type": "knob", "label": "Setpoint", "channel": float64(1)})
	if id != "knob1" || p != "CMD:NEW;CLS:control;STY:knob;ID:knob1;X:20;Y:20;LBL:Setpoint;CH:1\n" {
		t.Fatalf("control = %q id=%q", p, id)
	}
	// second control of the same style gets a fresh id and a fresh grid slot
	if p, id = mustBuild(t, "create_vi_control", map[string]any{"control_type": "knob"}); id != "knob2" || !strings.Contains(p, "X:190;Y:20") {
		t.Fatalf("layout did not advance: %q id=%q", p, id)
	}
	if _, id = mustBuild(t, "add_block_node", map[string]any{"function": "add"}); id != "add1" {
		t.Fatalf("node id = %q", id)
	}
	if p, _ = mustBuild(t, "wire_nodes", map[string]any{"source": "knob1", "destination": "add1", "dest_terminal": float64(1)}); p != "CMD:WIRE;SRC:knob1;ST:0;DST:add1;DT:1\n" {
		t.Fatalf("wire = %q", p)
	}
	// windows paths keep their colon
	if p, _ = mustBuild(t, "save_vi", map[string]any{"path": `C:\vis\demo.vi`}); p != "CMD:SAVE;PATH:C:\\vis\\demo.vi\n" {
		t.Fatalf("save = %q", p)
	}
	// a hostile argument must not forge a second field or a second command
	p, _ = mustBuild(t, "set_control_value", map[string]any{"id": "knob1", "value": "9;CMD:RUN\nCMD:RUN"})
	if strings.Count(p, ";") != 2 || strings.Count(p, "\n") != 1 {
		t.Fatalf("injection leaked: %q", p)
	}
	if _, _, err := buildPayload("create_vi_control", map[string]any{"control_type": "rootkit"}); err == nil {
		t.Fatal("unknown control_type accepted")
	}
	if _, _, err := buildPayload("create_vi_control", map[string]any{"control_type": "knob", "channel": float64(999)}); err == nil {
		t.Fatal("out-of-range channel accepted")
	}
}

// fields is what the LabVIEW bridge VI must reimplement; pin its edge case.
func TestFieldsKeepsColonsInValues(t *testing.T) {
	f := fields(`CMD:SAVE;PATH:C:\vis\demo.vi`)
	if f["CMD"] != "SAVE" || f["PATH"] != `C:\vis\demo.vi` {
		t.Fatalf("parse = %#v", f)
	}
}

func TestHandleRPC(t *testing.T) {
	var r rpcResp
	json.Unmarshal(handleRPC([]byte(`{"jsonrpc":"2.0","id":1,"method":"tools/list"}`)), &r)
	if r.Error != nil {
		t.Fatalf("tools/list: %v", r.Error)
	}
	json.Unmarshal(handleRPC([]byte(`{"jsonrpc":"2.0","id":2,"method":"nope"}`)), &r)
	if r.Error == nil || r.Error.Code != -32601 {
		t.Fatal("unknown method should be -32601")
	}
	if handleRPC([]byte(`{"jsonrpc":"2.0","method":"notifications/initialized"}`)) != nil {
		t.Fatal("notification must not get a response")
	}
}

// End to end over the transport the phone uses: dashboard, WebSocket, mock daemon.
func TestGatewayRoundTrip(t *testing.T) {
	addr := "127.0.0.1:6099"
	*labviewAddr = addr
	startMockDaemon(addr)
	go serve(8099)
	time.Sleep(200 * time.Millisecond)

	res, err := http.Get("http://127.0.0.1:8099/")
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if cc := res.Header.Get("Cache-Control"); !strings.Contains(cc, "no-store") {
		t.Fatalf("ephemeral session needs no-store, got %q", cc)
	}

	c, _, err := websocket.DefaultDialer.Dial("ws://127.0.0.1:8099/ws", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	c.WriteMessage(websocket.TextMessage, []byte(`{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"create_vi","arguments":{"name":"ws"}}}`))
	_, msg, err := c.ReadMessage()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(msg), "OK new VI ws") {
		t.Fatalf("round trip = %s", msg)
	}
}
