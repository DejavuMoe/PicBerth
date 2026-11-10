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

  document.querySelectorAll('.codeblock').forEach(function (box) {
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

  // The table of contents marks the section being read.
  var scroller = document.querySelector('.scroll');
  var secs = [].slice.call(document.querySelectorAll('.doc-sec[id]'));
  function mark() {
    var top = scroller.getBoundingClientRect().top + 24;
    var cur = secs[0];
    secs.forEach(function (sec) {
      if (sec.getBoundingClientRect().top <= top) cur = sec;
    });
    document.querySelectorAll('.doc-toc a').forEach(function (a) {
      if (a.getAttribute('href') === '#' + cur.id) a.setAttribute('aria-current', 'true');
      else a.removeAttribute('aria-current');
    });
  }
  if (scroller && secs.length) {
    scroller.addEventListener('scroll', mark, { passive: true });
    mark();
  }
})();
