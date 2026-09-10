# Building `lv_bridge.vi`, step by step

This is the LabVIEW-side addon. Blender MCP ships its addon as a `.py` text file.
A `.vi` is a binary graphical file, so this one is assembled by hand, once.

Budget 30 to 45 minutes. Build it in nine phases and test after each one, so you
never debug more than a few nodes at a time.

**Keep a terminal open next to LabVIEW.** After every phase you will run one
command against the VI while it is running. That command is your checkpoint.

```
cd C:\LabVIEW-AI-Engine-Bridge
bridge.exe -send "CMD:PING"
```

---

## Phase 0. Prepare LabVIEW

1. `Tools ▸ Options ▸ VI Server`. Tick **Show VI Scripting operations**. On some
   versions this sits under `Tools ▸ Options ▸ Environment` instead. If you
   cannot find it, LabVIEW 2010 and newer usually have scripting on already.
2. Restart LabVIEW.
3. `File ▸ New VI`. Save it as `C:\LabVIEW-AI-Engine-Bridge\lv_bridge.vi`.

**Check:** right-click the block diagram, open `Programming ▸ Application
Control`, drop an **Invoke Node**, wire nothing, and right-click it to pick a
class. If **Application** offers a method called **New VI**, scripting is on.
Delete the node and continue.

---

## Phase 1. An echo server

Everything else hangs off this. Get it right before adding logic.

On the block diagram:

1. `Data Communication ▸ Protocols ▸ TCP`. Drop **TCP Listen.vi**.
   Wire a constant `6060` to its **port** input. Leave **timeout** unwired.
2. Drop a **While Loop** to the right of it.
3. Wire the **connection ID** from TCP Listen into the loop. Right-click where
   the wire crosses the loop border and choose **Replace with Shift Register**.
4. Inside the loop, drop **TCP Read**.
   - Right-click its **mode** input, `Create ▸ Constant`, set it to **CRLF**.
   - Wire `4096` to **bytes to read**.
   - Wire `-1` to **timeout ms** so it blocks until a command arrives.
