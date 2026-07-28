let pipeReady = false;
const pipeGo = new Go();

WebAssembly.instantiateStreaming(fetch("pipe.wasm"), pipeGo.importObject).then(r => {
  pipeGo.run(r.instance);
  pipeReady = true;
  const bar = document.getElementById("play-bar");
  if (bar) {
    bar.innerHTML = '<span style="color:#3ce096">✓</span><span>WASM ready</span>';
    bar.style.color = "#9898a8";
  }
  const btn = document.getElementById("play-btn");
  if (btn) btn.disabled = false;
}).catch(e => {
  const bar = document.getElementById("play-bar");
  if (bar) bar.innerHTML = '<span style="color:#fc5c7c">✗</span><span>WASM failed</span>';
});

function runPipe(code) {
  if (!pipeReady) return "WASM not loaded";
  try { return pipeRun(code) || "(no output)"; }
  catch(e) { return "Error: " + e.message; }
}
