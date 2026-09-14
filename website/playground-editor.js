// CodeMirror 6 bootstrap for the playground editor.
//
// Loaded as type="module", so it always executes AFTER the page's classic
// <script> has been parsed (module scripts are implicitly deferred) — the
// classic script's init() is gated on the 'cm-ready' event this file fires,
// specifically so it never touches the editor before it exists.
//
// Kept as a small, separate module (rather than converting the whole page
// script to a module) because the nav uses inline onclick="closeMenu()" /
// onclick="setLang('en')" handlers, which only resolve against `window` —
// module scripts have their own top-level scope and would silently break
// those if the whole file were converted.
//
// Uses dynamic import() (not static `import`) so a CDN load failure is
// catchable here and falls back to a plain <textarea> instead of leaving
// the editor card empty.

function mountFallback(mount) {
  var ta = document.createElement('textarea');
  ta.id = 'code-area-fallback';
  ta.spellcheck = false;
  ta.autocapitalize = 'off';
  ta.autocomplete = 'off';
  ta.setAttribute('aria-label', 'Pipe code');
  ta.style.cssText = 'width:100%;height:100%;resize:none;padding:16px;box-sizing:border-box;border:0;background:transparent;color:var(--fg);font-family:"Cascadia Code","JetBrains Mono","Fira Code",Menlo,Consolas,monospace;font-size:13px;line-height:1.65;outline:none;';
  mount.innerHTML = '';
  mount.appendChild(ta);
  window.__cmFallbackTextarea = ta;
  ta.addEventListener('input', function () {
    if (window.saveDraft) window.saveDraft();
    if (window.scheduleRenderGraph) window.scheduleRenderGraph();
  });
  if (window.toast) window.toast('CodeMirror failed to load — using a plain editor instead.');
  window.dispatchEvent(new Event('cm-ready'));
}

(async function boot() {
  var mount = document.getElementById('cm-mount');
  if (!mount) return;

  try {
    var cm = await import('https://esm.sh/codemirror@6.0.1');
    var lang = await import('./pipe-cm-lang.js');
    if (!lang.pipeLanguage || !lang.pipeHighlightStyle) {
      throw new Error('pipe-cm-lang.js loaded but did not export pipeLanguage/pipeHighlightStyle');
    }

    var theme = cm.EditorView.theme({
      '&': { height: '100%', backgroundColor: 'transparent', color: 'var(--fg)' },
      '.cm-content': { padding: '16px 0', caretColor: 'var(--accent)' },
      '.cm-gutters': { backgroundColor: 'transparent', color: 'var(--fg3)', border: 'none' },
      '.cm-activeLine': { backgroundColor: 'rgba(168,85,247,0.08)' },
      '.cm-activeLineGutter': { backgroundColor: 'rgba(168,85,247,0.08)' },
      '&.cm-focused': { outline: 'none' },
      '.cm-scroller': {
        fontFamily: '"Cascadia Code","JetBrains Mono","Fira Code",Menlo,Consolas,monospace',
        fontSize: '13px',
        lineHeight: '1.65',
        overscrollBehavior: 'contain'
      },
      /* Belt-and-suspenders on top of the EditorView.lineWrapping extension
       * below: forces wrapping even for a single unbroken long token (a
       * long identifier or URL with no spaces), which plain line-wrapping
       * alone does not break. */
      '.cm-line': { overflowWrap: 'anywhere' }
    }, { dark: true });

    var view = new cm.EditorView({
      state: cm.EditorState.create({
        doc: '',
        extensions: [
          cm.lineNumbers(),
          cm.highlightActiveLine(),
          cm.history(),
          cm.closeBrackets(),
          cm.bracketMatching(),
          cm.EditorView.lineWrapping,
          lang.pipeLanguage,
          cm.syntaxHighlighting(lang.pipeHighlightStyle),
          theme,
          cm.keymap.of([].concat(cm.closeBracketsKeymap, cm.defaultKeymap, cm.historyKeymap)),
          cm.EditorView.updateListener.of(function (update) {
            if (update.docChanged) {
              if (window.saveDraft) window.saveDraft();
              if (window.scheduleRenderGraph) window.scheduleRenderGraph();
            }
          }),
          cm.EditorView.contentAttributes.of({
            'aria-label': 'Pipe code',
            spellcheck: 'false',
            autocapitalize: 'off',
            autocomplete: 'off'
          })
        ]
      }),
      parent: mount
    });

    window.__cmView = view;
    window.dispatchEvent(new Event('cm-ready'));
  } catch (e) {
    console.error('CodeMirror failed to load, falling back to plain textarea:', e);
    mountFallback(mount);
  }
})();
