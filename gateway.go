package main

import (
	"fmt"
	"log"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
)

var sessions int64

// Local-network only: any origin on the LAN is allowed, but the server never
// binds anything routable off it.
var upgrader = websocket.Upgrader{CheckOrigin: func(*http.Request) bool { return true }}

func serve(port int) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/", dashboard)
	mux.HandleFunc("/ws", wsPipe)
	srv := &http.Server{
		Addr:              fmt.Sprintf(":%d", port),
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}
	log.Printf("gateway listening on %s", srv.Addr)
	return srv.ListenAndServe()
}

func dashboard(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	// Ephemeral session: nothing may be cached, stored or restored by the client.
	w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate, max-age=0")
	w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'unsafe-inline'; style-src 'unsafe-inline'; connect-src ws: wss:")
	w.Header().Set("Referrer-Policy", "no-referrer")
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprint(w, dashboardHTML)
}

// wsPipe holds one ephemeral session. All state is this goroutine's stack:
// the tab closes, the socket dies, the session is gone.
func wsPipe(w http.ResponseWriter, r *http.Request) {
	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer c.Close()
	id := atomic.AddInt64(&sessions, 1)
	log.Printf("session %d open (%s)", id, r.RemoteAddr)
	defer func() { log.Printf("session %d destroyed", id) }()

	c.SetReadLimit(64 << 10)
	for {
		_, msg, err := c.ReadMessage()
		if err != nil {
			return
		}
		if out := handleRPC(msg); out != nil {
			if err := c.WriteMessage(websocket.TextMessage, out); err != nil {
				return
			}
		}
	}
}

