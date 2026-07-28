// Shared Pipe WASM loader for website
const pipeGo = new Go();
let pipeReady = false;

WebAssembly.instantiateStreaming(fetch("pipe.wasm"), pipeGo.importObject).then(r => {
  pipeGo.run(r.instance);
  pipeReady = true;
  document.querySelectorAll(".play-status, #play-status").forEach(el => {
    el.textContent = "Ready — type code and click Run";
    el.style.color = "#9898a8";
  });
  document.querySelectorAll("#play-btn, #runBtn").forEach(el => {
    el.disabled = false;
  });
}).catch(e => {
  document.querySelectorAll(".play-status, #play-status").forEach(el => {
    el.textContent = "WASM failed: " + e.message;
    el.style.color = "#fc5c7c";
  });
});

function runPipe(code) {
  if (!pipeReady) return "WASM not loaded yet...";
  try { return pipeRun(code) || "(no output)"; }
  catch(e) { return "Error: " + e; }
}
