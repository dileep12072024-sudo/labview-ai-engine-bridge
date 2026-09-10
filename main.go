// Package main is the LabVIEW-AI-Engine-Bridge: a local-only MCP server that
// lets an AI application build LabVIEW VIs from scratch.
//
// Chain: AI client <-MCP/stdio-> this binary <-TCP 6060-> lv_bridge.vi <-scripting-> LabVIEW
package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
)

var (
	httpPort    = flag.Int("port", 8080, "local HTTP/WebSocket port")
	labviewAddr = flag.String("labview", "127.0.0.1:6060", "LabVIEW bridge VI TCP address")
	noBrowser   = flag.Bool("no-browser", false, "do not launch the desktop browser")
	qrSVG       = flag.String("qr-svg", "", "also write the LAN QR code to this .svg file")
	stdio       = flag.Bool("stdio", false, "speak MCP over stdin/stdout (for Claude Desktop / Claude Code)")
	mock        = flag.Bool("mock", false, "run a fake LabVIEW daemon in-process, so the chain can be tested without LabVIEW")
	send        = flag.String("send", "", "send one raw command to the bridge VI, print the reply, and exit (for testing lv_bridge.vi)")
)

func main() {
	flag.Parse()
	if *mock {
		startMockDaemon(*labviewAddr)
	}
	if *send != "" {
		ack, err := sendToLabVIEW(*send + "\n")
		if err != nil {
			fmt.Println("FAILED:", err)
			os.Exit(1)
		}
		fmt.Println("reply:", ack)
		return
	}
	if *stdio {
		serveStdio()
		return
	}

	ip, err := localIP()
	if err != nil {
		log.Printf("wi-fi adapter sniff failed (%v); LAN access unavailable", err)
		ip = "127.0.0.1"
	}
	lanURL := fmt.Sprintf("http://%s:%d", ip, *httpPort)

	fmt.Printf("\nLabVIEW-AI-Engine-Bridge\n  local : http://localhost:%d\n  lan   : %s\n  daemon: %s\n\n", *httpPort, lanURL, *labviewAddr)
	printQR(lanURL)
	if *qrSVG != "" {
		if err := WriteQRSVG(lanURL, *qrSVG); err != nil {
			log.Printf("qr svg: %v", err)
		}
	}
	if !*noBrowser {
		openBrowser(fmt.Sprintf("http://localhost:%d", *httpPort))
	}
	log.Fatal(serve(*httpPort))
}

// serveStdio is the transport MCP clients actually speak: newline-delimited
// JSON-RPC on stdin/stdout. Logs go to stderr so they never corrupt the stream.
func serveStdio() {
	log.SetOutput(os.Stderr)
	in := bufio.NewScanner(os.Stdin)
	in.Buffer(make([]byte, 0, 64<<10), 4<<20)
	out := bufio.NewWriter(os.Stdout)
	defer out.Flush()
	for in.Scan() {
		line := strings.TrimSpace(in.Text())
		if line == "" {
			continue
		}
		if res := handleRPC([]byte(line)); res != nil {
			out.Write(res)
			out.WriteByte('\n')
			out.Flush()
		}
	}
}

// ---------- JSON-RPC 2.0 ----------

type rpcReq struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type rpcErr struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type rpcResp struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Result  any             `json:"result,omitempty"`
	Error   *rpcErr         `json:"error,omitempty"`
}

func reply(id json.RawMessage, res any) []byte {
	b, _ := json.Marshal(rpcResp{JSONRPC: "2.0", ID: id, Result: res})
	return b
}

func fail(id json.RawMessage, code int, msg string) []byte {
	b, _ := json.Marshal(rpcResp{JSONRPC: "2.0", ID: id, Error: &rpcErr{Code: code, Message: msg}})
	return b
}

