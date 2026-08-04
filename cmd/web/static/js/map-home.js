(function () {
  var global = typeof window !== 'undefined' ? window : globalThis;
  var root = document.getElementById('eez-map-root');
  if (!root) {
    return;
  }

  var svgURL = root.getAttribute('data-eez-svg') || '';
  var titleRaw = root.getAttribute('data-title-iso') || '{}';
  var mapAriaBase =
    root.getAttribute('data-map-aria-base') || root.getAttribute('aria-label') || '';
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

  // Pref extract — keep in sync with internal/ogmap PreferredFromIndexJSON.
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

  // Deploy % only when domain_count > 0 (parity with economies table).
  function buildDeployByISO(indexPayload) {
    var out = {};
    if (!indexPayload || !indexPayload.countries) {
      return out;
    }
    for (var k = 0; k < indexPayload.countries.length; k++) {
      var row = indexPayload.countries[k];
      if (!row || !row.iso2) {
        continue;
      }
      if (!(row.domain_count > 0)) {
        continue;
      }
      if (typeof row.deployment_score_pct === 'number' && !isNaN(row.deployment_score_pct)) {
        out[String(row.iso2).toUpperCase()] = row.deployment_score_pct;
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

  var activeMetric = 'pref';
  var preferredByISO = {};
  var deployByISO = {};
  var mapReady = false;

  function pctForMetric(iso, metric) {
    if (metric === 'deploy') {
      return deployByISO[iso];
    }
    return preferredByISO[iso];
  }

  function applyPathMetric(path, metric) {
    var iso = path.getAttribute('data-iso2');
    var baseLabel = path.getAttribute('data-base-label') || iso || '';
    var titleEl = path.querySelector('title');
    if (!titleEl) {
      return;
    }
    var pct = pctForMetric(iso, metric);
    if (pct != null) {
      path.style.setProperty('fill', ramp.colorForPct(pct));
      if (metric === 'deploy') {
        titleEl.textContent =
          baseLabel + ' — ' + pct.toFixed(2) + '% Deploy (domain DNS/Mail/Web/DNSSEC score)';
        path.setAttribute(
          'aria-label',
          baseLabel + ' — ' + pct.toFixed(2) + '% Deploy — open monitoring page'
        );
      } else {
        titleEl.textContent =
          baseLabel + ' — ' + pct.toFixed(2) + '% IPv6 preferred (APNIC Labs estimate)';
        path.setAttribute(
          'aria-label',
          baseLabel + ' — ' + pct.toFixed(2) + '% IPv6 preferred — open monitoring page'
        );
      }
    } else {
      path.style.setProperty('fill', ramp.NO_DATA_GRAY);
      titleEl.textContent = baseLabel;
      path.setAttribute('aria-label', baseLabel + ' — open monitoring page');
    }
  }

  function applyMetricToMap(metric) {
    if (!mapReady) {
      return;
    }
    var linked = root.querySelectorAll('.eez-region--linked');
    for (var i = 0; i < linked.length; i++) {
      applyPathMetric(linked[i], metric);
    }
    var metricLabel = metric === 'deploy' ? 'Deploy %' : 'IPv6 pref. %';
    root.setAttribute(
      'aria-label',
      mapAriaBase
        ? mapAriaBase + ' Colored by ' + metricLabel + '.'
        : 'EEZ map colored by ' + metricLabel + '.'
    );
  }

  function setActiveMetric(metric) {
    if (metric !== 'pref' && metric !== 'deploy') {
      return;
    }
    activeMetric = metric;
    var group = document.querySelector('.eez-map-metric');
    if (group) {
      var buttons = group.querySelectorAll('[role="radio"][data-metric]');
      for (var i = 0; i < buttons.length; i++) {
        var btn = buttons[i];
        var selected = btn.getAttribute('data-metric') === metric;
        btn.setAttribute('aria-checked', selected ? 'true' : 'false');
        btn.tabIndex = selected ? 0 : -1;
      }
    }
    applyMetricToMap(metric);
  }

  function wireMetricControl() {
    var group = document.querySelector('.eez-map-metric');
    if (!group) {
      return;
    }
    var buttons = group.querySelectorAll('[role="radio"][data-metric]');
    for (var i = 0; i < buttons.length; i++) {
      var btn = buttons[i];
      var selected = btn.getAttribute('aria-checked') === 'true';
      btn.tabIndex = selected ? 0 : -1;
      btn.addEventListener('click', function (el) {
        return function () {
          setActiveMetric(el.getAttribute('data-metric'));
        };
      }(btn));
      btn.addEventListener('keydown', function (el) {
        return function (e) {
          var key = e.key;
          if (key !== 'ArrowLeft' && key !== 'ArrowRight' && key !== 'ArrowUp' && key !== 'ArrowDown') {
            return;
          }
          e.preventDefault();
          var list = group.querySelectorAll('[role="radio"][data-metric]');
          var idx = -1;
          for (var j = 0; j < list.length; j++) {
            if (list[j] === el) {
              idx = j;
              break;
            }
          }
          if (idx < 0) {
            return;
          }
          var delta = key === 'ArrowLeft' || key === 'ArrowUp' ? -1 : 1;
          var next = list[(idx + delta + list.length) % list.length];
          setActiveMetric(next.getAttribute('data-metric'));
          next.focus();
        };
      }(btn));
    }
  }

  wireMetricControl();

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
      preferredByISO = buildPreferredByISO(results[1]);
      deployByISO = buildDeployByISO(results[1]);
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
        path.classList.add('eez-region--linked');
        path.setAttribute('data-iso2', iso);
        path.setAttribute('data-base-label', label);
        var prefPct = preferredByISO[iso];
        if (prefPct != null) {
          path.setAttribute('data-ipv6-preferred-pct', String(prefPct));
        }
        var deployPct = deployByISO[iso];
        if (deployPct != null) {
          path.setAttribute('data-deploy-pct', String(deployPct));
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
      mapReady = true;
      setActiveMetric(activeMetric);
    })
    .catch(function () {
      root.textContent = 'Could not load EEZ map.';
    });
})();
