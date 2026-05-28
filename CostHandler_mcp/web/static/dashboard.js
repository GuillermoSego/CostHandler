/* ============================================================
   CostHandler · Coach dashboard
   Vanilla JS (no Chart.js). Renders insights, categories,
   calendar heatmap, monthly trend sparkline, and recent
   transactions from the same /api/dashboard/summary and
   /api/expenses endpoints.
   ============================================================ */

// ---- State ---------------------------------------------------
let state = {
  period:        'month',
  category:      '',
  user:          '',
  view:          'resumen',
  selectedMonth: null,
};

// ---- Money / time formatters --------------------------------
function formatMoney(n, { sign = false, short = false } = {}) {
  if (n == null || isNaN(n)) n = 0;
  const abs = Math.abs(Math.round(n));
  let body;
  if (short && abs >= 1000) {
    const v = abs / 1000;
    body = v.toFixed(v >= 10 ? 1 : 1).replace(/\.0$/, '') + 'k';
  } else {
    body = abs.toLocaleString('es-MX');
  }
  const prefix = sign
    ? (n > 0 ? '+MX$' : n < 0 ? '−MX$' : 'MX$')
    : (n < 0 ? '−MX$' : 'MX$');
  return prefix + body;
}
function formatMoneyShort(n) { return formatMoney(n, { short: true }); }

const MONTHS_ES = ['enero', 'febrero', 'marzo', 'abril', 'mayo', 'junio',
                   'julio', 'agosto', 'septiembre', 'octubre', 'noviembre', 'diciembre'];
const MONTHS_ES_SHORT = ['ene', 'feb', 'mar', 'abr', 'may', 'jun',
                         'jul', 'ago', 'sep', 'oct', 'nov', 'dic'];
const DAYS_ES = ['domingo', 'lunes', 'martes', 'miércoles', 'jueves', 'viernes', 'sábado'];
const DAYS_ES_SHORT = ['dom', 'lun', 'mar', 'mié', 'jue', 'vie', 'sáb'];

function todayString() {
  const d = new Date();
  const day = DAYS_ES[d.getDay()];
  return day.charAt(0).toUpperCase() + day.slice(1) + ' ' + d.getDate() + ' de ' + MONTHS_ES[d.getMonth()] +
    ', ' + String(d.getHours()).padStart(2, '0') + ':' + String(d.getMinutes()).padStart(2, '0');
}
function greetingFor(date) {
  const h = date.getHours();
  if (h < 6)  return 'Buenas noches';
  if (h < 13) return 'Buenos días';
  if (h < 20) return 'Buenas tardes';
  return 'Buenas noches';
}

// ---- Category styling ---------------------------------------
// Matches the original JS but tuned to the Rappi palette. Keys
// are lowercase Spanish category names; the UI title-cases them.
const CATEGORY_COLORS = {
  supermercado:    '#00B45C',
  restaurantes:    '#FF441F',
  vivienda:        '#2F6BFF',
  servicios:       '#9C27B0',
  transporte:      '#FFB020',
  salud:           '#00BCD4',
  familia:         '#E91E63',
  suscripciones:   '#5C6370',
  entretenimiento: '#7A5AE0',
  compras:         '#FF7A4D',
  ahorro:          '#1F8A5B',
  otros:           '#919AAA',
};
function catColor(name) {
  const k = (name || '').toLowerCase();
  return CATEGORY_COLORS[k] || '#919AAA';
}

// Inline SVG icon strings, monoline / rounded, matching the Rappi
// iconography. Returned as raw SVG text for innerHTML insertion.
const ICONS = {
  fork:  '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><path d="M7 3v8a2 2 0 0 0 2 2v8M5 3v6M9 3v6M16 3c-2 0-3 2-3 5s1 4 3 4v9"/></svg>',
  cart:  '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><path d="M3 4h2l2.4 11.2a2 2 0 0 0 2 1.6h7.2a2 2 0 0 0 2-1.5L21 8H6"/><circle cx="10" cy="20" r="1.2"/><circle cx="17" cy="20" r="1.2"/></svg>',
  home:  '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><path d="M3 11l9-7 9 7v9a1 1 0 0 1-1 1h-5v-6h-6v6H4a1 1 0 0 1-1-1z"/></svg>',
  bolt:  '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><path d="M13 2L4 14h7l-1 8 9-12h-7z"/></svg>',
  car:   '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><path d="M5 17v2M19 17v2"/><path d="M3 13l2-5a2 2 0 0 1 2-1h10a2 2 0 0 1 2 1l2 5"/><rect x="3" y="13" width="18" height="5" rx="1.5"/></svg>',
  cross: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><path d="M9 3h6v6h6v6h-6v6H9v-6H3V9h6z"/></svg>',
  heart: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><path d="M12 21s-7-4.5-9-9a5 5 0 0 1 9-3 5 5 0 0 1 9 3c-2 4.5-9 9-9 9z"/></svg>',
  sub:   '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><path d="M4 4h12l4 4v12H4z"/><path d="M16 4v4h4M8 13h8M8 17h5"/></svg>',
  film:  '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><rect x="3" y="4" width="18" height="16" rx="2"/><path d="M3 9h18M3 15h18M8 4v16M16 4v16"/></svg>',
  bag:   '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><path d="M5 8h14l-1 12H6z"/><path d="M9 8V6a3 3 0 0 1 6 0v2"/></svg>',
  piggy: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><path d="M4 13a6 6 0 0 1 6-6h4a6 6 0 0 1 6 6v3a2 2 0 0 1-2 2h-1v2h-2v-2H9v2H7v-2H6a2 2 0 0 1-2-2z"/><circle cx="15" cy="13" r="1" fill="currentColor"/><path d="M4 13c-1 0-1.5-1-1.5-1.5S3 10 4 10"/></svg>',
  circle:'<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="8"/></svg>',
  trendingDown: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M3 7l6 6 4-4 8 9"/><path d="M14 18h7v-7"/></svg>',
  trendingUp:   '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M3 17l6-6 4 4 8-9"/><path d="M14 6h7v7"/></svg>',
  alert:        '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M12 3L2 21h20z"/><path d="M12 10v5M12 18h.01"/></svg>',
  compass:      '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="9"/><path d="M15 9l-2 6-6 2 2-6z"/></svg>',
  spark:        '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M12 2l1.6 4.6L18 8l-4.4 1.4L12 14l-1.6-4.6L6 8l4.4-1.4z"/><path d="M19 14l.8 2.2L22 17l-2.2.8L19 20l-.8-2.2L16 17l2.2-.8z"/></svg>',
};
const CAT_ICONS = {
  supermercado:    'cart',
  restaurantes:    'fork',
  vivienda:        'home',
  servicios:       'bolt',
  transporte:      'car',
  salud:           'cross',
  familia:         'heart',
  suscripciones:   'sub',
  entretenimiento: 'film',
  compras:         'bag',
  ahorro:          'piggy',
  otros:           'circle',
};
function catIcon(name) { return ICONS[CAT_ICONS[(name || '').toLowerCase()] || 'circle']; }

