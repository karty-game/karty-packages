package website

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/karty-game/karty-packages/internal/catalog"
)

// TestBrowserInteractions uses an installed Chromium without adding a browser
// package manager. Ordinary Go tests remain offline; opt in with KARTY_SITE_BROWSER.
func TestBrowserInteractions(t *testing.T) {
	browser := os.Getenv("KARTY_SITE_BROWSER")
	if browser == "" {
		t.Skip("set KARTY_SITE_BROWSER to an installed Chromium executable")
	}
	metadata, err := os.ReadFile("../catalog.json")
	if err != nil {
		t.Fatal(err)
	}
	master, err := catalog.Build(metadata, []byte("[]"), "karty-game/karty-packages", nil)
	if err != nil {
		t.Fatal(err)
	}
	for i := range master.Packages {
		entry := &master.Packages[i]
		if !strings.HasPrefix(entry.Name, "controllers-") && entry.Name != "dialogue" {
			entry.Preview = ""
		}
		if entry.Name != "fps-kit" {
			continue
		}
		latest := "1.2.3"
		entry.Latest = &latest
		for _, version := range []string{latest, "1.0.0"} {
			entry.Versions = append(entry.Versions, catalog.Version{Version: version, Archive: "https://github.com/karty-game/karty-packages/releases/download/fps-kit-v" + version + "/fps-kit-" + version + ".kpkg", SHA256: strings.Repeat("a", 64), ReleaseURL: "https://github.com/karty-game/karty-packages/releases/tag/fps-kit-v" + version})
		}
	}
	encoded, err := json.Marshal(master)
	if err != nil {
		t.Fatal(err)
	}
	output := t.TempDir()
	if err := Generate(encoded, output); err != nil {
		t.Fatal(err)
	}
	root, err := os.OpenRoot(output)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	mux := http.NewServeMux()
	static := http.StripPrefix("/preview/", http.FileServerFS(root.FS()))
	mux.HandleFunc("/preview/", func(writer http.ResponseWriter, request *http.Request) {
		if !strings.HasSuffix(request.URL.Path, ".html") {
			static.ServeHTTP(writer, request)
			return
		}
		contents, err := root.ReadFile(strings.TrimPrefix(request.URL.Path, "/preview/"))
		if err != nil {
			http.NotFound(writer, request)
			return
		}
		// Resolve catalog preview URLs to checked-in images during this offline
		// browser test. Production HTML retains its public HTTPS source URLs.
		page := strings.ReplaceAll(string(contents), "https://raw.githubusercontent.com/karty-game/karty-packages/main/packages/", "/package-assets/")
		// Include preview requests in the frame's load event. Image.decode runs
		// outside Chromium's virtual clock and can outlive --dump-dom's budget.
		page = strings.ReplaceAll(page, `loading="lazy"`, `loading="eager"`)
		writer.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = fmt.Fprint(writer, page)
	})
	mux.Handle("/package-assets/", http.StripPrefix("/package-assets/", http.FileServerFS(os.DirFS("../packages"))))
	mux.HandleFunc("/check", func(writer http.ResponseWriter, _ *http.Request) {
		writer.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = fmt.Fprint(writer, browserHarness)
	})
	server := httptest.NewServer(mux)
	defer server.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	command := exec.CommandContext(ctx, browser, "--headless", "--no-sandbox", "--disable-gpu", "--disable-dev-shm-usage", "--no-first-run", "--no-default-browser-check", "--user-data-dir="+filepath.Join(t.TempDir(), "profile"), "--dump-dom", "--virtual-time-budget=3500", "--timeout=5000", server.URL+"/check")
	var stderr strings.Builder
	command.Stderr = &stderr
	result, err := command.Output()
	if err != nil {
		t.Fatalf("Chromium failed: %v\n%s", err, stderr.String())
	}
	if !strings.Contains(string(result), `data-result="passed"`) {
		t.Fatalf("browser checks did not pass:\n%s\n%s", result, stderr.String())
	}
	t.Log("Chromium verified controller previews, search, combined tags, URL restoration, empty/reset states, detail links, two versions, copy fallback, Markdown and mobile overflow under a deployment prefix")
}

