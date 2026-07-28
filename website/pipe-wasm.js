// Shared Pipe WASM loader for website
const pipeGo = new Go();
let pipeReady = false;

WebAssembly.instantiateStreaming(fetch("pipe.wasm"), pipeGo.importObject).then(r => {
  pipeGo.run(r.instance);
  pipeReady = true;
  document.querySelectorAll(".play-status").forEach(el => {
    el.textContent = "Ready";
    el.style.color = "#3ce096";
  });
  document.querySelectorAll("#play-btn, #runBtn").forEach(el => {
    el.disabled = false;
  });
}).catch(e => {
  document.querySelectorAll(".play-status").forEach(el => {
    el.textContent = "WASM failed";
    el.style.color = "#fc5c7c";
  });
});

function runPipe(code) {
  if (!pipeReady) return "WASM not loaded yet...";
  try { return pipeRun(code) || "(no output)"; }
  catch(e) { return "Error: " + e; }
}