// ---- HTML escape -------------------------------------------
function escapeHtml(str) {
  const div = document.createElement('div');
  div.textContent = str == null ? '' : String(str);
  return div.innerHTML;
}

// ---- DOM ready ---------------------------------------------
document.addEventListener('DOMContentLoaded', function() {
  // Greeting + date
  document.getElementById('greeting').textContent = greetingFor(new Date()) + ' 👋';
  document.getElementById('today-date').textContent = todayString();

  // Period segment
  const seg = document.getElementById('period-segment');
  seg.querySelectorAll('.segment__item').forEach((b) => {
    b.addEventListener('click', () => {
      seg.querySelectorAll('.segment__item').forEach((x) => x.classList.remove('is-active'));
      b.classList.add('is-active');
      state.period = b.dataset.period;
      document.getElementById('month-select').style.display = state.period === 'month' ? '' : 'none';
      if (state.period !== 'month') state.selectedMonth = null;
      loadDashboard();
    });
  });

  // Month selector
  const monthSel = document.getElementById('month-select');
  const _now = new Date();
  const _curYear = _now.getFullYear();
  const _curMonth = _now.getMonth();
  for (let m = 0; m <= _curMonth; m++) {
    const opt = document.createElement('option');
    opt.value = _curYear + '-' + String(m + 1).padStart(2, '0');
    opt.textContent = MONTHS_ES[m].charAt(0).toUpperCase() + MONTHS_ES[m].slice(1);
    monthSel.appendChild(opt);
  }
  monthSel.value = _curYear + '-' + String(_curMonth + 1).padStart(2, '0');
  monthSel.addEventListener('change', (e) => {
    const val = e.target.value;
    const cur = _curYear + '-' + String(_curMonth + 1).padStart(2, '0');
    state.selectedMonth = val === cur ? null : val;
    loadDashboard();
  });

  // User filter
  document.getElementById('user-select').addEventListener('change', (e) => {
    state.user = e.target.value;
    updateUserCard();
    loadDashboard();
  });

  // Sidebar navigation
  document.querySelectorAll('.side .ni').forEach((a) => {
    a.addEventListener('click', (e) => {
      e.preventDefault();
      const hash = a.getAttribute('href') || '#resumen';
      window.location.hash = hash;
    });
  });
  window.addEventListener('hashchange', () => navigateTo(window.location.hash));

  // "Ver todas" / "Ver todos" links
  document.querySelectorAll('.panel__link').forEach((a) => {
    a.addEventListener('click', (e) => {
      e.preventDefault();
      window.location.hash = a.getAttribute('href');
    });
  });

  // Add expense modal
  document.getElementById('add-expense').addEventListener('click', () => {
    document.getElementById('expense-modal').style.display = '';
  });
  document.getElementById('modal-close').addEventListener('click', closeExpenseModal);
  document.getElementById('modal-cancel').addEventListener('click', closeExpenseModal);
  document.getElementById('expense-modal').addEventListener('click', (e) => {
    if (e.target === e.currentTarget) closeExpenseModal();
  });
  document.getElementById('expense-form').addEventListener('submit', handleExpenseSubmit);

  // Edit expense modal
  document.getElementById('edit-modal-close').addEventListener('click', closeEditModal);
  document.getElementById('edit-modal-cancel').addEventListener('click', closeEditModal);
  document.getElementById('edit-modal').addEventListener('click', (e) => {
    if (e.target === e.currentTarget) closeEditModal();
  });
  document.getElementById('edit-form').addEventListener('submit', handleEditSubmit);

  updateUserCard();
  navigateTo(window.location.hash);
  loadUsers();
  loadDashboard();
});

async function loadUsers() {
  try {
    const res = await fetch('/api/users');
    const users = await res.json();
    const sel = document.getElementById('user-select');
    const current = sel.value;
    sel.innerHTML = '<option value="">Todos los usuarios</option>';
    (users || []).forEach((u) => {
      const opt = document.createElement('option');
      opt.value = u;
      opt.textContent = u;
      sel.appendChild(opt);
    });
    if (current) sel.value = current;
  } catch (err) {
    console.error('Error cargando usuarios:', err);
  }
}

function updateUserCard() {
  const card = document.getElementById('user-card');
  const name = document.getElementById('user-name');
  const sub  = document.getElementById('user-sub');
  const av   = document.getElementById('user-avatar');
  if (state.user) {
    card.classList.remove('is-empty');
    name.textContent = state.user;
    sub.textContent  = 'Presupuesto activo';
    av.textContent   = state.user.slice(0, 2).toUpperCase();
  } else {
    card.classList.add('is-empty');
    name.textContent = 'Sin usuario';
    sub.textContent  = 'Define un usuario para presupuesto';
    av.textContent   = '·';
  }
}

function appendMonthParams(params) {
  if (state.period === 'month' && state.selectedMonth) {
    const [y, m] = state.selectedMonth.split('-').map(Number);
    const firstDay = y + '-' + String(m).padStart(2, '0') + '-01';
    const lastDay = new Date(y, m, 0).getDate();
    const toDate = y + '-' + String(m).padStart(2, '0') + '-' + String(lastDay).padStart(2, '0');
    params.set('from', firstDay);
    params.set('to', toDate);
  }
}

// ---- Data fetch + render orchestration ---------------------
async function loadDashboard() {
  const params = new URLSearchParams({ period: state.period });
  if (state.category) params.set('category', state.category);
  if (state.user)     params.set('user',     state.user);
  appendMonthParams(params);

  const msiParams = new URLSearchParams();
  if (state.user) msiParams.set('user', state.user);

  try {
    const [sumRes, expRes, msiRes] = await Promise.all([
      fetch('/api/dashboard/summary?' + params),
      fetch('/api/expenses?' + params),
      fetch('/api/installments?' + msiParams),
    ]);
    const summary  = await sumRes.json();
    const expenses = await expRes.json();
    const installments = await msiRes.json();
    render(summary || {}, expenses || [], installments || {});

    if (state.view === 'gastos') loadGastosView();
    if (state.view === 'categorias') loadCategoriasView();
    if (state.view === 'presupuestos') loadPresupuestosView();
  } catch (err) {
    console.error('Error cargando dashboard:', err);
  }
}

