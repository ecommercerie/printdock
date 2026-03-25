registerPage('#history', function(app) {
    const page = document.createElement('div');
    page.className = 'page';

    // Title
    const title = document.createElement('h1');
    title.textContent = 'Historique';
    page.appendChild(title);

    // Filter Bar
    const filterCard = document.createElement('div');
    filterCard.className = 'card mb-6';
    filterCard.style.marginBottom = '24px';

    const filterGrid = document.createElement('div');
    filterGrid.style.display = 'grid';
    filterGrid.style.gridTemplateColumns = '1fr 1fr 1fr auto';
    filterGrid.style.gap = '12px';
    filterGrid.style.alignItems = 'flex-end';

    // Status dropdown
    const statusGroup = document.createElement('div');
    statusGroup.className = 'form-group';
    const statusLabel = document.createElement('label');
    statusLabel.textContent = 'Statut';
    const statusSelect = document.createElement('select');
    statusSelect.id = 'filter-status';
    const statusOptions = [
        { text: 'Tous', value: '' },
        { text: 'Imprimé', value: 'printed' },
        { text: 'Échoué', value: 'failed' },
        { text: 'Inconnu', value: 'unknown' },
        { text: 'Ignoré', value: 'ignored' }
    ];
    statusOptions.forEach(opt => {
        const option = document.createElement('option');
        option.value = opt.value;
        option.textContent = opt.text;
        statusSelect.appendChild(option);
    });
    statusGroup.appendChild(statusLabel);
    statusGroup.appendChild(statusSelect);

    // Date from
    const dateFromGroup = document.createElement('div');
    dateFromGroup.className = 'form-group';
    const dateFromLabel = document.createElement('label');
    dateFromLabel.textContent = 'Du';
    const dateFromInput = document.createElement('input');
    dateFromInput.type = 'date';
    dateFromInput.id = 'filter-date-from';
    dateFromGroup.appendChild(dateFromLabel);
    dateFromGroup.appendChild(dateFromInput);

    // Date to
    const dateToGroup = document.createElement('div');
    dateToGroup.className = 'form-group';
    const dateToLabel = document.createElement('label');
    dateToLabel.textContent = 'Au';
    const dateToInput = document.createElement('input');
    dateToInput.type = 'date';
    dateToInput.id = 'filter-date-to';
    dateToGroup.appendChild(dateToLabel);
    dateToGroup.appendChild(dateToInput);

    // Filter button
    const filterBtn = document.createElement('button');
    filterBtn.className = 'btn btn-primary';
    filterBtn.textContent = 'Filtrer';
    filterBtn.style.height = 'fit-content';

    filterGrid.appendChild(statusGroup);
    filterGrid.appendChild(dateFromGroup);
    filterGrid.appendChild(dateToGroup);
    filterGrid.appendChild(filterBtn);
    filterCard.appendChild(filterGrid);
    page.appendChild(filterCard);

    // Preview panel
    const previewOverlay = document.createElement('div');
    previewOverlay.id = 'preview-overlay';
    previewOverlay.style.display = 'none';
    previewOverlay.style.position = 'fixed';
    previewOverlay.style.top = '0';
    previewOverlay.style.left = '0';
    previewOverlay.style.right = '0';
    previewOverlay.style.bottom = '0';
    previewOverlay.style.backgroundColor = 'rgba(0,0,0,0.6)';
    previewOverlay.style.zIndex = '200';
    previewOverlay.style.display = 'none';
    previewOverlay.style.alignItems = 'center';
    previewOverlay.style.justifyContent = 'center';
    previewOverlay.onclick = function(e) { if (e.target === previewOverlay) closePreview(); };

    const previewPanel = document.createElement('div');
    previewPanel.style.background = 'var(--color-surface)';
    previewPanel.style.borderRadius = '8px';
    previewPanel.style.width = '80%';
    previewPanel.style.height = '85%';
    previewPanel.style.maxWidth = '900px';
    previewPanel.style.display = 'flex';
    previewPanel.style.flexDirection = 'column';
    previewPanel.style.boxShadow = '0 20px 60px rgba(0,0,0,0.3)';
    previewPanel.style.overflow = 'hidden';

    const previewHeader = document.createElement('div');
    previewHeader.style.display = 'flex';
    previewHeader.style.justifyContent = 'space-between';
    previewHeader.style.alignItems = 'center';
    previewHeader.style.padding = '12px 16px';
    previewHeader.style.borderBottom = '1px solid var(--color-border)';

    const previewTitle = document.createElement('span');
    previewTitle.id = 'preview-title';
    previewTitle.style.fontWeight = '600';
    previewTitle.style.fontSize = '14px';
    previewHeader.appendChild(previewTitle);

    const previewCloseBtn = document.createElement('button');
    previewCloseBtn.className = 'btn btn-sm btn-secondary';
    previewCloseBtn.textContent = 'Fermer';
    previewCloseBtn.onclick = closePreview;
    previewHeader.appendChild(previewCloseBtn);

    previewPanel.appendChild(previewHeader);

    const previewFrame = document.createElement('iframe');
    previewFrame.id = 'preview-frame';
    previewFrame.style.flex = '1';
    previewFrame.style.border = 'none';
    previewFrame.style.width = '100%';
    previewPanel.appendChild(previewFrame);

    previewOverlay.appendChild(previewPanel);
    page.appendChild(previewOverlay);

    function closePreview() {
        previewOverlay.style.display = 'none';
        previewFrame.src = 'about:blank';
    }

    async function openPreview(id, filename) {
        previewTitle.textContent = filename;
        previewOverlay.style.display = 'flex';
        previewFrame.src = 'about:blank';

        try {
            const b64 = await window.go.main.App.GetDocumentBase64(id);
            previewFrame.src = 'data:application/pdf;base64,' + b64;
        } catch (e) {
            previewFrame.src = 'about:blank';
            showToast('Impossible de charger le document: ' + e, 'error');
            closePreview();
        }
    }

    // Table container
    const tableWrap = document.createElement('div');
    tableWrap.className = 'table-wrap';
    const table = document.createElement('table');

    // Table header
    const thead = document.createElement('thead');
    const headerRow = document.createElement('tr');
    // Select-all checkbox header cell
    const selectAllTh = document.createElement('th');
    selectAllTh.style.width = '36px';
    const selectAllCheckbox = document.createElement('input');
    selectAllCheckbox.type = 'checkbox';
    selectAllCheckbox.title = 'Tout sélectionner';
    selectAllTh.appendChild(selectAllCheckbox);
    headerRow.appendChild(selectAllTh);

    const headers = ['Date', 'Fichier', 'Imprimante', 'Règle', 'Statut', 'Actions'];
    headers.forEach(h => {
        const th = document.createElement('th');
        th.textContent = h;
        headerRow.appendChild(th);
    });
    thead.appendChild(headerRow);
    table.appendChild(thead);

    // Table body
    const tbody = document.createElement('tbody');
    table.appendChild(tbody);
    tableWrap.appendChild(table);
    page.appendChild(tableWrap);

    // Pagination
    const paginationDiv = document.createElement('div');
    paginationDiv.style.display = 'flex';
    paginationDiv.style.justifyContent = 'space-between';
    paginationDiv.style.alignItems = 'center';
    paginationDiv.style.marginTop = '24px';

    const pageInfoSpan = document.createElement('span');
    pageInfoSpan.style.fontSize = '13px';
    pageInfoSpan.style.color = 'var(--color-text-muted)';

    const paginationButtons = document.createElement('div');
    paginationButtons.style.display = 'flex';
    paginationButtons.style.gap = '8px';

    const prevBtn = document.createElement('button');
    prevBtn.className = 'btn btn-secondary';
    prevBtn.textContent = 'Précédent';
    prevBtn.id = 'prev-btn';

    const nextBtn = document.createElement('button');
    nextBtn.className = 'btn btn-secondary';
    nextBtn.textContent = 'Suivant';
    nextBtn.id = 'next-btn';

    paginationButtons.appendChild(prevBtn);
    paginationButtons.appendChild(nextBtn);
    paginationDiv.appendChild(pageInfoSpan);
    paginationDiv.appendChild(paginationButtons);
    page.appendChild(paginationDiv);

    // Batch action bar
    const batchBar = document.createElement('div');
    batchBar.style.display = 'none';
    batchBar.style.position = 'fixed';
    batchBar.style.bottom = '0';
    batchBar.style.left = '220px'; // account for sidebar width
    batchBar.style.right = '0';
    batchBar.style.padding = '12px 24px';
    batchBar.style.backgroundColor = 'var(--color-surface)';
    batchBar.style.borderTop = '1px solid var(--color-border)';
    batchBar.style.boxShadow = '0 -4px 16px rgba(0,0,0,0.15)';
    batchBar.style.display = 'none';
    batchBar.style.alignItems = 'center';
    batchBar.style.gap = '16px';
    batchBar.style.zIndex = '100';

    const batchCountLabel = document.createElement('span');
    batchCountLabel.style.fontSize = '14px';
    batchCountLabel.style.fontWeight = '500';
    batchCountLabel.style.color = 'var(--color-text)';

    const batchReprintBtn = document.createElement('button');
    batchReprintBtn.className = 'btn btn-primary';
    batchReprintBtn.textContent = 'Réimprimer la sélection';

    const batchCancelBtn = document.createElement('button');
    batchCancelBtn.className = 'btn btn-secondary';
    batchCancelBtn.textContent = 'Annuler';

    batchBar.appendChild(batchCountLabel);
    batchBar.appendChild(batchReprintBtn);
    batchBar.appendChild(batchCancelBtn);
    document.body.appendChild(batchBar);

    // State
    let currentOffset = 0;
    const pageSize = 50;
    let selectedIds = new Set();

    // Helper: update batch bar visibility and label
    function updateBatchBar() {
        const count = selectedIds.size;
        if (count > 0) {
            batchCountLabel.textContent = count + ' sélectionné(s)';
            batchBar.style.display = 'flex';
        } else {
            batchBar.style.display = 'none';
        }
        // Sync select-all checkbox state
        const rowCheckboxes = tbody.querySelectorAll('input[type="checkbox"]');
        if (rowCheckboxes.length > 0) {
            const allChecked = [...rowCheckboxes].every(cb => cb.checked);
            const anyChecked = [...rowCheckboxes].some(cb => cb.checked);
            selectAllCheckbox.checked = allChecked;
            selectAllCheckbox.indeterminate = anyChecked && !allChecked;
        } else {
            selectAllCheckbox.checked = false;
            selectAllCheckbox.indeterminate = false;
        }
    }

    // Select-all handler
    selectAllCheckbox.addEventListener('change', () => {
        const rowCheckboxes = tbody.querySelectorAll('input[type="checkbox"]');
        rowCheckboxes.forEach(cb => {
            cb.checked = selectAllCheckbox.checked;
            const id = parseInt(cb.dataset.id, 10);
            if (selectAllCheckbox.checked) {
                selectedIds.add(id);
            } else {
                selectedIds.delete(id);
            }
        });
        updateBatchBar();
    });

    // Batch reprint handler
    batchReprintBtn.addEventListener('click', async () => {
        if (selectedIds.size === 0) return;
        batchReprintBtn.disabled = true;
        const ids = [...selectedIds];
        try {
            await window.go.main.App.BatchReprint(ids);
            const count = ids.length;
            selectedIds.clear();
            updateBatchBar();
            showToast(count + ' document(s) en cours de réimpression...', 'success');
        } catch (e) {
            showToast('Erreur lors de la réimpression: ' + e, 'error');
        }
        batchReprintBtn.disabled = false;
    });

    // Batch cancel handler
    batchCancelBtn.addEventListener('click', () => {
        selectedIds.clear();
        tbody.querySelectorAll('input[type="checkbox"]').forEach(cb => { cb.checked = false; });
        updateBatchBar();
    });

    // Helper function to format date
    function formatDate(dateString) {
        const date = new Date(dateString);
        const day = String(date.getDate()).padStart(2, '0');
        const month = String(date.getMonth() + 1).padStart(2, '0');
        const year = date.getFullYear();
        const hours = String(date.getHours()).padStart(2, '0');
        const minutes = String(date.getMinutes()).padStart(2, '0');
        return `${day}/${month}/${year} ${hours}:${minutes}`;
    }

    // Helper function to get status badge
    function getStatusBadge(status) {
        const badges = {
            'printed': { text: 'Imprimé', color: '#22c55e' },
            'failed': { text: 'Échoué', color: '#ef4444' },
            'unknown': { text: 'Inconnu', color: '#9ca3af' },
            'ignored': { text: 'Ignoré', color: '#9ca3af' }
        };
        const badge = badges[status] || { text: status, color: '#9ca3af' };
        const span = document.createElement('span');
        span.style.display = 'inline-flex';
        span.style.alignItems = 'center';
        span.style.gap = '6px';
        span.style.padding = '4px 10px';
        span.style.borderRadius = '4px';
        span.style.fontSize = '12px';
        span.style.fontWeight = '600';
        span.style.backgroundColor = badge.color + '20';
        span.style.color = badge.color;
        span.textContent = badge.text;
        return span;
    }

    // Helper function to load data
    async function loadData(offset) {
        const status = statusSelect.value;
        const dateFrom = dateFromInput.value;
        const dateTo = dateToInput.value;

        const filter = {
            status: status || undefined,
            dateFrom: dateFrom || undefined,
            dateTo: dateTo || undefined,
            offset: offset,
            limit: pageSize
        };

        try {
            const data = await window.go.main.App.GetHistory(filter);
            if (!data) {
                tbody.innerHTML = '<tr><td colspan="7" style="text-align: center; padding: 20px;">Aucun résultat</td></tr>';
                pageInfoSpan.textContent = 'Aucun résultat';
                prevBtn.disabled = true;
                nextBtn.disabled = true;
                return;
            }

            tbody.innerHTML = '';
            selectedIds.clear();
            updateBatchBar();
            data.forEach(row => {
                const tr = document.createElement('tr');

                // Checkbox
                const checkTd = document.createElement('td');
                checkTd.style.width = '36px';
                const rowCheckbox = document.createElement('input');
                rowCheckbox.type = 'checkbox';
                rowCheckbox.dataset.id = row.id;
                if (selectedIds.has(row.id)) rowCheckbox.checked = true;
                rowCheckbox.addEventListener('change', () => {
                    if (rowCheckbox.checked) {
                        selectedIds.add(row.id);
                    } else {
                        selectedIds.delete(row.id);
                    }
                    updateBatchBar();
                });
                checkTd.appendChild(rowCheckbox);
                tr.appendChild(checkTd);

                // Date
                const dateTd = document.createElement('td');
                dateTd.textContent = formatDate(row.createdAt);
                tr.appendChild(dateTd);

                // Filename
                const fileTd = document.createElement('td');
                fileTd.style.maxWidth = '200px';
                fileTd.style.overflow = 'hidden';
                fileTd.style.textOverflow = 'ellipsis';
                fileTd.title = row.filename;
                fileTd.textContent = row.filename;
                tr.appendChild(fileTd);

                // Printer
                const printerTd = document.createElement('td');
                printerTd.textContent = row.printer || '-';
                tr.appendChild(printerTd);

                // Rule
                const ruleTd = document.createElement('td');
                ruleTd.textContent = row.ruleName || '-';
                tr.appendChild(ruleTd);

                // Status
                const statusTd = document.createElement('td');
                const statusBadge = getStatusBadge(row.status);

                // Add tooltip for failed status
                if (row.status === 'failed' && row.errorMessage) {
                    const container = document.createElement('div');
                    container.style.position = 'relative';
                    container.style.display = 'inline-block';

                    const tooltip = document.createElement('div');
                    tooltip.style.position = 'absolute';
                    tooltip.style.bottom = '100%';
                    tooltip.style.left = '0';
                    tooltip.style.backgroundColor = '#1f2937';
                    tooltip.style.color = '#ffffff';
                    tooltip.style.padding = '8px 12px';
                    tooltip.style.borderRadius = '4px';
                    tooltip.style.fontSize = '12px';
                    tooltip.style.whiteSpace = 'normal';
                    tooltip.style.maxWidth = '250px';
                    tooltip.style.zIndex = '10';
                    tooltip.style.marginBottom = '4px';
                    tooltip.style.display = 'none';
                    tooltip.style.boxShadow = 'var(--shadow-md)';
                    tooltip.textContent = row.errorMessage;

                    container.appendChild(statusBadge);
                    container.appendChild(tooltip);
                    container.addEventListener('mouseenter', () => {
                        tooltip.style.display = 'block';
                    });
                    container.addEventListener('mouseleave', () => {
                        tooltip.style.display = 'none';
                    });
                    statusTd.appendChild(container);
                } else {
                    statusTd.appendChild(statusBadge);
                }
                tr.appendChild(statusTd);

                // Actions
                const actionsTd = document.createElement('td');
                actionsTd.style.whiteSpace = 'nowrap';

                const previewBtn = document.createElement('button');
                previewBtn.className = 'btn btn-sm btn-primary';
                previewBtn.textContent = 'Voir';
                previewBtn.style.marginRight = '6px';
                previewBtn.onclick = function() {
                    openPreview(row.id, row.filename);
                };
                actionsTd.appendChild(previewBtn);

                const reprintBtn = document.createElement('button');
                reprintBtn.className = 'btn btn-sm btn-secondary';
                reprintBtn.textContent = 'Réimprimer';
                reprintBtn.onclick = async () => {
                    reprintBtn.disabled = true;
                    try {
                        await window.go.main.App.ReprintDocument(row.id);
                        showToast('Document en cours de réimpression...', 'success');
                    } catch (e) {
                        showToast('Erreur lors de la réimpression: ' + e, 'error');
                    }
                    reprintBtn.disabled = false;
                };
                actionsTd.appendChild(reprintBtn);
                tr.appendChild(actionsTd);

                tbody.appendChild(tr);
            });

            // Update pagination
            pageInfoSpan.textContent = `Affichage ${offset + 1} - ${offset + data.length}`;
            prevBtn.disabled = offset === 0;
            nextBtn.disabled = data.length < pageSize;
        } catch (e) {
            showToast('Erreur lors du chargement: ' + e, 'error');
            tbody.innerHTML = '<tr><td colspan="7" style="text-align: center; padding: 20px;">Erreur de chargement</td></tr>';
        }
    }

    // Event listeners
    filterBtn.addEventListener('click', () => {
        currentOffset = 0;
        loadData(currentOffset);
    });

    prevBtn.addEventListener('click', () => {
        if (currentOffset >= pageSize) {
            currentOffset -= pageSize;
            loadData(currentOffset);
        }
    });

    nextBtn.addEventListener('click', () => {
        currentOffset += pageSize;
        loadData(currentOffset);
    });

    // Allow Enter key on date inputs to filter
    [dateFromInput, dateToInput].forEach(input => {
        input.addEventListener('keypress', (e) => {
            if (e.key === 'Enter') {
                currentOffset = 0;
                loadData(currentOffset);
            }
        });
    });

    // Initial load
    loadData(currentOffset);

    // Listen to file:processed event
    window.runtime.EventsOn('file:processed', () => {
        currentOffset = 0;
        loadData(currentOffset);
    });

    app.appendChild(page);
});
