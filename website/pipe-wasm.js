let pipeReady = false;
const pipeGo = new Go();

WebAssembly.instantiateStreaming(fetch("pipe.wasm?v=2"), pipeGo.importObject).then(r => {
  pipeGo.run(r.instance);
  pipeReady = true;
  const bar = document.getElementById("play-bar");
  if (bar) bar.innerHTML = '<span style="color:var(--green)">✓ ready</span>';
  const btn = document.getElementById("play-btn");
  if (btn) btn.disabled = false;
  const gb = document.getElementById("gen-btn");
  if (gb) gb.disabled = false;
}).catch(e => {
  const bar = document.getElementById("play-bar");
  if (bar) bar.innerHTML = '<span style="color:var(--red)">✗ failed: ' + e.message + '</span>';
  console.error("WASM load error:", e);
});

function runPipe(code) {
  if (!pipeReady) return "WASM not loaded";
  try { return pipeRun(code) || "(no output)"; }
  catch(e) { return "Error: " + e.message; }
}

function generatePipe() {
  if (!pipeReady) return "// WASM not loaded";
  try { return pipeGenerate(); }
  catch(e) { return "// Error: " + e.message; }
}
