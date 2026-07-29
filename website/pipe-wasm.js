let pipeReady = false;
const pipeGo = new Go();

WebAssembly.instantiateStreaming(fetch("pipe.wasm"), pipeGo.importObject).then(r => {
  pipeGo.run(r.instance);
  pipeReady = true;
  document.getElementById("st").textContent = "ready";
  const btn = document.getElementById("runBtn");
  if (btn) btn.disabled = false;
}).catch(e => {
  document.getElementById("st").textContent = "failed";
});

function runPipe(code) {
  if (!pipeReady) return "WASM not loaded";
  try { return pipeRun(code) || "(no output)"; }
  catch(e) { return "Error: " + e.message; }
}
