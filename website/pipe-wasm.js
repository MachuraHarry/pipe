// Pipe WASM loader — inline, no worker
let pipeReady = false;
const pipeGo = new Go();

WebAssembly.instantiateStreaming(fetch("pipe.wasm"), pipeGo.importObject).then(r => {
  pipeGo.run(r.instance);
  pipeReady = true;
  
  const bar = document.getElementById("play-bar");
  if (bar) {
    bar.innerHTML = '<span style="color:#3ce096">✓</span><span>WASM ready — type code and click Run</span>';
    bar.style.color = "#9898a8";
  }
  const btn = document.getElementById("play-btn");
  if (btn) btn.disabled = false;
  const testBtn = document.getElementById("test-ai-btn");
  if (testBtn) testBtn.disabled = false;
}).catch(e => {
  const bar = document.getElementById("play-bar");
  if (bar) {
    bar.innerHTML = '<span style="color:#fc5c7c">✗</span><span>WASM failed: ' + e.message + '</span>';
    bar.style.color = "#fc5c7c";
  }
});

function runPipe(code) {
  if (!pipeReady) return "[WASM not loaded]";
  try {
    var result = pipeRun(code);
    if (!result || result === "") return "[pipeRun returned empty]";
    return result;
  } catch(e) {
    return "[JS Error: " + e.message + "]";
  }
}
