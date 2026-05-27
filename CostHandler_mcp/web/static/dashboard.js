let currentPeriod = 'month';
let currentCategory = '';

let categoryChart = null;
let dailyChart = null;
let monthlyChart = null;

const CATEGORY_COLORS = {
    supermercado:    '#4CAF50',
    restaurantes:    '#FF9800',
    vivienda:        '#2196F3',
    servicios:       '#9C27B0',
    transporte:      '#F44336',
    salud:           '#00BCD4',
    familia:         '#E91E63',
    suscripciones:   '#607D8B',
    entretenimiento: '#FFC107',
    compras:         '#8BC34A',
    ahorro:          '#3F51B5',
    otros:           '#795548',
};

document.addEventListener('DOMContentLoaded', function() {
    document.getElementById('period-select').addEventListener('change', onFilterChange);
    document.getElementById('category-select').addEventListener('change', onFilterChange);
    document.getElementById('user-input').addEventListener('change', onFilterChange);
    loadDashboard();
});

function onFilterChange() {
    currentPeriod = document.getElementById('period-select').value;
    currentCategory = document.getElementById('category-select').value;
    loadDashboard();
}

async function loadDashboard() {
    const params = new URLSearchParams({ period: currentPeriod });
    if (currentCategory) params.set('category', currentCategory);
    var user = document.getElementById('user-input').value.trim();
    if (user) params.set('user', user);

    try {
        const [summaryRes, expensesRes] = await Promise.all([
            fetch('/api/dashboard/summary?' + params),
            fetch('/api/expenses?' + params)
        ]);

        const summary = await summaryRes.json();
        const expenses = await expensesRes.json();

        updateCards(summary);
        renderCategoryChart(summary.by_category || []);
        renderDailyChart(summary.by_day || []);
        renderMonthlyChart(summary.by_month || []);
        renderBudgetBars(summary.budget_comparison || []);
        renderExpenseTable(expenses || []);
    } catch (err) {
        console.error('Error cargando dashboard:', err);
    }
}

function updateCards(data) {
    document.getElementById('total-amount').textContent = formatMoney(data.total_amount);
    document.getElementById('top-category').textContent = data.top_category || '-';
    document.getElementById('daily-avg').textContent = formatMoney(data.daily_average);

    var el = document.getElementById('prev-comparison');
    if (data.prev_total > 0) {
        var diff = data.total_amount - data.prev_total;
        var pct = ((diff / data.prev_total) * 100).toFixed(0);
        var sign = diff >= 0 ? '+' : '';
        el.textContent = sign + pct + '%';
        el.className = 'card-value ' + (diff >= 0 ? 'negative' : 'positive');
    } else {
        el.textContent = '-';
        el.className = 'card-value';
    }
}

function renderCategoryChart(byCategory) {
    if (categoryChart) categoryChart.destroy();
    var ctx = document.getElementById('category-chart').getContext('2d');

    var colors = byCategory.map(function(c) {
        return CATEGORY_COLORS[c.category] || '#999';
    });

    categoryChart = new Chart(ctx, {
        type: 'doughnut',
        data: {
            labels: byCategory.map(function(c) { return c.category; }),
            datasets: [{
                data: byCategory.map(function(c) { return c.total; }),
                backgroundColor: colors
            }]
        },
        options: {
            responsive: true,
            maintainAspectRatio: false,
            plugins: {
                legend: { position: 'bottom', labels: { boxWidth: 12 } }
            }
        }
    });
}

function renderDailyChart(byDay) {
    if (dailyChart) dailyChart.destroy();
    var ctx = document.getElementById('daily-chart').getContext('2d');

    dailyChart = new Chart(ctx, {
        type: 'bar',
        data: {
            labels: byDay.map(function(d) { return d.date; }),
            datasets: [{
                label: 'Gasto diario',
                data: byDay.map(function(d) { return d.total; }),
                backgroundColor: '#2196F3'
            }]
        },
        options: {
            responsive: true,
            maintainAspectRatio: false,
            plugins: { legend: { display: false } },
            scales: { y: { beginAtZero: true } }
        }
    });
}

function renderMonthlyChart(byMonth) {
    if (monthlyChart) monthlyChart.destroy();
    var ctx = document.getElementById('monthly-chart').getContext('2d');

    monthlyChart = new Chart(ctx, {
        type: 'line',
        data: {
            labels: byMonth.map(function(m) { return m.month; }),
            datasets: [{
                label: 'Tendencia mensual',
                data: byMonth.map(function(m) { return m.total; }),
                borderColor: '#4CAF50',
                backgroundColor: 'rgba(76, 175, 80, 0.1)',
                fill: true,
                tension: 0.3
            }]
        },
        options: {
            responsive: true,
            maintainAspectRatio: false,
            plugins: { legend: { display: false } },
            scales: { y: { beginAtZero: true } }
        }
    });
}

function renderExpenseTable(expenses) {
    var tbody = document.querySelector('#expense-list tbody');
    if (expenses.length === 0) {
        tbody.innerHTML = '<tr><td colspan="4" class="empty">Sin gastos en este período</td></tr>';
        return;
    }
    tbody.innerHTML = expenses.map(function(e) {
        var date = e.created_at ? e.created_at.split(' ')[0] : '';
        return '<tr>' +
            '<td>' + date + '</td>' +
            '<td>' + escapeHtml(e.description) + '</td>' +
            '<td>' + escapeHtml(e.category.name) + '</td>' +
            '<td>' + formatMoney(e.amount) + '</td>' +
            '</tr>';
    }).join('');
}

function renderBudgetBars(comparison) {
    var section = document.getElementById('budget-section');
    var container = document.getElementById('budget-bars');

    if (!comparison || comparison.length === 0) {
        section.style.display = 'none';
        return;
    }

    section.style.display = '';
    container.innerHTML = comparison.map(function(c) {
        var pct = Math.min(c.percentage, 150);
        var widthPct = Math.min(c.percentage, 100);
        var cls = 'ok';
        if (c.percentage >= 100) cls = 'over';
        else if (c.percentage >= 80) cls = 'warning';

        return '<div class="budget-row">' +
            '<div class="budget-label">' +
                '<span class="budget-category">' + escapeHtml(c.category) + '</span>' +
                '<span class="budget-amounts">' + formatMoney(c.spent) + ' / ' + formatMoney(c.budgeted) + ' (' + c.percentage.toFixed(0) + '%)</span>' +
            '</div>' +
            '<div class="budget-bar">' +
                '<div class="budget-bar-fill ' + cls + '" style="width:' + widthPct + '%"></div>' +
            '</div>' +
        '</div>';
    }).join('');
}

function formatMoney(amount) {
    return '$' + (amount || 0).toFixed(2).replace(/\B(?=(\d{3})+(?!\d))/g, ',');
}

function escapeHtml(str) {
    var div = document.createElement('div');
    div.textContent = str || '';
    return div.innerHTML;
}
