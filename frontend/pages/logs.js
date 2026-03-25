registerPage('#logs', function(app) {
    const page = document.createElement('div');
    page.className = 'page';

    // Header row: title + clear button
    const header = document.createElement('div');
    header.style.display = 'flex';
    header.style.justifyContent = 'space-between';
    header.style.alignItems = 'center';
    header.style.marginBottom = '24px';

    const title = document.createElement('h1');
    title.textContent = 'Logs';
    title.style.margin = '0';
    header.appendChild(title);

    const clearBtn = document.createElement('button');
    clearBtn.className = 'btn btn-danger';
    clearBtn.textContent = 'Effacer les logs';
    clearBtn.onclick = async () => {
        try {
            await window.go.main.App.ClearLogs();
            showToast('Logs effacés', 'success');
            refreshLogs();
        } catch (e) {
            showToast('Erreur lors de l\'effacement des logs: ' + e, 'error');
        }
    };
    header.appendChild(clearBtn);

    page.appendChild(header);

    // Table container
    const tableWrap = document.createElement('div');
    tableWrap.className = 'table-wrap';

    const table = document.createElement('table');

    const thead = document.createElement('thead');
    const headerRow = document.createElement('tr');
    ['Heure', 'Niveau', 'Message'].forEach(h => {
        const th = document.createElement('th');
        th.textContent = h;
        headerRow.appendChild(th);
    });
    thead.appendChild(headerRow);
    table.appendChild(thead);

    const tbody = document.createElement('tbody');
    tbody.id = 'logs-tbody';
    table.appendChild(tbody);

    tableWrap.appendChild(table);
    page.appendChild(tableWrap);

    app.appendChild(page);

    // Level color map
    const levelColors = {
        'info':  '#3b82f6',
        'warn':  '#f97316',
        'error': '#ef4444'
    };

    function formatTime(timestamp) {
        const d = new Date(timestamp);
        const hh = String(d.getHours()).padStart(2, '0');
        const mm = String(d.getMinutes()).padStart(2, '0');
        const ss = String(d.getSeconds()).padStart(2, '0');
        return `${hh}:${mm}:${ss}`;
    }

    async function refreshLogs() {
        try {
            const logs = await window.go.main.App.GetLogs();
            tbody.innerHTML = '';

            if (!logs || logs.length === 0) {
                const tr = document.createElement('tr');
                const td = document.createElement('td');
                td.colSpan = 3;
                td.style.textAlign = 'center';
                td.style.padding = '20px';
                td.style.color = 'var(--color-text-muted)';
                td.textContent = 'Aucun log disponible';
                tr.appendChild(td);
                tbody.appendChild(tr);
                return;
            }

            // Show newest first (reverse order)
            const reversed = logs.slice().reverse();
            reversed.forEach(entry => {
                const tr = document.createElement('tr');

                const timeTd = document.createElement('td');
                timeTd.style.whiteSpace = 'nowrap';
                timeTd.style.fontFamily = 'monospace';
                timeTd.style.fontSize = '13px';
                timeTd.textContent = formatTime(entry.timestamp);
                tr.appendChild(timeTd);

                const levelTd = document.createElement('td');
                levelTd.style.whiteSpace = 'nowrap';
                const level = (entry.level || 'info').toLowerCase();
                const color = levelColors[level] || '#9ca3af';
                const badge = document.createElement('span');
                badge.style.display = 'inline-block';
                badge.style.padding = '2px 8px';
                badge.style.borderRadius = '4px';
                badge.style.fontSize = '12px';
                badge.style.fontWeight = '600';
                badge.style.backgroundColor = color + '20';
                badge.style.color = color;
                badge.textContent = level.toUpperCase();
                levelTd.appendChild(badge);
                tr.appendChild(levelTd);

                const msgTd = document.createElement('td');
                msgTd.style.fontFamily = 'monospace';
                msgTd.style.fontSize = '13px';
                msgTd.textContent = entry.message;
                tr.appendChild(msgTd);

                tbody.appendChild(tr);
            });

            // Scroll to top after refresh
            tableWrap.scrollTop = 0;
        } catch (e) {
            showToast('Erreur lors du chargement des logs: ' + e, 'error');
        }
    }

    // Initial load
    refreshLogs();

    // Auto-refresh every 5 seconds
    const intervalId = setInterval(refreshLogs, 5000);

    // Clean up interval when page is replaced
    const observer = new MutationObserver(() => {
        if (!document.body.contains(page)) {
            clearInterval(intervalId);
            observer.disconnect();
        }
    });
    observer.observe(document.body, { childList: true, subtree: true });
});
