// Pipe WASM — Web Worker frontend
let pipeReady = false;
let callId = 0;
const pending = {};
const worker = new Worker("pipe-worker.js");

worker.onmessage = function(e) {
  const { type, id, text } = e.data;
  if (type === "ready") {
    pipeReady = true;
    document.querySelectorAll(".play-bar").forEach(el => {
      el.innerHTML = '<span>✓</span><span>WASM ready — type code and click Run</span>';
      el.style.color = "#9898a8";
    });
    document.querySelectorAll(".play-bar-text").forEach(el => {
      el.textContent = "WASM ready — type code and click Run";
      el.style.color = "#9898a8";
    });
    document.querySelectorAll("#play-btn, #runBtn").forEach(el => {
      el.disabled = false;
    });
    var icon = document.getElementById("play-bar-icon");
    if (icon) { icon.innerHTML = "✓"; icon.style.color = "#3ce096"; }
    var text = document.getElementById("play-bar-text");
    if (text) text.textContent = "Ready — type code and click Run";
  } else if (type === "result" || type === "error") {
    if (pending[id]) {
      pending[id](text, type === "error");
      delete pending[id];
    }
  }
};

worker.onerror = function(e) {
  document.querySelectorAll("#play-status").forEach(el => {
    el.textContent = "Worker error";
    el.style.color = "#fc5c7c";
  });
};

function runPipe(code) {
  return new Promise(resolve => {
    const id = ++callId;
    pending[id] = (text, isError) => resolve(text);
    const providerEl = document.getElementById("play-provider") || document.getElementById("provider");
    const keyEl = document.getElementById("play-key") || document.getElementById("apikey");
    const provider = providerEl?.value || "deepseek";
    const key = keyEl?.value?.trim() || "";
    worker.postMessage({ id, code, provider, key });
  });
}

async function runPipeAsync(code) {
  document.getElementById("play-out").textContent = "Running...";
  const result = await runPipe(code);
  document.getElementById("play-out").textContent = result || "(no output)";
}
