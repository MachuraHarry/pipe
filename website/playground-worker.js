'use strict';

/* Pipe Playground worker — runs the Pipe WASM runtime off the main thread. */

importScripts('wasm_exec.js');

var go = new Go();

function fail(err) {
  self.postMessage({ type: 'error', message: String((err && err.message) || err) });
}

function pipeLoaded() {
  return typeof self.pipeRun === 'function';
}

fetch('pipe.wasm')
  .then(function (r) {
    if (!r.ok) throw new Error('HTTP ' + r.status);
    return r.arrayBuffer();
  })
  .then(function (buf) {
    return WebAssembly.instantiate(buf, go.importObject);
  })
  .then(function (res) {
    go.run(res.instance);
    /* The Go main registers pipeRun/pipeParse/pipeGenerate/pipeSetKey/pipeVersion
     * synchronously; poll briefly in case the scheduler defers it. */
    var tries = 0;
    (function poll() {
      if (pipeLoaded()) {
        self.postMessage({ type: 'ready', version: self.pipeVersion || 'unknown' });
      } else if (++tries < 200) {
        setTimeout(poll, 25);
      } else {
        fail(new Error('Pipe runtime did not initialize'));
      }
    })();
  })
  .catch(fail);

/* Sync fetch used by the runtime's HTTP builtins (sandbox allows net in playground). */
self.pipeFetchSync = function (url, opts) {
  var method = (opts && opts.method) || 'GET';
  var body = (opts && opts.body) || undefined;
  var headers = (opts && opts.headers) || {};
  var xhr = new XMLHttpRequest();
  xhr.open(method, url, false);
  try {
    Object.keys(headers).forEach(function (k) {
      try { xhr.setRequestHeader(k, String(headers[k])); } catch (e) {}
    });
  } catch (e) {}
  try {
    xhr.send(body);
    return { body: xhr.responseText, error: xhr.status >= 400 ? 'HTTP ' + xhr.status : '' };
  } catch (e) {
    return { body: '', error: 'error: ' + e.message };
  }
};

self.onmessage = function (e) {
  var d = e.data;
  if (!d) return;
  if (!pipeLoaded()) {
    self.postMessage({ type: 'notready', id: d.id });
    return;
  }
  try {
    switch (d.type) {
      case 'run': {
        var t0 = self.performance.now();
        var out = self.pipeRun(d.code);
        var ms = Math.round(self.performance.now() - t0);
        self.postMessage({ type: 'result', id: d.id, kind: 'run', output: out, ms: ms });
        break;
      }
      case 'parse': {
        var t1 = self.performance.now();
        var json = self.pipeParse(d.code);
        var ms1 = Math.round(self.performance.now() - t1);
        self.postMessage({ type: 'result', id: d.id, kind: 'parse', output: json, ms: ms1 });
        break;
      }
      case 'generate': {
        var t2 = self.performance.now();
        var gen = self.pipeGenerate();
        var ms2 = Math.round(self.performance.now() - t2);
        self.postMessage({ type: 'result', id: d.id, kind: 'generate', output: gen, ms: ms2 });
        break;
      }
      case 'setkey': {
        self.pipeSetKey(d.provider, d.key);
        self.postMessage({ type: 'result', id: d.id, kind: 'setkey', output: 'ok' });
        break;
      }
      default:
        self.postMessage({ type: 'error', id: d.id, message: 'unknown request: ' + d.type });
    }
  } catch (err) {
    self.postMessage({ type: 'result', id: d.id, kind: d.type, output: 'Error: ' + ((err && err.message) || err) });
  }
};