function render(summary, expenses, installments) {
  const byCategory = summary.by_category   || [];
  const byDay      = summary.by_day        || [];
  const byMonth    = summary.by_month      || [];
  const budgets    = summary.budget_comparison || [];

  const total      = +summary.total_amount   || 0;
  const prev       = +summary.prev_total     || 0;
  const dailyAvg   = +summary.daily_average  || 0;
  const top        = summary.top_category    || '';

  renderHero(total, prev, dailyAvg, budgets, byCategory);
  renderInsights(total, prev, dailyAvg, budgets, byCategory, top);
  renderCategories(byCategory, budgets, total);
  renderCalendar(byDay, expenses);
  renderTrend(byMonth);
  renderInstallments(installments || {});
  renderTransactions(expenses);
}

// ============================================================
// HERO
// ============================================================
function renderHero(total, prev, dailyAvg, budgets, byCategory) {
  // Eyebrow
  const ey = document.getElementById('hero-eyebrow');
  const selMonth = state.selectedMonth
    ? MONTHS_ES[parseInt(state.selectedMonth.split('-')[1], 10) - 1]
    : MONTHS_ES[new Date().getMonth()];
  ey.textContent = state.period === 'week'  ? 'Has gastado esta semana'
                  : state.period === 'year' ? 'Has gastado este año'
                  : 'Has gastado en ' + selMonth;

  // Total
  document.getElementById('total-amount').textContent =
    Math.round(total).toLocaleString('es-MX');

  // Prev pill
  const pill = document.getElementById('prev-pill');
  pill.classList.remove('is-good', 'is-warn', 'is-bad');
  if (prev > 0) {
    const diff = total - prev;
    const pct  = Math.abs((diff / prev) * 100);
    const tone = diff <= 0 ? 'is-good' : 'is-bad';
    pill.classList.add(tone);
    const icon = diff <= 0 ? ICONS.trendingDown : ICONS.trendingUp;
    const verb = diff <= 0 ? 'menos' : 'más';
    const prevLabel = state.period === 'week' ? 'la semana pasada'
                     : state.period === 'year' ? 'el año pasado'
                     : 'el mes pasado';
    pill.innerHTML = icon + '<span>' + formatMoney(Math.abs(diff)) +
      ' ' + verb + ' que ' + prevLabel + ' (' + pct.toFixed(0) + '%)</span>';
  } else {
    pill.innerHTML = 'Primer período registrado';
  }

  // Day-of message
  const dayMsg = document.getElementById('day-of');
  const now = new Date();
  const isPastMonth = state.selectedMonth && state.period === 'month' &&
    state.selectedMonth !== (now.getFullYear() + '-' + String(now.getMonth() + 1).padStart(2, '0'));
  if (isPastMonth) {
    dayMsg.textContent = '· mes cerrado';
  } else if (state.period === 'month') {
    const dim = new Date(now.getFullYear(), now.getMonth() + 1, 0).getDate();
    dayMsg.textContent = '· día ' + now.getDate() + ' de ' + dim;
  } else if (state.period === 'week') {
    dayMsg.textContent = '· día ' + ((now.getDay() + 6) % 7 + 1) + ' de 7';
  } else {
    dayMsg.textContent = '· mes ' + (now.getMonth() + 1) + ' de 12';
  }

  // Coach copy — adaptive narrative
  document.getElementById('coach-line').innerHTML = coachCopy(total, prev, dailyAvg, budgets);

  // Budget ring
  renderBudgetRing(total, budgets);
}

function coachCopy(total, prev, dailyAvg, budgets) {
  const now = new Date();
  const budgetTotal = budgets.reduce((s, b) => s + (+b.budgeted || 0), 0);
  const parts = [];

  if (prev > 0) {
    const pct = Math.abs(((total - prev) / prev) * 100).toFixed(0);
    if (total <= prev) {
      parts.push('Vas un <b class="accent">' + pct + '% mejor</b> que el período anterior.');
    } else {
      parts.push('Llevas un <b class="accent">' + pct + '% más</b> que el período anterior.');
    }
  }

  const isPast = state.selectedMonth && state.period === 'month' &&
    state.selectedMonth !== (now.getFullYear() + '-' + String(now.getMonth() + 1).padStart(2, '0'));

  if (state.period === 'month' && isPast) {
    parts.push('Cerraste el mes en <b class="accent">' + formatMoney(total) + '</b>.');
    if (budgetTotal > 0) {
      const diff = total - budgetTotal;
      if (diff <= 0) {
        parts.push('Te mantuviste <b>' + formatMoney(Math.abs(diff)) + '</b> por debajo de tu presupuesto.');
      } else {
        parts.push('Te pasaste <b>' + formatMoney(diff) + '</b> sobre tu presupuesto.');
      }
    }
  } else if (state.period === 'month') {
    const day = now.getDate();
    const dim = new Date(now.getFullYear(), now.getMonth() + 1, 0).getDate();
    const remaining = dim - day;
    const projected = day > 0 ? Math.round((total / day) * dim) : total;
    if (remaining > 0) {
      parts.push('Te quedan <b>' + remaining + ' día' + (remaining === 1 ? '' : 's') + '</b>.');
    }
    if (budgetTotal > 0) {
      const overUnder = projected - budgetTotal;
      if (overUnder <= 0) {
        parts.push('Si mantienes el ritmo, cerrarás cerca de <b class="accent">' + formatMoney(projected) +
          '</b> — <b>' + formatMoney(Math.abs(overUnder)) + '</b> por debajo de tu presupuesto.');
      } else {
        parts.push('A este ritmo proyectas <b class="accent">' + formatMoney(projected) +
          '</b>, <b>' + formatMoney(overUnder) + '</b> sobre tu presupuesto.');
      }
    } else {
      parts.push('A este ritmo proyectas cerrar en <b class="accent">' + formatMoney(projected) + '</b>.');
    }
  } else if (state.period === 'week') {
    parts.push('Tu promedio diario va en <b>' + formatMoney(dailyAvg) + '</b>.');
  } else {
    parts.push('Tu promedio diario en el año va en <b>' + formatMoney(dailyAvg) + '</b>.');
  }

  return parts.join(' ');
}