func handleRPC(raw []byte) []byte {
	var req rpcReq
	if err := json.Unmarshal(raw, &req); err != nil {
		return fail(nil, -32700, "parse error")
	}
	switch req.Method {
	case "initialize":
		return reply(req.ID, map[string]any{
			"protocolVersion": "2024-11-05",
			"capabilities":    map[string]any{"tools": map[string]any{}},
			"serverInfo":      map[string]any{"name": "labview-ai-engine-bridge", "version": "1.0.0"},
			"instructions":    "Build a LabVIEW VI by calling create_vi first, then create_vi_control and add_block_node. Every object call returns an id; pass those ids to wire_nodes. Finish with save_vi and run_vi. Call labview_status first if unsure the LabVIEW bridge VI is running.",
		})
	case "notifications/initialized", "ping":
		if len(req.ID) == 0 {
			return nil
		}
		return reply(req.ID, map[string]any{})
	case "tools/list":
		return reply(req.ID, map[string]any{"tools": tools})
	case "tools/call":
		var p struct {
			Name string         `json:"name"`
			Args map[string]any `json:"arguments"`
		}
		if err := json.Unmarshal(req.Params, &p); err != nil {
			return fail(req.ID, -32602, "invalid params")
		}
		payload, id, err := buildPayload(p.Name, p.Args)
		if err != nil {
			return fail(req.ID, -32602, err.Error())
		}
		ack, err := sendToLabVIEW(payload)
		if err != nil {
			return fail(req.ID, -32000, "LabVIEW bridge VI unreachable on "+*labviewAddr+" ("+err.Error()+"). Open lv_bridge.vi in LabVIEW and press Run.")
		}
		text := ack
		if id != "" {
			text = ack + "  [id: " + id + "]"
		}
		return reply(req.ID, map[string]any{
			"content": []map[string]any{{"type": "text", "text": text}},
			"id":      id,
			"sent":    strings.TrimSpace(payload),
		})
	default:
		return fail(req.ID, -32601, "unknown method: "+req.Method)
	}
}

// ---------- Tool registry ----------

func obj(props map[string]any, required ...string) map[string]any {
	s := map[string]any{"type": "object", "properties": props}
	if len(required) > 0 {
		s["required"] = required
	}
	return s
}

var (
	str = map[string]any{"type": "string"}
	num = map[string]any{"type": "number"}
)

var tools = []map[string]any{
	{
		"name":        "labview_status",
		"description": "Check that the LabVIEW bridge VI is running and reachable. Call this first if a build fails.",
		"inputSchema": obj(map[string]any{}),
	},
	{
		"name":        "create_vi",
		"description": "Create a new blank VI and make it the active target. Resets the auto-layout cursor and the object id table.",
		"inputSchema": obj(map[string]any{"name": str}, "name"),
	},
	{
		"name":        "create_vi_control",
		"description": "Place a front-panel control or indicator on the active VI. Returns the id used for wiring.",
		"inputSchema": obj(map[string]any{
			"control_type": map[string]any{"type": "string", "enum": controlList()},
			"label":        str,
			"channel":      map[string]any{"type": "integer", "minimum": 0, "maximum": 255},
			"indicator":    map[string]any{"type": "boolean", "description": "true makes it an indicator instead of a control"},
			"x":            num, "y": num,
		}, "control_type"),
	},
	{
		"name":        "add_block_node",
		"description": "Drop a function or structure on the block diagram of the active VI. Returns the id used for wiring.",
		"inputSchema": obj(map[string]any{
			"function": map[string]any{"type": "string", "description": "e.g. add, subtract, multiply, divide, while_loop, for_loop, case, wait, random"},
			"id":       str,
			"x":        num, "y": num,
		}, "function"),
	},
	{
		"name":        "wire_nodes",
		"description": "Wire one object's terminal to another's on the block diagram, using ids returned by earlier calls.",
		"inputSchema": obj(map[string]any{
			"source": str, "destination": str,
			"source_terminal": map[string]any{"type": "integer", "minimum": 0},
			"dest_terminal":   map[string]any{"type": "integer", "minimum": 0},
		}, "source", "destination"),
	},
	{
		"name":        "set_control_value",
		"description": "Set the value of a front-panel control by id.",
		"inputSchema": obj(map[string]any{"id": str, "value": str}, "id", "value"),
	},
	{
		"name":        "list_objects",
		"description": "List the objects the bridge has placed on the active VI.",
		"inputSchema": obj(map[string]any{}),
	},
	{
		"name":        "save_vi",
		"description": "Save the active VI to an absolute path on the host PC, e.g. C:\\\\vis\\\\demo.vi",
		"inputSchema": obj(map[string]any{"path": str}, "path"),
	},
	{
		"name":        "run_vi",
		"description": "Run the active VI.",
		"inputSchema": obj(map[string]any{}),
	},
}

// ---------- LabVIEW wire protocol ----------

// Front-panel control styles the daemon knows how to instantiate.
var controls = []string{"knob", "dial", "button", "switch", "led", "graph", "chart",
	"numeric", "string", "slider", "gauge", "tank", "boolean", "path", "table", "ring"}

func controlList() []string { return controls }

func isControl(s string) bool {
	for _, c := range controls {
		if c == s {
			return true
		}
	}
	return false
}

