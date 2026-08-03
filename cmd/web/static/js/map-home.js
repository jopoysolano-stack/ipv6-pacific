(function () {
  var global = typeof window !== 'undefined' ? window : globalThis;
  var root = document.getElementById('eez-map-root');
  if (!root) {
    return;
  }

  var svgURL = root.getAttribute('data-eez-svg') || '';
  var titleRaw = root.getAttribute('data-title-iso') || '{}';
  var TITLE_TO_ISO = {};
  try {
    TITLE_TO_ISO = JSON.parse(titleRaw) || {};
  } catch (e) {
    console.error('eez title map JSON parse failed', e);
  }

  // EEZ paths carry a <title>; land outlines often only have an id — ignore id so we color EEZ only.
  function territoryLabel(pathEl) {
    var tEl = pathEl.querySelector('title');
    if (!tEl) {
      return '';
    }
    return tEl.textContent.replace(/\s+/g, ' ').trim();
  }

  function buildPreferredByISO(indexPayload) {
    var out = {};
    if (!indexPayload || !indexPayload.countries) {
      return out;
    }
    for (var k = 0; k < indexPayload.countries.length; k++) {
      var row = indexPayload.countries[k];
      if (!row || !row.iso2) {
        continue;
      }
      var al = row.apnic_labs;
      if (al && typeof al.preferred_pc_raw === 'number' && !isNaN(al.preferred_pc_raw)) {
        out[String(row.iso2).toUpperCase()] = al.preferred_pc_raw;
      }
    }
    return out;
  }

  var ramp = global.PacificPctColorRamp;
  if (!ramp || typeof ramp.colorForPct !== 'function') {
    ramp = {
      colorForPct: function () {
        return '#9aa3b2';
      },
      NO_DATA_GRAY: '#9aa3b2',
    };
    console.error('PacificPctColorRamp missing; load pct-color-ramp.js before map-home.js');
  }

  if (!svgURL) {
    root.textContent = 'EEZ map not configured.';
    return;
  }

  Promise.all([
    fetch(svgURL).then(function (r) {
      if (!r.ok) {
        throw new Error('fetch failed');
      }
      return r.text();
    }),
    fetch('/api/index.json')
      .then(function (r) {
        return r.ok ? r.json() : null;
      })
      .catch(function () {
        return null;
      }),
  ])
    .then(function (results) {
      var svgText = results[0];
      var preferredByISO = buildPreferredByISO(results[1]);
      var parser = new DOMParser();
      var doc = parser.parseFromString(svgText, 'image/svg+xml');
      var svg = doc.documentElement;
      if (!svg || svg.querySelector('parsererror')) {
        throw new Error('invalid svg');
      }
      svg.setAttribute('class', 'eez-map-svg');
      if (!svg.getAttribute('viewBox')) {
        var w = parseFloat(svg.getAttribute('width')) || 385;
        var h = parseFloat(svg.getAttribute('height')) || 215;
        svg.setAttribute('viewBox', '0 0 ' + w + ' ' + h);
      }
      svg.setAttribute('preserveAspectRatio', 'xMidYMid meet');
      svg.removeAttribute('width');
      svg.removeAttribute('height');
      svg.setAttribute('role', 'img');

      var defs = svg.querySelector('defs');
      var ocean = svg.querySelector('#rect5538-5');
      var vb = (svg.getAttribute('viewBox') || '0 0 385 215').split(/[\s,]+/);
      if (ocean && defs && defs.parentNode === svg && vb.length >= 4) {
        ocean.setAttribute('x', vb[0]);
        ocean.setAttribute('y', vb[1]);
        ocean.setAttribute('width', vb[2]);
        ocean.setAttribute('height', vb[3]);
        defs.parentNode.insertBefore(ocean, defs.nextSibling);
      }

      root.appendChild(svg);

      var svgNS = 'http://www.w3.org/2000/svg';
      var paths = svg.querySelectorAll('path');

      for (var j = 0; j < paths.length; j++) {
        var p = paths[j];
        var territoryName = territoryLabel(p);
        if (!territoryName || !TITLE_TO_ISO[territoryName]) {
          p.classList.add('eez-region--outside');
          p.style.setProperty('fill', '#b8bcc4');
          p.style.setProperty('stroke', '#9ca3af');
          p.style.setProperty('stroke-width', '0.25');
          p.style.setProperty('cursor', 'default');
        }
      }

      var labelLayer = document.createElementNS(svgNS, 'g');
      labelLayer.setAttribute('class', 'eez-iso-labels');
      labelLayer.setAttribute('pointer-events', 'none');

      for (var i = 0; i < paths.length; i++) {
        var path = paths[i];
        var label = territoryLabel(path);
        var iso = label ? TITLE_TO_ISO[label] : '';
        if (!iso) {
          continue;
        }
        var titleEl = path.querySelector('title');
        if (!titleEl) {
          titleEl = document.createElementNS(svgNS, 'title');
          path.insertBefore(titleEl, path.firstChild);
        }
        var baseLabel = label;
        path.classList.add('eez-region--linked');
        path.setAttribute('data-iso2', iso);
        var pct = preferredByISO[iso];
        if (pct != null) {
          path.style.setProperty('fill', ramp.colorForPct(pct));
          path.setAttribute('data-ipv6-preferred-pct', String(pct));
          titleEl.textContent =
            baseLabel + ' — ' + pct.toFixed(2) + '% IPv6 preferred (APNIC Labs estimate)';
          path.setAttribute(
            'aria-label',
            baseLabel + ' — ' + pct.toFixed(2) + '% IPv6 preferred — open monitoring page'
          );
        } else {
          path.style.setProperty('fill', ramp.NO_DATA_GRAY);
          titleEl.textContent = baseLabel;
          path.setAttribute('aria-label', baseLabel + ' — open monitoring page');
        }
        path.style.setProperty('stroke', '#4b5563');
        path.style.setProperty('stroke-width', '0.25');
        path.style.cursor = 'pointer';
        path.setAttribute('tabindex', '0');
        path.setAttribute('role', 'link');

        path.addEventListener('click', function (iso2) {
          return function (e) {
            e.preventDefault();
            window.location.href = '/country/' + iso2;
          };
        }(iso));

        path.addEventListener('keydown', function (iso2) {
          return function (e) {
            if (e.key === 'Enter' || e.key === ' ') {
              e.preventDefault();
              window.location.href = '/country/' + iso2;
            }
          };
        }(iso));

        try {
          var box = path.getBBox();
          if (box.width < 0.5 || box.height < 0.5) {
            continue;
          }
          var cx = box.x + box.width / 2;
          var cy = box.y + box.height / 2;
          var dim = Math.min(box.width, box.height);
          var fontSize = Math.min(13, Math.max(5, dim * 0.38));

          var text = document.createElementNS(svgNS, 'text');
          text.setAttribute('x', String(cx));
          text.setAttribute('y', String(cy));
          text.setAttribute('text-anchor', 'middle');
          text.setAttribute('dominant-baseline', 'central');
          text.setAttribute('font-size', String(fontSize));
          text.setAttribute('class', 'eez-iso-label');
          text.textContent = iso;
          labelLayer.appendChild(text);
        } catch (e) {
          /* ignore bbox errors */
        }
      }

      svg.appendChild(labelLayer);
    })
    .catch(function () {
      root.textContent = 'Could not load EEZ map.';
    });
})();