function renderBudgetRing(total, budgets) {
  const budgetTotal = budgets.reduce((s, b) => s + (+b.budgeted || 0), 0);
  const ring   = document.getElementById('budget-ring-fg');
  const label  = document.getElementById('ring-label');
  const pctEl  = document.getElementById('ring-pct');
  const subEl  = document.getElementById('ring-sub');
  const C = 2 * Math.PI * 98;
  ring.setAttribute('stroke-dasharray', C.toFixed(2));

  if (budgetTotal > 0) {
    const pct  = total / budgetTotal;
    const cap  = Math.min(pct, 1);
    ring.setAttribute('stroke-dashoffset', (C * (1 - cap)).toFixed(2));
    ring.setAttribute('stroke', pct >= 1 ? 'var(--rp-danger)' : pct >= 0.85 ? 'var(--rp-warning)' : 'var(--rp-orange)');
    label.textContent = pct >= 1 ? 'Excediste' : 'Presupuesto';
    pctEl.textContent = Math.round(pct * 100) + '%';
    const remain = budgetTotal - total;
    subEl.textContent = remain >= 0
      ? formatMoney(remain) + ' libres'
      : formatMoney(Math.abs(remain)) + ' encima';
  } else {
    ring.setAttribute('stroke-dashoffset', C.toFixed(2));
    ring.setAttribute('stroke', 'var(--rp-bg-2)');
    label.textContent = 'Sin presupuesto';
    pctEl.textContent = '—';
    subEl.textContent = 'Define un usuario para activarlo';
  }
}

// ============================================================
// INSIGHTS
// ============================================================
function renderInsights(total, prev, dailyAvg, budgets, byCategory, topCategory) {
  const cards = [];

  // 1) Vs previous period
  if (prev > 0) {
    const diff = total - prev;
    if (diff <= 0) {
      cards.push({
        tone: 'good', icon: ICONS.trendingDown,
        head: formatMoney(Math.abs(diff)) + ' menos que antes',
        body: 'Vas un ' + Math.abs(((diff / prev) * 100)).toFixed(0) + '% mejor que el período anterior. Si sigues así, cerrarás bajo presupuesto.',
      });
    } else {
      cards.push({
        tone: 'warn', icon: ICONS.trendingUp,
        head: formatMoney(diff) + ' más que antes',
        body: 'Vas un ' + ((diff / prev) * 100).toFixed(0) + '% por encima del período anterior. Revisa qué categoría creció.',
      });
    }
  } else {
    cards.push({
      tone: 'neutral', icon: ICONS.spark,
      head: 'Comencemos este período',
      body: 'Aún no hay datos comparables. Tus insights aparecerán a medida que registres gastos.',
    });
  }

  // 2) Worst category vs budget (or biggest mover)
  const over = budgets
    .filter((b) => b.budgeted > 0)
    .map((b) => ({ ...b, pct: b.percentage ?? (b.spent / b.budgeted) * 100 }))
    .sort((a, b) => b.pct - a.pct)[0];
  if (over && over.pct >= 90) {
    const isOver = over.pct >= 100;
    cards.push({
      tone: isOver ? 'warn' : 'neutral',
      icon: isOver ? ICONS.alert : ICONS.spark,
      head: capitalize(over.category) + (isOver ? ' ya pasó tu presupuesto' : ' está cerca del límite'),
      body: 'Gastaste ' + formatMoney(over.spent) + ' de ' + formatMoney(over.budgeted) +
            ' (' + over.pct.toFixed(0) + '%). ' +
            (isOver ? 'Considera pausar gastos de esta categoría.' : 'Te queda poco margen.'),
    });
  } else if (topCategory && byCategory.length) {
    const t = byCategory.find((c) => c.category === topCategory) || byCategory[0];
    const sharePct = total > 0 ? (t.total / total) * 100 : 0;
    cards.push({
      tone: 'neutral', icon: ICONS.spark,
      head: 'Tu categoría top: ' + capitalize(t.category),
      body: 'Representa el ' + sharePct.toFixed(0) + '% (' + formatMoney(t.total) + ') de lo que llevas gastado.',
    });
  } else {
    cards.push({
      tone: 'neutral', icon: ICONS.spark,
      head: 'Sin categoría dominante',
      body: 'Tus gastos están repartidos. Registra más movimientos para descubrir patrones.',
    });
  }

  // 3) Projection or daily average insight
  if (state.period === 'month') {
    const now = new Date();
    const day = now.getDate();
    const dim = new Date(now.getFullYear(), now.getMonth() + 1, 0).getDate();
    const projected = day > 0 ? Math.round((total / day) * dim) : total;
    const remaining = dim - day;
    const budgetTotal = budgets.reduce((s, b) => s + (+b.budgeted || 0), 0);
    if (remaining > 0) {
      let body;
      if (budgetTotal > 0) {
        const delta = budgetTotal - projected;
        body = 'Quedan ' + remaining + ' día' + (remaining === 1 ? '' : 's') + '. Si gastas tu promedio diario (' + formatMoney(dailyAvg) + '), terminarás ' +
          (delta >= 0 ? formatMoney(delta) + ' por debajo' : formatMoney(Math.abs(delta)) + ' por encima') +
          ' de tu presupuesto.';
      } else {
        body = 'Quedan ' + remaining + ' día' + (remaining === 1 ? '' : 's') + '. Con tu promedio diario de ' + formatMoney(dailyAvg) + ', cerrarás el mes en este monto.';
      }
      cards.push({
        tone: 'neutral', icon: ICONS.compass,
        head: 'A este ritmo cerrarás en ' + formatMoney(projected),
        body,
      });
    } else {
      cards.push({
        tone: 'good', icon: ICONS.compass,
        head: 'Cerraste el mes en ' + formatMoney(total),
        body: 'Tu promedio fue de ' + formatMoney(dailyAvg) + ' por día.',
      });
    }
  } else {
    cards.push({
      tone: 'neutral', icon: ICONS.compass,
      head: 'Promedio: ' + formatMoney(dailyAvg) + ' por día',
      body: 'Es tu ritmo de gasto promedio en este período.',
    });
  }

  const el = document.getElementById('insights');
  el.innerHTML = cards.slice(0, 3).map((c) => `
    <article class="insight">
      <div class="insight__icon is-${c.tone}">${c.icon}</div>
      <h3 class="insight__head">${escapeHtml(c.head)}</h3>
      <p class="insight__body">${escapeHtml(c.body)}</p>
    </article>
  `).join('');
}