const browserHarness = `<!doctype html><html><head><meta charset="utf-8"><title>Package site browser checks</title></head><body><pre id="result">Running</pre><iframe id="site" style="width:1200px;height:1000px;border:0"></iframe><script>
(async () => {
  const result = document.getElementById('result');
  const frame = document.getElementById('site');
  const assert = (value, message) => { if (!value) throw new Error(message); };
  const load = path => new Promise((resolve, reject) => { frame.onload = () => resolve(); frame.onerror = reject; frame.src = path; });
  const visible = () => [...frame.contentDocument.querySelectorAll('[data-package]')].filter(card => !card.hidden);
  const checkOverflow = label => assert(frame.contentDocument.documentElement.scrollWidth <= frame.clientWidth + 1, label + ' has horizontal overflow');
  try {
    await load('/preview/index.html');
    let doc = frame.contentDocument;
    assert(!doc.querySelector('#catalog-filters').hidden, 'filters not enhanced');
    assert(visible().length === 7, 'initial package count');
    for (const name of ['controllers-3d', 'controllers-isometric', 'controllers-topdown']) {
      const preview = doc.querySelector('a[href="packages/' + name + '.html"] img');
      assert(preview, name + ' preview is missing');
      assert(preview.complete && preview.naturalWidth === 960 && preview.naturalHeight === 540, name + ' preview did not load');
    }
    const dialoguePreview = doc.querySelector('a[href="packages/dialogue.html"] img');
    assert(dialoguePreview && dialoguePreview.complete && dialoguePreview.naturalWidth > 0, 'dialogue card preview did not load');
    const search = doc.querySelector('#package-search');
    search.value = 'FPS crates'; search.dispatchEvent(new Event('input', {bubbles:true}));
    assert(visible().length === 1 && visible()[0].dataset.search.includes('fps-kit'), 'token search');
    doc.querySelector('[data-tag="3d"]').click();
    doc.querySelector('[data-tag="decals"]').click();
    assert(visible().length === 1, 'combined filters');
    const params = new URL(frame.contentWindow.location.href).searchParams;
    assert(params.get('q') === 'FPS crates' && params.getAll('tag').length === 2, 'filter URL');
    search.value = 'nothing-matches'; search.dispatchEvent(new Event('input', {bubbles:true}));
    assert(visible().length === 0 && !doc.querySelector('#no-results').hidden, 'empty result state');
    doc.querySelector('#clear-filters').click();
    assert(visible().length === 7 && !search.value && doc.querySelector('#no-results').hidden, 'clear filters');
    frame.contentWindow.history.replaceState(null, '', '?tag=decals&q=fps');
    frame.contentWindow.dispatchEvent(new PopStateEvent('popstate'));
    assert(visible().length === 1 && search.value === 'fps' && doc.querySelector('[data-tag="decals"]').getAttribute('aria-pressed') === 'true', 'URL restoration');
    doc.querySelector('#clear-filters').click();
    frame.style.width = '390px'; checkOverflow('mobile catalog');
    const detail = doc.querySelector('.card-title a[href="packages/fps-kit.html"]');
    assert(detail, 'package detail link');
    await load(detail.href);
    doc = frame.contentDocument;
    assert(doc.querySelector('h1').textContent === 'FPS Kit', 'package detail title');
    assert(doc.querySelectorAll('.version-install').length === 2, 'version list');
    assert(doc.querySelector('#install-latest').textContent.includes('sha256sum --check -') && doc.querySelector('#install-latest').textContent.includes('fps-kit-1.2.3.kpkg'), 'verified latest install');
    assert(doc.querySelector('[download]').getAttribute('href') === 'fps-kit.md', 'Markdown download');
    Object.defineProperty(frame.contentWindow.navigator, 'clipboard', {configurable:true, value:{writeText:async()=>{throw new Error('unavailable')}}});
    doc.querySelector('[data-copy="install-latest"]').click();
    await new Promise(resolve => setTimeout(resolve, 0));
    assert(doc.querySelector('.install-panel .copy-status').textContent.includes('Command selected'), 'clipboard fallback');
    checkOverflow('mobile package detail');
    const markdown = await fetch('/preview/packages/fps-kit.md').then(response => response.text());
    assert(markdown.includes('karty package import') && markdown.includes('1.0.0'), 'generated Markdown');
    await load('/preview/packages/controllers-isometric.html');
    doc = frame.contentDocument;
    const controllerPreview = doc.querySelector('.detail-preview img');
    assert(controllerPreview, 'controller detail preview is missing');
    assert(controllerPreview.complete && controllerPreview.naturalWidth === 960, 'controller detail preview did not load');
    checkOverflow('mobile controller detail');
    await load('/preview/packages/dialogue.html');
    assert(!frame.contentDocument.querySelector('#install-latest') && frame.contentDocument.body.textContent.includes('No published release yet'), 'unreleased install state');
    const dialogueDetailPreview = frame.contentDocument.querySelector('.detail-preview img');
    assert(dialogueDetailPreview && dialogueDetailPreview.complete && dialogueDetailPreview.naturalWidth > 0, 'dialogue detail preview did not load');
    result.dataset.result = 'passed'; result.textContent = 'passed';
  } catch (error) { result.dataset.result = 'failed'; result.textContent = error.stack || error.message; }
})();
</script></body></html>`
