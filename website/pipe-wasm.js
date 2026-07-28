// Pipe WASM loader
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
  document.querySelectorAll("#play-btn, #test-ai-btn").forEach(el => { if(el) el.disabled = false; });
}).catch(e => {
  const bar = document.getElementById("play-bar");
  if (bar) {
    bar.innerHTML = '<span style="color:#fc5c7c">✗</span><span>WASM failed</span>';
    bar.style.color = "#fc5c7c";
  }
});

function runPipe(code, provider, key) {
  if (!pipeReady) return "[WASM not loaded]";
  try {
    var result = pipeRun(code, provider || "", key || "");
    return result || "(no output)";
  } catch(e) {
    return "[Error: " + e.message + "]";
  }
}