function capitalize(s) {
  if (!s) return '';
  return s.charAt(0).toUpperCase() + s.slice(1);
}

// ============================================================
// CATEGORIES
// ============================================================
function renderCategories(byCategory, budgets, total) {
  const list = document.getElementById('cat-list');
  const sub  = document.getElementById('cat-sub');

  if (!byCategory.length) {
    sub.textContent = '';
    list.innerHTML = '<div class="txn-empty">Sin gastos en este período</div>';
    return;
  }

  sub.textContent = byCategory.length + ' ' +
    (byCategory.length === 1 ? 'categoría' : 'categorías') +
    (budgets.length ? ' · comparado con tu presupuesto' : ' · ordenadas por gasto');

  renderCategoryList(list, byCategory, budgets, total);
}

// ============================================================
// CALENDAR HEATMAP (month period)
// ============================================================
function renderCalendar(byDay, expenses) {
  const grid = document.getElementById('cal-grid');
  const dow  = document.getElementById('cal-dow');
  const meta = document.getElementById('calendar-meta');
  const calPanel = grid.closest('.panel');

  if (state.period !== 'month') {
    dow.style.display  = 'none';
    grid.style.display = 'none';
    meta.textContent   = 'Disponible en vista mensual';
    if (calPanel) calPanel.style.display = 'none';
    return;
  }
  dow.style.display  = '';
  grid.style.display = '';
  if (calPanel) calPanel.style.display = '';

  let year, month, today;
  if (state.selectedMonth) {
    const parts = state.selectedMonth.split('-').map(Number);
    year = parts[0];
    month = parts[1] - 1;
    const realNow = new Date();
    if (year === realNow.getFullYear() && month === realNow.getMonth()) {
      today = realNow.getDate();
    } else {
      today = new Date(year, month + 1, 0).getDate();
    }
  } else {
    const now = new Date();
    year = now.getFullYear();
    month = now.getMonth();
    today = now.getDate();
  }
  const dim = new Date(year, month + 1, 0).getDate();
  const wd0 = (new Date(year, month, 1).getDay() + 6) % 7;

  const byDayMap = {};

  // Primary: use by_day from summary API (DATE() aggregation)
  (byDay || []).forEach((d) => {
    const ds = String(d.date || '');
    const parts = ds.match(/^(\d{4})-(\d{2})-(\d{2})$/);
    if (parts) {
      const m = parseInt(parts[2], 10);
      const dy = parseInt(parts[3], 10);
      if (m === month + 1 && parseInt(parts[1], 10) === year) {
        byDayMap[dy] = (byDayMap[dy] || 0) + (+d.total || 0);
      }
    } else {
      const fallback = ds.match(/(\d{1,2})$/);
      if (fallback) byDayMap[+fallback[1]] = (+d.total || 0);
    }
  });

  // Fallback: if by_day was empty/null, compute from expenses list
  if (Object.keys(byDayMap).length === 0 && expenses && expenses.length) {
    expenses.forEach((e) => {
      const ds = String(e.created_at || '').replace('T', ' ');
      const parts = ds.match(/^(\d{4})-(\d{2})-(\d{2})/);
      if (parts && parseInt(parts[1], 10) === year && parseInt(parts[2], 10) === month + 1) {
        const dy = parseInt(parts[3], 10);
        byDayMap[dy] = (byDayMap[dy] || 0) + (+e.amount || 0);
      }
    });
  }

  let maxAmt = 0, peakDay = 0;
  for (let i = 1; i <= today; i++) {
    if ((byDayMap[i] || 0) > maxAmt) { maxAmt = byDayMap[i]; peakDay = i; }
  }
  if (maxAmt === 0) maxAmt = 1;

  let html = '';
  for (let i = 0; i < wd0; i++) html += '<div class="cal-cell is-empty"></div>';
  for (let d = 1; d <= dim; d++) {
    const v = byDayMap[d] || 0;
    const future = d > today;
    const isToday = d === today;
    let style = '', cls = 'cal-cell';
    if (future) {
      cls += ' is-future';
    } else if (v > 0) {
      const intensity = v / maxAmt;
      const alpha = (0.18 + intensity * 0.82).toFixed(2);
      style = 'background-color: rgba(255, 68, 31, ' + alpha + '); color: ' + (intensity > 0.55 ? '#fff' : 'var(--rp-ink)') + ';';
    }
    if (isToday) cls += ' is-today';
    const amt = v > 0 && !future ? '<div class="cal-cell__amount">' + formatMoneyShort(v) + '</div>' : '';
    html += '<div class="' + cls + '" style="' + style + '"><span>' + d + '</span>' + amt + '</div>';
  }
  grid.innerHTML = html;

  if (peakDay > 0) {
    meta.textContent = 'Día más alto: ' + peakDay + ' ' + MONTHS_ES_SHORT[month] +
      ' · ' + formatMoneyShort(maxAmt);
  } else {
    meta.textContent = 'Sin gastos aún';
  }
}

