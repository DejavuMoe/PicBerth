// API documentation page: theme and copy buttons. The page reads fine without this script.
(function () {
  var pref = 'system';
  try {
    pref = new URLSearchParams(location.search).get('theme') || localStorage.getItem('picberth.theme') || 'system';
  } catch (e) {}
  var mq = matchMedia('(prefers-color-scheme: dark)');
  function apply() {
    var dark = pref === 'dark' || (pref === 'system' && mq.matches);
    document.documentElement.dataset.theme = dark ? 'dark' : 'light';
  }
  apply();
  mq.addEventListener('change', apply);

  document.querySelectorAll('.code').forEach(function (box) {
    var pre = box.querySelector('pre');
    var btn = document.createElement('button');
    btn.type = 'button';
    btn.className = 'btn btn-ghost btn-sm copy';
    btn.textContent = '复制';
    btn.addEventListener('click', function () {
      try {
        navigator.clipboard.writeText(pre.textContent);
      } catch (e) {}
      btn.textContent = '已复制';
      setTimeout(function () {
        btn.textContent = '复制';
      }, 1500);
    });
    box.appendChild(btn);
  });
})();