// clean strips the field and record separators so a tool argument can never
// forge extra fields or extra commands on the daemon socket. Colons survive:
// the daemon splits each field on its FIRST colon only, so "C:\path" is safe.
func clean(v any) string {
	if v == nil {
		return ""
	}
	s := fmt.Sprintf("%v", v)
	s = strings.NewReplacer(";", "", "\n", "", "\r", "").Replace(s)
	if len(s) > 260 {
		s = s[:260]
	}
	return strings.TrimSpace(s)
}

// layout hands out object ids and grid positions so the AI never has to think
// about pixel coordinates and LabVIEW never has to compute them.
type layout struct {
	mu  sync.Mutex
	seq map[string]int
	fp,
	bd int
}

var lay = layout{seq: map[string]int{}}

func (l *layout) reset() {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.seq = map[string]int{}
	l.fp, l.bd = 0, 0
}

// next returns a unique id like "knob1" and the next free slot on a 3-wide grid.
func (l *layout) next(style string, panel bool) (id string, x, y int) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.seq[style]++
	id = fmt.Sprintf("%s%d", style, l.seq[style])
	n := &l.bd
	ox, oy, dx, dy := 60, 60, 200, 120
	if panel {
		n, ox, oy, dx, dy = &l.fp, 20, 20, 170, 150
	}
	x, y = ox+(*n%3)*dx, oy+(*n/3)*dy
	*n++
	return
}

func coord(a map[string]any, key string, def int) int {
	if f, ok := a[key].(float64); ok {
		return int(f)
	}
	return def
}

// buildPayload translates an MCP tool call into one daemon command line.
// It returns the object id when the call creates something wireable.
func buildPayload(tool string, a map[string]any) (payload, id string, err error) {
	switch tool {
	case "labview_status":
		return "CMD:PING\n", "", nil

	case "create_vi":
		n := clean(a["name"])
		if n == "" {
			return "", "", fmt.Errorf("name is required")
		}
		lay.reset()
		return "CMD:NEWVI;NAME:" + n + "\n", "", nil

	case "create_vi_control":
		st := strings.ToLower(clean(a["control_type"]))
		if !isControl(st) {
			return "", "", fmt.Errorf("unsupported control_type %q; use one of %s", st, strings.Join(controls, ", "))
		}
		id, dx, dy := lay.next(st, true)
		line := fmt.Sprintf("CMD:NEW;CLS:control;STY:%s;ID:%s;X:%d;Y:%d", st, id, coord(a, "x", dx), coord(a, "y", dy))
		if b, _ := a["indicator"].(bool); b {
			line += ";IND:1"
		}
		if lbl := clean(a["label"]); lbl != "" {
			line += ";LBL:" + lbl
		}
		if f, ok := a["channel"].(float64); ok {
			if f < 0 || f > 255 {
				return "", "", fmt.Errorf("channel out of range 0-255")
			}
			line += fmt.Sprintf(";CH:%d", int(f))
		}
		return line + "\n", id, nil

	case "add_block_node":
		fn := strings.ToLower(clean(a["function"]))
		if fn == "" {
			return "", "", fmt.Errorf("function is required")
		}
		auto, dx, dy := lay.next(fn, false)
		if given := clean(a["id"]); given != "" {
			auto = given
		}
		return fmt.Sprintf("CMD:NEW;CLS:function;STY:%s;ID:%s;X:%d;Y:%d\n",
			fn, auto, coord(a, "x", dx), coord(a, "y", dy)), auto, nil

	case "wire_nodes":
		src, dst := clean(a["source"]), clean(a["destination"])
		if src == "" || dst == "" {
			return "", "", fmt.Errorf("source and destination ids are required")
		}
		return fmt.Sprintf("CMD:WIRE;SRC:%s;ST:%d;DST:%s;DT:%d\n",
			src, coord(a, "source_terminal", 0), dst, coord(a, "dest_terminal", 0)), "", nil

	case "set_control_value":
		i, v := clean(a["id"]), clean(a["value"])
		if i == "" {
			return "", "", fmt.Errorf("id is required")
		}
		return "CMD:SET;ID:" + i + ";VAL:" + v + "\n", "", nil

	case "list_objects":
		return "CMD:LIST\n", "", nil

	case "save_vi":
		p := clean(a["path"])
		if p == "" {
			return "", "", fmt.Errorf("path is required")
		}
		return "CMD:SAVE;PATH:" + p + "\n", "", nil

	case "run_vi":
		return "CMD:RUN\n", "", nil
	}
	return "", "", fmt.Errorf("unknown tool %q", tool)
}