// ============================================================
// TREND (sparkline + month labels)
// ============================================================
function renderTrend(byMonth) {
  const svg = document.getElementById('trend-svg');
  const months = document.getElementById('trend-months');
  const avgEl  = document.getElementById('trend-avg');
  const tagEl  = document.getElementById('trend-tag');

  if (!byMonth.length) {
    svg.innerHTML = '';
    months.innerHTML = '';
    avgEl.textContent = 'Sin datos';
    tagEl.innerHTML = '';
    return;
  }

  const W = 360, H = 80, PAD_T = 8, PAD_B = 12;
  const vals = byMonth.map((m) => +m.total || 0);
  const min = Math.min(...vals);
  const max = Math.max(...vals);
  const range = max - min || 1;
  const stepX = vals.length > 1 ? W / (vals.length - 1) : W;
  const pts = vals.map((v, i) => {
    const x = vals.length > 1 ? i * stepX : W / 2;
    const y = PAD_T + (1 - (v - min) / range) * (H - PAD_T - PAD_B);
    return [x, y];
  });
  const d = pts.map((p, i) => (i === 0 ? 'M' : 'L') + p[0].toFixed(1) + ',' + p[1].toFixed(1)).join(' ');
  const area = d + ` L${W},${H} L0,${H} Z`;
  const lastIdx = pts.length - 1;
  const [lx, ly] = pts[lastIdx];

  svg.innerHTML = `
    <path d="${area}" fill="var(--rp-orange)" opacity="0.12"/>
    <path d="${d}" stroke="var(--rp-orange)" stroke-width="2.5" fill="none" stroke-linecap="round" stroke-linejoin="round"/>
    ${pts.map((p, i) => i === lastIdx
      ? `<circle cx="${p[0]}" cy="${p[1]}" r="4.5" fill="var(--rp-orange)"/>`
      : `<circle cx="${p[0]}" cy="${p[1]}" r="2.5" fill="var(--rp-orange)" opacity="0.5"/>`).join('')}
  `;

  // Month labels
  months.innerHTML = byMonth.map((m, i) => {
    const isCurrent = i === lastIdx;
    const lbl = monthLabel(m.month);
    return `<span class="${isCurrent ? 'is-current' : ''}">${escapeHtml(lbl)}</span>`;
  }).join('');

  const avg = vals.reduce((s, v) => s + v, 0) / vals.length;
  avgEl.textContent = 'Promedio ' + formatMoney(avg);

  // Trend tag — lowest so far / above avg / etc.
  const cur = vals[lastIdx];
  const prevs = vals.slice(0, lastIdx);
  if (prevs.length && cur < Math.min(...prevs)) {
    tagEl.innerHTML = '<span class="tag is-good"><span class="dot"></span>Mes más bajo del período</span>';
  } else if (prevs.length && cur > Math.max(...prevs)) {
    tagEl.innerHTML = '<span class="tag is-warn"><span class="dot"></span>Mes más alto del período</span>';
  } else if (cur < avg) {
    tagEl.innerHTML = '<span class="tag is-good"><span class="dot"></span>Bajo el promedio</span>';
  } else {
    tagEl.innerHTML = '';
  }
}

function monthLabel(s) {
  if (!s) return '';
  // accept "YYYY-MM" or "MM" or "ene"
  const m = String(s).match(/^(\d{4})-(\d{2})$/);
  if (m) return MONTHS_ES_SHORT[+m[2] - 1];
  const just = String(s).match(/^(\d{1,2})$/);
  if (just) return MONTHS_ES_SHORT[+just[1] - 1];
  return String(s).slice(0, 3).toLowerCase();
}

// ============================================================
// MSI (Meses sin Intereses)
// ============================================================
function renderInstallments(data) {
  const panel = document.getElementById('msi-panel');
  const summaryEl = document.getElementById('msi-summary');
  const list = document.getElementById('msi-list');
  const sub = document.getElementById('msi-sub');
  const groups = data.groups || [];

  if (!groups.length) {
    panel.style.display = 'none';
    return;
  }
  panel.style.display = '';

  sub.textContent = groups.length + ' ' +
    (groups.length === 1 ? 'compra activa' : 'compras activas') +
    ' a meses sin intereses';

  const debtFreeDate = formatDebtFreeDate(data.debt_free_date);
  summaryEl.innerHTML =
    '<div class="msi-stat">' +
      '<div class="msi-stat__label">Deuda restante</div>' +
      '<div class="msi-stat__value msi-stat__value--danger">' + formatMoney(data.total_remaining) + '</div>' +
    '</div>' +
    '<div class="msi-stat">' +
      '<div class="msi-stat__label">Libre de deuda</div>' +
      '<div class="msi-stat__value">' + escapeHtml(debtFreeDate) + '</div>' +
    '</div>';

  list.innerHTML = groups.map(function(g) {
    var color = catColor(g.category);
    var icon = catIcon(g.category);
    var pct = g.total_count > 0 ? (g.paid_count / g.total_count) * 100 : 0;
    var lastDate = formatMsiDate(g.last_payment_date);
    return '<div class="msi-row">' +
      '<span class="cat-chip" style="background:' + color + '1A;color:' + color + ';">' + icon + '</span>' +
      '<div>' +
        '<div class="msi-desc">' + escapeHtml(g.description) + '</div>' +
        '<div class="msi-progress-text">' + g.paid_count + ' de ' + g.total_count + ' pagos · ' + escapeHtml(lastDate) + '</div>' +
      '</div>' +
      '<div>' +
        '<div class="msi-bar"><div class="msi-bar__fill" style="width:' + pct.toFixed(1) + '%"></div></div>' +
        '<div class="msi-bar-label">' + formatMoney(g.per_installment) + '/mes · ' + pct.toFixed(0) + '% pagado</div>' +
      '</div>' +
      '<div class="msi-amount">' +
        '<div class="msi-amount__remaining">' + formatMoney(g.remaining_amount) + '</div>' +
        '<div class="msi-amount__total">de ' + formatMoney(g.total_amount) + '</div>' +
      '</div>' +
    '</div>';
  }).join('');
}

function formatDebtFreeDate(dateStr) {
  if (!dateStr) return '---';
  var m = String(dateStr).match(/^(\d{4})-(\d{2})-(\d{2})/);
  if (!m) return dateStr;
  var month = MONTHS_ES[parseInt(m[2], 10) - 1];
  return capitalize(month) + ' ' + parseInt(m[1], 10);
}

function formatMsiDate(dateStr) {
  if (!dateStr) return '';
  var m = String(dateStr).match(/^(\d{4})-(\d{2})/);
  if (!m) return dateStr;
  var month = MONTHS_ES_SHORT[parseInt(m[2], 10) - 1];
  return 'hasta ' + month + ' ' + m[1];
}

// ============================================================
// TRANSACTIONS
// ============================================================
function renderTransactions(expenses) {
  const list = document.getElementById('txn-list');
  const sub  = document.getElementById('txn-sub');

  if (!expenses.length) {
    sub.textContent = 'Sin movimientos';
    list.innerHTML = '<div class="txn-empty">Sin gastos en este período</div>';
    return;
  }
  sub.textContent = 'Últimos ' + Math.min(expenses.length, 10) + ' movimientos';

  list.innerHTML = expenses.slice(0, 10).map((e) => {
    const catName = e.category && e.category.name ? e.category.name : '—';
    const color   = catColor(catName);
    const { day, time } = parseDate(e.created_at);
    return `
      <div class="txn-row">
        <div class="txn-day-col">
          <div class="txn-day">${escapeHtml(day)}</div>
          <div class="txn-time">${escapeHtml(time)}</div>
        </div>
        <div class="txn-desc">${escapeHtml(e.description || '—')}</div>
        <div class="txn-cat-col">
          <span class="txn-cat-tag" style="background:${color}1A;color:${color};">
            <span class="dot" style="background:${color};"></span>${escapeHtml(catName)}
          </span>
        </div>
        <div class="txn-amount">${formatMoney(e.amount)}</div>
      </div>
    `;
  }).join('');
}

