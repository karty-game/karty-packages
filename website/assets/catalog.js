(() => {
  "use strict";

  document.querySelectorAll(".preview-image").forEach((image) => {
    const showPlaceholder = () => {
      image.hidden = true;
      const preview = image.closest(".card-preview, .detail-preview");
      const placeholder = preview && preview.querySelector(".card-placeholder");
      if (placeholder) placeholder.hidden = false;
    };
    image.addEventListener("error", showPlaceholder);
    if (image.complete && !image.naturalWidth) showPlaceholder();
  });

  const form = document.getElementById("catalog-filters");
  const search = document.getElementById("package-search");
  if (form && search) {
    const cards = [...document.querySelectorAll("[data-package]")];
    const buttons = [...form.querySelectorAll("button[data-tag]")];
    const count = document.getElementById("result-count");
    const empty = document.getElementById("no-results");
    const clear = document.getElementById("clear-filters");
    const available = new Set(buttons.map((button) => button.dataset.tag));
    const selected = new Set();
    const normalize = (value) => value.toLocaleLowerCase().normalize("NFKC");
    const entries = cards.map((card) => ({
      card,
      text: normalize(card.dataset.search || card.textContent),
      tags: new Set((card.dataset.tags || "").split(/\s+/).filter(Boolean)),
    }));

    function apply(updateURL) {
      const tokens = normalize(search.value)
        .trim()
        .split(/\s+/)
        .filter(Boolean);
      let visible = 0;
      for (const entry of entries) {
        const matches =
          tokens.every((token) => entry.text.includes(token)) &&
          [...selected].every((tag) => entry.tags.has(tag));
        entry.card.hidden = !matches;
        if (matches) visible++;
      }
      for (const button of buttons) {
        button.setAttribute(
          "aria-pressed",
          String(selected.has(button.dataset.tag)),
        );
      }
      if (count)
        count.textContent = `${visible} ${visible === 1 ? "package" : "packages"}${tokens.length || selected.size ? ` of ${cards.length}` : ""}`;
      if (empty) empty.hidden = visible !== 0;
      if (clear) clear.hidden = !search.value && !selected.size;
      if (updateURL) {
        const url = new URL(window.location.href);
        url.searchParams.delete("q");
        url.searchParams.delete("tag");
        if (search.value.trim()) url.searchParams.set("q", search.value.trim());
        [...selected]
          .sort()
          .forEach((tag) => url.searchParams.append("tag", tag));
        window.history.replaceState(null, "", url);
      }
    }

    function readURL() {
      const params = new URLSearchParams(window.location.search);
      search.value = params.get("q") || "";
      selected.clear();
      params.getAll("tag").forEach((tag) => {
        if (available.has(tag)) selected.add(tag);
      });
      apply(false);
    }

    form.addEventListener("submit", (event) => {
      event.preventDefault();
      apply(true);
    });
    search.addEventListener("input", () => apply(true));
    buttons.forEach((button) =>
      button.addEventListener("click", () => {
        const tag = button.dataset.tag;
        if (selected.has(tag)) selected.delete(tag);
        else selected.add(tag);
        apply(true);
      }),
    );
    if (clear)
      clear.addEventListener("click", () => {
        search.value = "";
        selected.clear();
        apply(true);
        search.focus();
      });
    window.addEventListener("popstate", readURL);
    form.querySelectorAll("[data-enhanced]").forEach((element) => {
      element.hidden = false;
    });
    if (form.matches("[data-enhanced]")) form.hidden = false;
    readURL();
  }

  document.querySelectorAll("button[data-copy]").forEach((button) => {
    const target = document.getElementById(button.dataset.copy);
    if (!target) return;
    button.hidden = false;
    button.addEventListener("click", async () => {
      const panel =
        button.closest(".install-panel, details, .panel") ||
        button.parentElement;
      const status = panel.querySelector(".copy-status");
      try {
        await navigator.clipboard.writeText(target.textContent.trim());
        if (status) status.textContent = "Copied to clipboard.";
        button.textContent = "Copied";
        window.setTimeout(() => {
          button.textContent = "Copy";
        }, 2000);
      } catch {
        const selection = window.getSelection();
        const range = document.createRange();
        range.selectNodeContents(target);
        if (selection) {
          selection.removeAllRanges();
          selection.addRange(range);
        }
        if (status)
          status.textContent =
            "Command selected. Use your browser’s copy command.";
      }
    });
  });
})();
