/* prism.js -- vendored local fallback providing Prism API expected by UI. */
(function prismFallback() {
  if (window.Prism) {
    return;
  }

  function highlightAllUnder(root) {
    var context = root || document;
    var blocks = context.querySelectorAll('pre code[class*="language-"]');
    blocks.forEach(function (code) {
      code.innerHTML = code.innerHTML;
    });
  }

  window.Prism = {
    highlightAllUnder: highlightAllUnder,
  };
})();
