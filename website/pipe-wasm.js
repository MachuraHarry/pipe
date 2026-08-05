let pipeReady = false;
const pipeGo = new Go();

WebAssembly.instantiateStreaming(fetch("pipe.wasm?v=4"), pipeGo.importObject).then(r => {
  pipeGo.run(r.instance);
  pipeReady = true;
  const bar = document.getElementById("play-bar");
  if (bar) bar.innerHTML = '<span style="color:var(--green)">✓ ready</span>';
  const btn = document.getElementById("play-btn");
  if (btn) btn.disabled = false;
  const gb = document.getElementById("gen-btn");
  if (gb) gb.disabled = false;
  restoreAPIKey();
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

function setAPIKey(provider, key) {
  if (!pipeReady || !provider || !key) return;
  var envMap = {deepseek:"DEEPSEEK_API_KEY", openai:"OPENAI_API_KEY", anthropic:"ANTHROPIC_API_KEY"};
  var envVar = envMap[provider];
  if (!envVar) return;
  try {
    pipeSetKey(envVar, key);
    localStorage.setItem("pipe-provider", provider);
    localStorage.setItem("pipe-key", key);
    var bar = document.getElementById("play-bar");
    if (bar) { bar.className = "play-bar success"; bar.innerHTML = '<span>✓ key set for ' + provider + '</span>'; }
  } catch(e) { console.error(e); }
}

function restoreAPIKey() {
  var provider = localStorage.getItem("pipe-provider");
  var key = localStorage.getItem("pipe-key");
  if (provider && key) {
    var envMap = {deepseek:"DEEPSEEK_API_KEY", openai:"OPENAI_API_KEY", anthropic:"ANTHROPIC_API_KEY"};
    var envVar = envMap[provider];
    if (envVar) pipeSetKey(envVar, key);
  }
}