5. Drop **TCP Write** to its right. Wire the connection ID through.
6. Between them, drop a **Concatenate Strings** and join the read string with a
   **String Constant** containing a carriage return and a line feed. Switch that
   constant to `\` codes display by right-clicking it, then type `\r\n`.
7. Wire the loop's stop terminal to a **False** constant for now.
8. Wire the error clusters straight through every node, left to right.

**Checkpoint.** Press Run. In the terminal:

```
bridge.exe -send "CMD:PING"
```

You should see `reply: CMD:PING`. The VI echoed you. Press the Abort button to
stop it. If this hangs, the CRLF mode constant is wrong or the port is taken.

---

## Phase 2. A field parser

Every command is `KEY:VALUE;KEY:VALUE`. Values may contain colons, because
`PATH:C:\vis\demo.vi` has to survive. So split on `;` first, then split each
field on its **first** colon only.

Make a subVI so you only build this once.

1. `File ▸ New VI`. Save as `GetField.vi` beside the bridge.
2. Front panel: two string controls, `line` and `key`, one string indicator,
   `value`.
3. Block diagram:
   - **Spreadsheet String To Array** (`Programming ▸ String`). Wire `line` to
     its string input. Set **delimiter** to a `;` constant. Right-click its
     array type input and wire an empty **String Array Constant** so it returns
     a 1D array of strings.
   - Drop a **For Loop** over that array with auto-indexing.
   - Inside, **Search/Split String** (`Programming ▸ String`). Wire the element
     in, and a `:` constant to **search string**. It returns the substring
     before the match and the substring after. That is your key and value, split
     on the first colon exactly as required.
   - Trim the leading colon off the second output with **String Subset**, offset
     `1`.
   - Auto-index both out of the loop as two arrays: keys and values.
   - **Search 1D Array** (`Programming ▸ Array`) on the keys array with `key`.
   - **Index Array** on the values array with that index.
   - Wire to `value`. If the index is `-1`, output an empty string. Use
     **Select** with a **Greater Or Equal To 0** comparison.
4. Assign the connector pane: `line` and `key` as inputs, `value` as output.
5. Save.

**Checkpoint.** Run `GetField.vi` by hand. Type
`CMD:SAVE;PATH:C:\vis\demo.vi` into `line` and `PATH` into `key`. You must get
`C:\vis\demo.vi` back, colon and all. If you get `C`, you split on every colon
instead of the first one.

---

## Phase 3. State

Back in `lv_bridge.vi`. Add three more shift registers to the While Loop:

| Register | Type | Holds |
|---|---|---|
| VI refnum | VI refnum | the active VI being built |
| id table | Variant | name to refnum map |
| connection ID | already added in Phase 1 | the socket |

Initialise the id table with an **Empty Variant** constant outside the loop.
Leave the VI refnum register unwired on the left for now.

The id table is the piece people forget. It is what lets `wire_nodes` refer to
`knob1` three calls after `knob1` was created. `Set Variant Attribute` writes a
refnum under a name, `Get Variant Attribute` reads it back. Both live in
`Programming ▸ Cluster, Class, & Variant ▸ Variant`.

---

## Phase 4. Dispatch and the first real command

1. Between TCP Read and TCP Write, drop a **Case Structure**.
2. Drop `GetField.vi`, wire the read string to `line` and a `CMD` constant to
   `key`. Wire its output to the case selector. The case structure now switches
   on the command name.
3. Rename the default case to `PING`, and inside it wire a string constant `OK`
   to the output tunnel. Add a second case `NEWVI`.

Inside `NEWVI`:

4. `Programming ▸ Application Control`. Drop **Open Application Reference**.
   Leave **machine name** empty, which means this LabVIEW.
5. Drop an **Invoke Node**, wire the application reference to it. Right-click,
   `Select Method ▸ New VI`. Wire a **VI Type** constant of `Standard VI`.
6. Its **VI Refnum** output goes to the VI refnum shift register.
7. Drop a **Property Node** on that refnum, select
   **Front Panel Window ▸ Open**, right-click it and choose **Change To Write**,
   wire a **True** constant. Now you can watch the VI being assembled.
8. Reset the id table: wire an **Empty Variant** constant to the id shift
   register inside this case.
9. Output `OK new VI` to the tunnel.

**Checkpoint.** Run the VI, then:

```
bridge.exe -send "CMD:PING"
bridge.exe -send "CMD:NEWVI;NAME:test"
```

The first prints `reply: OK`. The second prints `reply: OK new VI` and a blank
VI window opens in LabVIEW. That window opening is the moment the bridge becomes
real.

---

## Phase 5. Placing controls

This is the case that does most of the work. One command, `CMD:NEW`, creates
every kind of object. Add a `NEW` case.

Read six fields with `GetField.vi`: `CLS`, `STY`, `ID`, `X`, `Y`, `LBL`.
Convert `X` and `Y` with **Decimal String To Number**.

**The style trick.** LabVIEW has no primitive that turns the string `"knob"`
into a style enum. Building a Case structure per control type would mean forty
cases. Instead:

1. On the bridge VI's **front panel**, drop the style enum. The quickest way to
   get one of the correct type: drop a **New VI Object** invoke node on the
   diagram, right-click its **Style** input, `Create ▸ Control`. Name it
   `StyleEnum` and set it **Hidden** on the front panel.
2. On the diagram, drop a **Property Node** on `StyleEnum` and select
   **Strings[]**. It returns every valid style name as an array.
3. **Search 1D Array** that array for your incoming `STY` string. Uppercase both
   sides first with **To Upper Case** so `knob` matches `Knob`.
4. **Type Cast** (`Programming ▸ Numeric ▸ Data Manipulation`) the resulting
   index into `StyleEnum`. You now have the enum the invoke node wants.

Five nodes, and it supports every style LabVIEW has without you naming a single
one. If `Search 1D Array` returns `-1`, output `ERR unknown style` and skip.

Then:

5. **Property Node** on the VI refnum, select **Front Panel**. That returns a
   panel refnum.
6. **Invoke Node** on the panel refnum, `Select Method ▸ New VI Object`.
   - **Object Class**: `Control`
   - **Style**: your type-cast enum
   - **Position**: bundle `X` and `Y` with **Bundle** into the position cluster
   - **Data Type**: wire a **DBL constant** for numeric styles. Check the node's
     terminals with Context Help, `Ctrl+H`, because they differ slightly by
     version.
7. On the returned object refnum, drop a **Property Node** and select
   **Label ▸ Text**, change to write, wire `LBL`.
8. If `GetField.vi` returns `1` for the `IND` key, drop another Property Node
   and set **Control ▸ Indicator** to True.
9. **Set Variant Attribute**: name is `ID`, value is the object refnum, variant
   in and out go through the id shift register.
10. Output `OK placed` concatenated with `ID`.

**Checkpoint.**

```
bridge.exe -send "CMD:NEWVI;NAME:test"
bridge.exe -send "CMD:NEW;CLS:control;STY:knob;ID:knob1;X:20;Y:20;LBL:Setpoint"
```

A knob labelled Setpoint appears on the new VI's front panel. Stop here and take
a break if you like. From this point the AI can already build front panels.

---

## Phase 6. Placing block diagram functions

Almost free, because the hard part is done. Still inside the `NEW` case:

1. Add a small Case structure on the `CLS` field, `control` versus `function`.
2. In the `function` branch, use the VI's **Block Diagram** property instead of
   **Front Panel**, and set **Object Class** to `Function` on the New VI Object
   node.
3. You need a second hidden enum for the function style, created the same way
   from the Function variant of the node. Same `Strings[]`, `Search 1D Array`,
   `Type Cast` trick.
4. Store the refnum under `ID` exactly as before.

**Checkpoint.**

```
bridge.exe -send "CMD:NEW;CLS:function;STY:add;ID:sum;X:60;Y:60"
```

An Add function appears on the block diagram.

---

## Phase 7. Save and run

Two tiny cases.

`SAVE`: **Invoke Node** on the VI refnum, `Select Method ▸ Save Instrument`.
Wire the `PATH` field to its **VI Path** input through a **String To Path**
conversion. Output `OK saved`.

`RUN`: **Invoke Node** on the VI refnum, `Select Method ▸ Run VI`. Wire
**Wait Until Done** to False so the bridge does not block. Output `OK running`.

**Checkpoint.**

```
bridge.exe -send "CMD:SAVE;PATH:C:\vis\test.vi"
```

The file exists on disk. That is the proof that the AI can produce real,
openable LabVIEW files.

---

## Phase 8. Wiring

The hardest case. Do it last, and only once everything above works.

Add a `WIRE` case reading `SRC`, `ST`, `DST`, `DT`.

1. **Get Variant Attribute** twice, on `SRC` and `DST`, against the id table.
   Both give you object refnums.
2. Turn each refnum into a terminal:
   - A **front-panel control** has a single diagram terminal. Property Node,
     select **Terminal**.
   - A **function** has several. Property Node, select **Terminals[]**, then
     **Index Array** with `ST` or `DT`.
   - You will not know which kind you have. Use **To More Specific Class** with
     an error case, or store a `control`/`function` marker alongside each id
     when you create it. Storing the marker is simpler: keep a second variant
     attribute named `ID` plus `.cls`.
3. **Property Node** on the VI refnum, select **Block Diagram**.
4. **Invoke Node** on the diagram refnum, `Select Method ▸ Connect Wire`. Wire
   the two terminal refnums into **Terminal 1** and **Terminal 2**.
5. Output `OK wired`.

**Checkpoint.**

```
bridge.exe -send "CMD:WIRE;SRC:knob1;ST:0;DST:sum;DT:0"
```

A wire appears. Terminal indices are not guessable: for `Add`, terminal 0 is x,
1 is y, 2 is the sum. Probe `Terminals[]` once with a string indicator and write
the order down.

---

## Phase 9. Errors, and the rest

1. `SET`: Invoke Node on the VI refnum, **Set Control Value**, control name from
   the label you stored, value from `VAL` through **To Variant**.
2. `LIST`: **Get Variant Attribute** with the name input unwired returns every
   attribute name. Join with **Array To Spreadsheet String**, delimiter `, `.
3. **Error reporting.** This one matters more than it looks. Take the error
   cluster leaving the case structure, run it through **Select**: if there is an
   error, output `ERR ` concatenated with the error's **source**, otherwise
   output the normal ack. The Go side turns any reply starting with `ERR` into a
   failed tool call with your message attached, so the AI sees the real LabVIEW
   error and can correct itself. Without this the AI is building blind.
4. Wire **Clear Errors** after that, so one bad command does not kill the loop.

**Final check.** Leave `lv_bridge.vi` running and ask your AI to build the two
knob adder. Watch it assemble.

---

## Things that will bite you

- **Scripting is development-environment only.** It does not exist in the
  Run-Time Engine. LabVIEW must be open, and this can never be built into an
  executable.
- **`New VI Object` needs an explicit position** or everything stacks at the
  origin. The Go side always sends `X` and `Y`, so just wire them.
- **Terminal names vary slightly between LabVIEW versions.** Use Context Help,
  `Ctrl+H`, over any invoke node whose terminals do not match this document.
- **Stopping and rerunning the VI drops the socket.** The Go side reconnects
  automatically on the next command, so this is safe, but the first command
  after a restart may report one failure.
- **Do not close the VI window LabVIEW created.** It is the active target. Use
  `CMD:NEWVI` to start a fresh one.