function parseDate(s) {
  if (!s) return { day: '—', time: '' };
  const norm = String(s).replace('T', ' ');
  const [d, t] = norm.split(' ');
  const md = (d || '').match(/^(\d{4})-(\d{2})-(\d{2})$/);
  if (!md) return { day: norm.slice(0, 10), time: t ? t.slice(0, 5) : '' };
  const date = new Date(+md[1], +md[2] - 1, +md[3]);
  const today = new Date(); today.setHours(0, 0, 0, 0);
  const diff = Math.round((today - date) / 86400000);
  let day;
  if (diff === 0)      day = 'Hoy';
  else if (diff === 1) day = 'Ayer';
  else if (diff < 7)   day = capitalize(DAYS_ES[date.getDay()]);
  else                 day = date.getDate() + ' ' + MONTHS_ES_SHORT[date.getMonth()];
  return { day, time: t ? t.slice(0, 5) : '' };
}

// ============================================================
// VIEW ROUTER
// ============================================================
function navigateTo(hash) {
  const viewName = (hash || '').replace('#', '') || 'resumen';
  document.querySelectorAll('[data-view]').forEach((v) => {
    v.style.display = v.dataset.view === viewName ? '' : 'none';
  });
  document.querySelectorAll('.side .ni').forEach((a) => {
    const linkHash = (a.getAttribute('href') || '').replace('#', '');
    a.classList.toggle('is-active', linkHash === viewName);
  });
  state.view = viewName;
  if (viewName === 'gastos') loadGastosView();
  if (viewName === 'categorias') loadCategoriasView();
  if (viewName === 'presupuestos') loadPresupuestosView();
}

// ============================================================
// ADD EXPENSE MODAL
// ============================================================
function closeExpenseModal() {
  document.getElementById('expense-modal').style.display = 'none';
}

async function handleExpenseSubmit(e) {
  e.preventDefault();
  const form = e.target;
  const body = {
    user: state.user,
    amount: parseFloat(form.amount.value),
    description: form.description.value.trim(),
    category: { name: form.category.value },
  };
  try {
    const res = await fetch('/api/expenses', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(body),
    });
    if (!res.ok) {
      const msg = await res.text();
      alert('Error: ' + msg);
      return;
    }
    form.reset();
    closeExpenseModal();
    loadUsers();
    loadDashboard();
  } catch (err) {
    alert('Error de red: ' + err.message);
  }
}

// ============================================================
// EDIT EXPENSE MODAL
// ============================================================
function openEditModal(expense) {
  const form = document.getElementById('edit-form');
  form.expense_id.value = expense.id;
  form.description.value = expense.description || '';
  form.category.value = expense.category && expense.category.name ? expense.category.name : '';
  const dateStr = String(expense.created_at || '').replace('T', ' ');
  const match = dateStr.match(/^(\d{4}-\d{2}-\d{2})/);
  form.date.value = match ? match[1] : '';
  document.getElementById('edit-modal').style.display = '';
}

function closeEditModal() {
  document.getElementById('edit-modal').style.display = 'none';
}

async function handleEditSubmit(e) {
  e.preventDefault();
  const form = e.target;
  const id = form.expense_id.value;
  const body = {
    description: form.description.value.trim(),
    category: { name: form.category.value },
    created_at: form.date.value + ' 00:00:00',
  };
  try {
    const res = await fetch('/api/expenses/' + id, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(body),
    });
    if (!res.ok) {
      const msg = await res.text();
      alert('Error: ' + msg);
      return;
    }
    closeEditModal();
    loadGastosView();
    loadDashboard();
  } catch (err) {
    alert('Error de red: ' + err.message);
  }
}

// ============================================================
// VIEW: GASTOS (full expense list with delete)
// ============================================================
async function loadGastosView() {
  const params = new URLSearchParams({ period: state.period });
  if (state.category) params.set('category', state.category);
  if (state.user) params.set('user', state.user);
  appendMonthParams(params);
  try {
    const res = await fetch('/api/expenses?' + params);
    const expenses = await res.json();
    renderGastosView(expenses || []);
  } catch (err) {
    console.error('Error cargando gastos:', err);
  }
}

function renderGastosView(expenses) {
  const list = document.getElementById('gastos-list');
  const sub  = document.getElementById('gastos-sub');
  if (!expenses.length) {
    sub.textContent = 'Sin movimientos';
    list.innerHTML = '<div class="txn-empty">Sin gastos en este período</div>';
    return;
  }
  sub.textContent = expenses.length + ' ' + (expenses.length === 1 ? 'gasto' : 'gastos');
  list.innerHTML = expenses.map((e) => {
    const catName = e.category && e.category.name ? e.category.name : '—';
    const color   = catColor(catName);
    const { day, time } = parseDate(e.created_at);
    return `
      <div class="txn-row">
        <div class="txn-day-col">
          <div class="txn-day">${escapeHtml(day)}</div>
          <div class="txn-time">${escapeHtml(time)}</div>
        </div>
        <div class="txn-desc">${escapeHtml(e.description || '—')}</div>
        <div class="txn-cat-col">
          <span class="txn-cat-tag" style="background:${color}1A;color:${color};">
            <span class="dot" style="background:${color};"></span>${escapeHtml(catName)}
          </span>
        </div>
        <div class="txn-amount">${formatMoney(e.amount)}</div>
        <button class="btn-edit" data-expense='${JSON.stringify(e).replace(/'/g, "&#39;")}' title="Editar">&#9998;</button>
        <button class="btn-delete" data-id="${e.id}" title="Eliminar">&times;</button>
      </div>
    `;
  }).join('');

  list.querySelectorAll('.btn-edit').forEach((btn) => {
    btn.addEventListener('click', () => {
      const expense = JSON.parse(btn.dataset.expense);
      openEditModal(expense);
    });
  });

  list.querySelectorAll('.btn-delete').forEach((btn) => {
    btn.addEventListener('click', async () => {
      if (!confirm('¿Eliminar este gasto?')) return;
      try {
        const res = await fetch('/api/expenses/' + btn.dataset.id, { method: 'DELETE' });
        if (res.ok) {
          loadGastosView();
          loadDashboard();
        } else {
          alert('Error eliminando gasto');
        }
      } catch (err) {
        alert('Error de red: ' + err.message);
      }
    });
  });
}

