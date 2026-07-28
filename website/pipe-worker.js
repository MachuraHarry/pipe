// Pipe WASM Worker — runs Pipe off the main thread
importScripts("wasm_exec.js");

const go = new Go();
let ready = false;

WebAssembly.instantiateStreaming(fetch("pipe.wasm"), go.importObject).then(r => {
  go.run(r.instance);
  ready = true;
  postMessage({ type: "ready" });
});

onmessage = function(e) {
  if (!ready) {
    postMessage({ type: "error", text: "WASM not loaded yet" });
    return;
  }

  const { id, code, provider, key } = e.data;

  if (provider && key) {
    pipeSetKey(provider, key);
  }

  try {
    const result = pipeRun(code);
    postMessage({ type: "result", id, text: result || "(no output)" });
  } catch(err) {
    postMessage({ type: "error", id, text: "Error: " + err });
  }
};