const dashboardHTML = `<!doctype html>
<html lang="en"><head>
<meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1">
<meta name="referrer" content="no-referrer">
<title>LabVIEW AI Engine Bridge</title>
<style>
:root{--bg:#0b0f14;--panel:#131a22;--line:#223040;--fg:#dbe5ee;--dim:#7c8b9a;--acc:#38bdf8;--ok:#4ade80;--err:#f87171}
*{box-sizing:border-box}
body{margin:0;font:14px/1.5 ui-monospace,SFMono-Regular,Menlo,monospace;background:var(--bg);color:var(--fg)}
header{padding:14px 18px;border-bottom:1px solid var(--line);display:flex;gap:12px;align-items:center;flex-wrap:wrap}
h1{font-size:15px;margin:0;letter-spacing:.08em;text-transform:uppercase;color:var(--acc)}
#dot{width:9px;height:9px;border-radius:50%;background:var(--err)}
#dot.on{background:var(--ok)}
main{display:grid;grid-template-columns:320px 1fr;gap:16px;padding:16px}
@media(max-width:760px){main{grid-template-columns:1fr}}
section{background:var(--panel);border:1px solid var(--line);border-radius:8px;padding:14px}
h2{font-size:11px;letter-spacing:.14em;text-transform:uppercase;color:var(--dim);margin:0 0 10px}
label{display:block;font-size:11px;color:var(--dim);margin:10px 0 4px}
input,select,button{width:100%;padding:9px 10px;border-radius:6px;border:1px solid var(--line);background:#0d141b;color:var(--fg);font:inherit}
button{background:var(--acc);color:#04121c;border:0;font-weight:700;cursor:pointer;margin-top:12px}
button:active{transform:translateY(1px)}
#log{height:62vh;overflow:auto;margin:0;white-space:pre-wrap;word-break:break-word;font-size:12.5px}
.tx{color:var(--acc)}.rx{color:var(--ok)}.er{color:var(--err)}.t{color:var(--dim)}
footer{padding:0 18px 18px;color:var(--dim);font-size:11px}
</style></head><body>
<header><span id="dot"></span><h1>LabVIEW AI Engine Bridge</h1><span class="t" id="stat">connecting…</span></header>
<main>
 <section>
  <h2>Active VI</h2>
  <label>vi name</label><input id="vi" value="demo">
  <button onclick="call('tools/call',{name:'create_vi',arguments:{name:v('vi')}})">create_vi</button>
  <button onclick="demo()" style="background:#1e3a4c;color:var(--acc)">build demo VI</button>
  <h2 style="margin-top:22px">Spawn control</h2>
  <label>control_type</label>
  <select id="ct"><option>knob</option><option>button</option><option>graph</option><option>slider</option><option>led</option><option>numeric</option><option>string</option></select>
  <label>channel</label><input id="ch" type="number" value="1" min="0" max="255">
  <label>label (optional)</label><input id="lbl" placeholder="Setpoint">
  <button onclick="spawn()">create_vi_control</button>
  <h2 style="margin-top:22px">Wire nodes</h2>
  <label>source id</label><input id="src" placeholder="knob1">
  <label>destination id</label><input id="dst" placeholder="add1">
  <button onclick="wire()">wire_nodes</button>
  <h2 style="margin-top:22px">Session</h2>
  <label>save path</label><input id="path" value="C:\\vis\\demo.vi">
  <button onclick="call('tools/call',{name:'save_vi',arguments:{path:v('path')}})">save_vi</button>
  <button onclick="call('tools/call',{name:'run_vi',arguments:{}})">run_vi</button>
  <button onclick="call('tools/call',{name:'labview_status',arguments:{}})" style="background:#1e3a4c;color:var(--acc)">labview_status</button>
 </section>
 <section><h2>Stream</h2><pre id="log"></pre></section>
</main>
<footer>Ephemeral session — state lives in memory only. Closing this tab destroys it.</footer>
<script>
// No localStorage, no sessionStorage, no cookies, no history writes. On purpose.
let id=0, ws;
const log=(cls,t)=>{const p=document.getElementById('log');
  p.insertAdjacentHTML('beforeend','<span class="t">'+new Date().toLocaleTimeString()+'  </span><span class="'+cls+'">'+t.replace(/[<&]/g,c=>({'<':'&lt;','&':'&amp;'}[c]))+'</span>\n');
  p.scrollTop=p.scrollHeight;};
function connect(){
  ws=new WebSocket((location.protocol==='https:'?'wss://':'ws://')+location.host+'/ws');
  ws.onopen=()=>{document.getElementById('dot').classList.add('on');document.getElementById('stat').textContent='live · '+location.host;
    call('initialize',{});call('tools/list',{});};
  ws.onclose=()=>{document.getElementById('dot').classList.remove('on');document.getElementById('stat').textContent='disconnected';setTimeout(connect,1500);};
  ws.onmessage=e=>{const m=JSON.parse(e.data);log(m.error?'er':'rx','< '+(m.error?m.error.message:JSON.stringify(m.result)));};
}
function call(method,params){const f={jsonrpc:'2.0',id:++id,method,params};ws.send(JSON.stringify(f));log('tx','> '+method+' '+JSON.stringify(params));}
const v=i=>document.getElementById(i).value.trim();
function spawn(){call('tools/call',{name:'create_vi_control',arguments:{control_type:v('ct'),channel:Number(v('ch'))||0,label:v('lbl')}});}
function wire(){call('tools/call',{name:'wire_nodes',arguments:{source:v('src'),destination:v('dst')}});}
function demo(){
  const t=(n,a)=>call('tools/call',{name:n,arguments:a});
  t('create_vi',{name:v('vi')});
  t('create_vi_control',{control_type:'knob',label:'A'});
  t('create_vi_control',{control_type:'knob',label:'B'});
  t('add_block_node',{function:'add',id:'sum'});
  t('create_vi_control',{control_type:'numeric',label:'Result',indicator:true});
  t('wire_nodes',{source:'knob1',destination:'sum',dest_terminal:0});
  t('wire_nodes',{source:'knob2',destination:'sum',dest_terminal:1});
  t('wire_nodes',{source:'sum',destination:'numeric1'});
}
addEventListener('pagehide',()=>ws&&ws.close());
connect();
</script></body></html>`