// ============================================================
// VIEW: CATEGORÍAS (detailed category breakdown)
// ============================================================
async function loadCategoriasView() {
  const params = new URLSearchParams({ period: state.period });
  if (state.category) params.set('category', state.category);
  if (state.user) params.set('user', state.user);
  appendMonthParams(params);
  try {
    const res = await fetch('/api/dashboard/summary?' + params);
    const summary = await res.json();
    renderCategoriasView(summary || {});
  } catch (err) {
    console.error('Error cargando categorías:', err);
  }
}

function renderCategoriasView(summary) {
  const byCategory = summary.by_category || [];
  const budgets    = summary.budget_comparison || [];
  const total      = +summary.total_amount || 0;
  const list = document.getElementById('categorias-list');
  const sub  = document.getElementById('categorias-sub');
  if (!byCategory.length) {
    sub.textContent = 'Sin datos';
    list.innerHTML = '<div class="txn-empty">Sin gastos en este período</div>';
    return;
  }
  sub.textContent = byCategory.length + ' categorías activas';
  renderCategoryList(list, byCategory, budgets, total);
}

function renderCategoryList(container, byCategory, budgets, total) {
  const budgetMap = {};
  budgets.forEach((b) => { budgetMap[(b.category || '').toLowerCase()] = b; });
  const sorted = [...byCategory].sort((a, b) => b.total - a.total);
  const maxTotal = sorted[0].total || 1;

  container.innerHTML = sorted.map((c) => {
    const color = catColor(c.category);
    const icon  = catIcon(c.category);
    const budget = budgetMap[c.category.toLowerCase()];
    const sharePct = total > 0 ? (c.total / total) * 100 : 0;
    let barCls = 'is-ok';
    let barPct = (c.total / maxTotal) * 100;
    let barColorStyle = 'background:' + color + ';';
    let metaHtml = '';
    if (budget) {
      const pct  = budget.percentage ?? (budget.spent / budget.budgeted) * 100;
      const over = pct >= 100;
      const warn = pct >= 80;
      barCls = over ? 'is-over' : warn ? 'is-warn' : 'is-ok';
      barPct = Math.min(pct, 100);
      barColorStyle = '';
      metaHtml = formatMoney(c.total) + ' de ' + formatMoney(budget.budgeted) +
        (over ? ' <span class="over">· sobrepasado</span>' : '');
    } else {
      metaHtml = sharePct.toFixed(0) + '% del total';
    }
    const subHtml = budget
      ? ((c.total / (budget.budgeted || 1)) * 100).toFixed(0) + '% de su presupuesto'
      : (c.count ? c.count + ' ' + (c.count === 1 ? 'gasto' : 'gastos') : ' ');
    return `
      <a class="cat-row" href="#cat-${escapeHtml(c.category)}">
        <span class="cat-chip" style="background:${color}1A;color:${color};">${icon}</span>
        <div>
          <div class="cat-name">${escapeHtml(c.category)}</div>
          <div class="cat-sub">${escapeHtml(subHtml)}</div>
        </div>
        <div>
          <div class="cat-bar"><div class="cat-bar__fill ${barCls}" style="width:${barPct.toFixed(1)}%;${barColorStyle}"></div></div>
          <div class="cat-meta">${metaHtml}</div>
        </div>
        <div class="cat-amount">
          <div class="cat-amount__value">${formatMoney(c.total)}</div>
          <div class="cat-amount__pct">${sharePct.toFixed(0)}%</div>
        </div>
      </a>
    `;
  }).join('');
}

// ============================================================
// VIEW: PRESUPUESTOS (budget management)
// ============================================================
async function loadPresupuestosView() {
  const container = document.getElementById('budget-form-container');
  const sub = document.getElementById('presupuestos-sub');
  if (!state.user) {
    sub.textContent = 'Define un usuario primero';
    container.innerHTML = '<div class="txn-empty">Ingresa un usuario en la barra superior para gestionar presupuestos</div>';
    return;
  }
  sub.textContent = 'Presupuestos de ' + state.user;
  try {
    const res = await fetch('/api/budgets?user=' + encodeURIComponent(state.user));
    const budgets = await res.json();
    renderPresupuestosView(budgets || []);
  } catch (err) {
    console.error('Error cargando presupuestos:', err);
  }
}

function renderPresupuestosView(budgets) {
  const budgetMap = {};
  budgets.forEach((b) => { budgetMap[b.category] = b; });
  const container = document.getElementById('budget-form-container');
  const categories = Object.keys(CATEGORY_COLORS);

  container.innerHTML = '<div class="budget-grid">' + categories.map((cat) => {
    const existing = budgetMap[cat];
    const color = catColor(cat);
    const icon  = catIcon(cat);
    return `
      <div class="budget-row">
        <span class="cat-chip" style="background:${color}1A;color:${color};">${icon}</span>
        <div class="budget-row__name">${capitalize(cat)}</div>
        <input type="number" class="budget-input" data-category="${escapeHtml(cat)}"
               value="${existing ? existing.amount : ''}"
               placeholder="Sin límite" min="0" step="100">
        <button class="btn btn--tertiary budget-save" data-category="${escapeHtml(cat)}">Guardar</button>
      </div>
    `;
  }).join('') + '</div>';

  container.querySelectorAll('.budget-save').forEach((btn) => {
    btn.addEventListener('click', async () => {
      const cat = btn.dataset.category;
      const input = container.querySelector('input[data-category="' + cat + '"]');
      const amount = parseFloat(input.value);
      if (!amount || amount <= 0) { alert('Monto inválido'); return; }
      try {
        const res = await fetch('/api/budgets', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ user: state.user, category: cat, amount: amount }),
        });
        if (!res.ok) {
          alert('Error: ' + await res.text());
          return;
        }
        btn.textContent = '✓';
        setTimeout(() => { btn.textContent = 'Guardar'; }, 1200);
        loadDashboard();
      } catch (err) {
        alert('Error de red: ' + err.message);
      }
    });
  });
}
