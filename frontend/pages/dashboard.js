registerPage('#dashboard', function(app) {
    const page = document.createElement('div');
    page.className = 'page';

    // Title
    const title = document.createElement('h1');
    title.textContent = 'Tableau de bord';
    page.appendChild(title);

    // ===== Notification sound =====
    function playNotificationSound() {
        try {
            const ctx = new (window.AudioContext || window.webkitAudioContext)();
            const osc = ctx.createOscillator();
            const gain = ctx.createGain();
            osc.connect(gain);
            gain.connect(ctx.destination);
            osc.frequency.value = 800;
            gain.gain.value = 0.3;
            osc.start();
            osc.stop(ctx.currentTime + 0.15);
        } catch(e) {}
    }

    // ===== SumatraPDF Status Banner =====
    const sumatraBanner = document.createElement('div');
    sumatraBanner.className = 'card';
    sumatraBanner.style.display = 'none';
    sumatraBanner.style.background = '#fef3c7';
    sumatraBanner.style.border = '1px solid #f59e0b';
    sumatraBanner.style.marginBottom = '16px';
    sumatraBanner.style.padding = '12px 16px';
    sumatraBanner.style.alignItems = 'center';
    sumatraBanner.style.justifyContent = 'space-between';
    sumatraBanner.style.gap = '12px';

    const sumatraText = document.createElement('span');
    sumatraText.style.fontSize = '13px';
    sumatraText.style.color = '#92400e';
    sumatraBanner.appendChild(sumatraText);

    const sumatraBtn = document.createElement('button');
    sumatraBtn.className = 'btn btn-sm';
    sumatraBtn.style.background = '#f59e0b';
    sumatraBtn.style.color = '#fff';
    sumatraBtn.style.border = 'none';
    sumatraBtn.style.flexShrink = '0';
    sumatraBtn.onclick = async () => {
        sumatraBtn.disabled = true;
        sumatraBtn.textContent = 'Téléchargement...';
        try {
            await window.go.main.App.DownloadSumatra();
            sumatraBanner.style.display = 'none';
            showToast('SumatraPDF installé/mis à jour avec succès', 'success');
        } catch(e) {
            showToast('Erreur: ' + e, 'error');
            sumatraBtn.textContent = 'Réessayer';
            sumatraBtn.disabled = false;
        }
    };
    sumatraBanner.appendChild(sumatraBtn);
    page.appendChild(sumatraBanner);

    // Check SumatraPDF on load
    (async () => {
        try {
            const status = await window.go.main.App.GetSumatraStatus();
            if (!status.installed) {
                sumatraText.textContent = 'SumatraPDF non trouvé — nécessaire pour imprimer.';
                sumatraBtn.textContent = 'Télécharger SumatraPDF';
                sumatraBanner.style.display = 'flex';
            } else if (status.updateAvailable) {
                sumatraText.textContent = 'SumatraPDF ' + status.currentVersion + ' — mise à jour disponible (v' + status.latestVersion + ')';
                sumatraBtn.textContent = 'Mettre à jour';
                sumatraBanner.style.display = 'flex';
            }
        } catch(e) {}
    })();

    // ===== PrintDock Update Banner =====
    const updateBanner = document.createElement('div');
    updateBanner.className = 'card';
    updateBanner.style.display = 'none';
    updateBanner.style.background = '#dbeafe';
    updateBanner.style.border = '1px solid #3b82f6';
    updateBanner.style.marginBottom = '16px';
    updateBanner.style.padding = '12px 16px';
    updateBanner.style.alignItems = 'center';
    updateBanner.style.justifyContent = 'space-between';
    updateBanner.style.gap = '12px';

    const updateText = document.createElement('span');
    updateText.style.fontSize = '13px';
    updateText.style.color = '#1e40af';
    updateBanner.appendChild(updateText);

    const updateBtn = document.createElement('button');
    updateBtn.className = 'btn btn-sm';
    updateBtn.style.background = '#3b82f6';
    updateBtn.style.color = '#fff';
    updateBtn.style.border = 'none';
    updateBtn.style.flexShrink = '0';
    updateBtn.textContent = 'Mettre à jour';
    updateBtn.onclick = async () => {
        updateBtn.disabled = true;
        updateBtn.textContent = 'Mise à jour...';
        try {
            await window.go.main.App.ApplyUpdate();
            showToast('Mise à jour téléchargée, redémarrage...', 'success');
        } catch(e) {
            showToast('Erreur: ' + e, 'error');
            updateBtn.textContent = 'Réessayer';
            updateBtn.disabled = false;
        }
    };
    updateBanner.appendChild(updateBtn);
    page.appendChild(updateBanner);

    // Check for PrintDock update
    (async () => {
        try {
            const status = await window.go.main.App.CheckForUpdate();
            if (status.updateAvailable) {
                updateText.textContent = 'PrintDock v' + status.currentVersion + ' — nouvelle version disponible (v' + status.latestVersion + ')';
                updateBanner.style.display = 'flex';
            }
        } catch(e) {}
    })();

    // ===== Section 1: Watcher Status Bar =====
    const watcherSection = document.createElement('div');
    watcherSection.className = 'mb-6';

    const watcherSectionTitle = document.createElement('div');
    watcherSectionTitle.className = 'section-title';
    watcherSectionTitle.textContent = 'État du surveillant';
    watcherSection.appendChild(watcherSectionTitle);

    const watcherCard = document.createElement('div');
    watcherCard.className = 'card';
    watcherCard.style.display = 'flex';
    watcherCard.style.alignItems = 'center';
    watcherCard.style.justifyContent = 'space-between';
    watcherCard.style.gap = '16px';

    // Left: dot + labels
    const watcherLeft = document.createElement('div');
    watcherLeft.style.display = 'flex';
    watcherLeft.style.alignItems = 'center';
    watcherLeft.style.gap = '12px';

    const watcherDot = document.createElement('span');
    watcherDot.className = 'status-dot';
    watcherLeft.appendChild(watcherDot);

    const watcherInfo = document.createElement('div');

    const watcherLabel = document.createElement('div');
    watcherLabel.style.fontWeight = '600';
    watcherLabel.style.fontSize = '14px';
    watcherLabel.textContent = '—';
    watcherInfo.appendChild(watcherLabel);

    const watcherPath = document.createElement('div');
    watcherPath.style.fontSize = '12px';
    watcherPath.style.color = 'var(--color-text-muted)';
    watcherPath.style.marginTop = '3px';
    watcherInfo.appendChild(watcherPath);

    const watcherError = document.createElement('div');
    watcherError.style.fontSize = '12px';
    watcherError.style.color = 'var(--color-danger)';
    watcherError.style.marginTop = '3px';
    watcherError.style.display = 'none';
    watcherInfo.appendChild(watcherError);

    watcherLeft.appendChild(watcherInfo);
    watcherCard.appendChild(watcherLeft);

    // Right: Pause/Resume button
    const pauseBtn = document.createElement('button');
    pauseBtn.className = 'btn btn-secondary btn-sm';
    pauseBtn.style.flexShrink = '0';
    pauseBtn.textContent = '…';
    pauseBtn.disabled = true;

    let isPaused = false;

    pauseBtn.onclick = async () => {
        pauseBtn.disabled = true;
        try {
            if (isPaused) {
                await window.go.main.App.ResumeWatcher();
            } else {
                await window.go.main.App.PauseWatcher();
            }
            await refreshWatcherStatus();
        } catch(e) {
            console.error('Error toggling watcher pause:', e);
        } finally {
            pauseBtn.disabled = false;
        }
    };

    watcherCard.appendChild(pauseBtn);
    watcherSection.appendChild(watcherCard);
    page.appendChild(watcherSection);

    // ===== Section 2: Stats Cards =====
    const statsSectionTitle = document.createElement('div');
    statsSectionTitle.className = 'section-title';
    statsSectionTitle.textContent = 'Statistiques du jour';
    page.appendChild(statsSectionTitle);

    const cardsGrid = document.createElement('div');
    cardsGrid.className = 'cards-grid';

    function createStatCard(label) {
        const card = document.createElement('div');
        card.className = 'card';

        const cardTitle = document.createElement('div');
        cardTitle.className = 'card-title';
        cardTitle.textContent = label;
        card.appendChild(cardTitle);

        const cardValue = document.createElement('div');
        cardValue.className = 'card-value';
        cardValue.textContent = '—';
        card.appendChild(cardValue);

        cardsGrid.appendChild(card);
        return cardValue;
    }

    const valPrinted  = createStatCard('Imprimés aujourd\'hui');
    const valFailed   = createStatCard('Échoués');
    const valPending  = createStatCard('En attente');
    const valTotal    = createStatCard('Total traités');

    page.appendChild(cardsGrid);

    // ===== Section 3: Printers =====
    const printersSectionTitle = document.createElement('div');
    printersSectionTitle.className = 'section-title';
    printersSectionTitle.textContent = 'Imprimantes';
    page.appendChild(printersSectionTitle);

    const printersCard = document.createElement('div');
    printersCard.className = 'card mb-6';

    const printersList = document.createElement('div');
    printersCard.appendChild(printersList);
    page.appendChild(printersCard);

    // ===== Section 4: Chart =====
    const chartSectionTitle = document.createElement('div');
    chartSectionTitle.className = 'section-title';
    chartSectionTitle.textContent = 'Activité (30 derniers jours)';
    page.appendChild(chartSectionTitle);

    const chartCard = document.createElement('div');
    chartCard.className = 'card mb-6';

    const chartScrollWrap = document.createElement('div');
    chartScrollWrap.style.overflowX = 'auto';
    chartScrollWrap.style.overflowY = 'hidden';
    chartScrollWrap.style.paddingBottom = '8px';

    const chartInner = document.createElement('div');
    chartInner.style.display = 'flex';
    chartInner.style.flexDirection = 'column';
    chartInner.style.gap = '0';
    chartInner.style.minWidth = '600px';

    chartScrollWrap.appendChild(chartInner);
    chartCard.appendChild(chartScrollWrap);

    // Legend
    const chartLegend = document.createElement('div');
    chartLegend.style.display = 'flex';
    chartLegend.style.gap = '20px';
    chartLegend.style.marginTop = '12px';
    chartLegend.style.fontSize = '12px';
    chartLegend.style.color = 'var(--color-text-muted)';

    function makeLegendItem(color, label) {
        const item = document.createElement('div');
        item.style.display = 'flex';
        item.style.alignItems = 'center';
        item.style.gap = '6px';
        const swatch = document.createElement('span');
        swatch.style.display = 'inline-block';
        swatch.style.width = '12px';
        swatch.style.height = '12px';
        swatch.style.borderRadius = '2px';
        swatch.style.background = color;
        item.appendChild(swatch);
        const text = document.createElement('span');
        text.textContent = label;
        item.appendChild(text);
        return item;
    }

    chartLegend.appendChild(makeLegendItem('var(--color-success)', 'Imprimés'));
    chartLegend.appendChild(makeLegendItem('var(--color-danger)', 'Échoués'));
    chartCard.appendChild(chartLegend);

    page.appendChild(chartCard);

    app.appendChild(page);

    // ===== Data loading functions =====

    async function refreshWatcherStatus() {
        try {
            const status = await window.go.main.App.GetWatcherStatus();
            let paused = false;
            try {
                paused = await window.go.main.App.IsPaused();
            } catch(e) {}

            isPaused = paused;

            if (!status.running) {
                watcherDot.className = 'status-dot red';
                watcherLabel.textContent = 'Inactif';
                pauseBtn.textContent = 'Reprendre';
                pauseBtn.disabled = true;
            } else if (paused) {
                watcherDot.className = 'status-dot yellow';
                watcherLabel.textContent = 'En pause';
                pauseBtn.textContent = 'Reprendre';
                pauseBtn.disabled = false;
            } else {
                watcherDot.className = 'status-dot green';
                watcherLabel.textContent = 'Actif';
                pauseBtn.textContent = 'Pause';
                pauseBtn.disabled = false;
            }

            watcherPath.textContent = status.watchDir || '(Aucun répertoire configuré)';

            if (status.error) {
                watcherError.textContent = 'Erreur : ' + status.error;
                watcherError.style.display = 'block';
            } else {
                watcherError.style.display = 'none';
            }
        } catch(e) {
            console.error('Error loading watcher status:', e);
            watcherDot.className = 'status-dot yellow';
            watcherLabel.textContent = 'Erreur';
            pauseBtn.disabled = true;
        }
    }

    async function refreshStats() {
        try {
            const stats = await window.go.main.App.GetDashboardStats();
            valPrinted.textContent = String(stats.todayPrinted  ?? '—');
            valFailed.textContent  = String(stats.todayFailed   ?? '—');
            valPending.textContent = String(stats.pendingLearning ?? '—');
            valTotal.textContent   = String(stats.totalProcessed ?? '—');
        } catch(e) {
            console.error('Error loading dashboard stats:', e);
        }
    }

    async function refreshPrinters() {
        try {
            const [statuses, statsToday] = await Promise.all([
                window.go.main.App.GetPrinterStatuses(),
                window.go.main.App.GetPrinterStatsToday(),
            ]);

            // Build lookup maps
            const onlineMap = {};
            if (statuses && statuses.length) {
                statuses.forEach(s => { onlineMap[s.name] = s.online; });
            }
            const statsMap = {};
            if (statsToday && statsToday.length) {
                statsToday.forEach(s => { statsMap[s.printer] = s; });
            }

            // Collect all printer names
            const names = new Set();
            if (statuses)  statuses.forEach(s => names.add(s.name));
            if (statsToday) statsToday.forEach(s => names.add(s.printer));

            printersList.innerHTML = '';

            if (names.size === 0) {
                const empty = document.createElement('div');
                empty.className = 'placeholder';
                empty.textContent = 'Aucune imprimante disponible';
                printersList.appendChild(empty);
                return;
            }

            [...names].forEach((name, idx) => {
                const row = document.createElement('div');
                row.style.display = 'flex';
                row.style.alignItems = 'center';
                row.style.gap = '12px';
                row.style.padding = '10px 0';
                if (idx < names.size - 1) {
                    row.style.borderBottom = '1px solid var(--color-border)';
                }

                // Status dot
                const dot = document.createElement('span');
                const online = onlineMap[name];
                dot.className = 'status-dot ' + (online === false ? 'red' : 'green');
                row.appendChild(dot);

                // Name
                const nameEl = document.createElement('div');
                nameEl.style.flex = '1';
                nameEl.style.fontWeight = '500';
                nameEl.style.fontSize = '13px';
                nameEl.textContent = name;
                row.appendChild(nameEl);

                // Today's stats
                const statsEl = document.createElement('div');
                statsEl.style.fontSize = '12px';
                statsEl.style.color = 'var(--color-text-muted)';
                const s = statsMap[name];
                if (s) {
                    statsEl.textContent = s.count + ' doc' + (s.count !== 1 ? 's' : '') + ', ' + s.pages + ' page' + (s.pages !== 1 ? 's' : '');
                } else {
                    statsEl.textContent = '0 docs, 0 pages';
                }
                row.appendChild(statsEl);

                // Test button
                const testBtn = document.createElement('button');
                testBtn.className = 'btn btn-secondary btn-sm';
                testBtn.textContent = 'Test';
                testBtn.onclick = async () => {
                    testBtn.disabled = true;
                    testBtn.textContent = '…';
                    try {
                        const err = await window.go.main.App.TestPrint(name);
                        if (err) {
                            showToast('Erreur de test pour ' + name + ' : ' + err, 'error');
                        } else {
                            showToast('Page de test envoyée à ' + name, 'success');
                        }
                    } catch(e) {
                        showToast('Erreur lors du test d\'impression', 'error');
                    } finally {
                        testBtn.disabled = false;
                        testBtn.textContent = 'Test';
                    }
                };
                row.appendChild(testBtn);

                printersList.appendChild(row);
            });
        } catch(e) {
            console.error('Error loading printers:', e);
            printersList.innerHTML = '';
            const errEl = document.createElement('div');
            errEl.className = 'placeholder';
            errEl.textContent = 'Erreur lors du chargement des imprimantes';
            printersList.appendChild(errEl);
        }
    }

    async function refreshChart() {
        chartInner.innerHTML = '';
        try {
            const data = await window.go.main.App.GetDailyStatsChart(30);
            if (!data || data.length === 0) {
                const empty = document.createElement('div');
                empty.className = 'placeholder';
                empty.style.padding = '16px 0';
                empty.textContent = 'Aucune donnée disponible';
                chartInner.appendChild(empty);
                return;
            }

            // Find max value for scaling
            const maxVal = data.reduce((m, d) => Math.max(m, (d.printed || 0) + (d.failed || 0)), 1);

            // Bars container (horizontal bars, one row per day)
            data.forEach(d => {
                const printed = d.printed || 0;
                const failed  = d.failed  || 0;
                const total   = printed + failed;

                const row = document.createElement('div');
                row.style.display = 'flex';
                row.style.alignItems = 'center';
                row.style.gap = '8px';
                row.style.marginBottom = '4px';

                // Date label
                const dateLabel = document.createElement('div');
                dateLabel.style.width = '72px';
                dateLabel.style.flexShrink = '0';
                dateLabel.style.fontSize = '11px';
                dateLabel.style.color = 'var(--color-text-muted)';
                dateLabel.style.textAlign = 'right';
                // Format date: show MM-DD
                try {
                    const dt = new Date(d.date);
                    const mm = String(dt.getMonth() + 1).padStart(2, '0');
                    const dd = String(dt.getDate()).padStart(2, '0');
                    dateLabel.textContent = mm + '/' + dd;
                } catch(e) {
                    dateLabel.textContent = d.date || '';
                }
                row.appendChild(dateLabel);

                // Bar track
                const track = document.createElement('div');
                track.style.flex = '1';
                track.style.height = '16px';
                track.style.background = '#f3f4f6';
                track.style.borderRadius = '3px';
                track.style.overflow = 'hidden';
                track.style.display = 'flex';

                if (total > 0) {
                    const printedPct = (printed / maxVal) * 100;
                    const failedPct  = (failed  / maxVal) * 100;

                    if (printed > 0) {
                        const printedBar = document.createElement('div');
                        printedBar.style.width = printedPct + '%';
                        printedBar.style.height = '100%';
                        printedBar.style.background = 'var(--color-success)';
                        printedBar.title = printed + ' imprimé' + (printed !== 1 ? 's' : '');
                        track.appendChild(printedBar);
                    }
                    if (failed > 0) {
                        const failedBar = document.createElement('div');
                        failedBar.style.width = failedPct + '%';
                        failedBar.style.height = '100%';
                        failedBar.style.background = 'var(--color-danger)';
                        failedBar.title = failed + ' échoué' + (failed !== 1 ? 's' : '');
                        track.appendChild(failedBar);
                    }
                }

                row.appendChild(track);

                // Value label
                const valLabel = document.createElement('div');
                valLabel.style.width = '36px';
                valLabel.style.flexShrink = '0';
                valLabel.style.fontSize = '11px';
                valLabel.style.color = total > 0 ? 'var(--color-text)' : 'var(--color-text-muted)';
                valLabel.textContent = total > 0 ? String(total) : '0';
                row.appendChild(valLabel);

                chartInner.appendChild(row);
            });
        } catch(e) {
            console.error('Error loading chart data:', e);
            const errEl = document.createElement('div');
            errEl.className = 'placeholder';
            errEl.style.padding = '16px 0';
            errEl.textContent = 'Erreur lors du chargement du graphique';
            chartInner.appendChild(errEl);
        }
    }

    async function loadAll() {
        await Promise.all([
            refreshWatcherStatus(),
            refreshStats(),
            refreshPrinters(),
            refreshChart(),
        ]);
    }

    // Initial load
    loadAll();

    // Listen for events → refresh and play sound
    const unsubProcessed = window.runtime.EventsOn('file:processed', () => {
        playNotificationSound();
        loadAll();
    });

    const unsubLearning = window.runtime.EventsOn('learning:new', () => {
        playNotificationSound();
        refreshStats();
    });

    if (!page._unsubscribes) {
        page._unsubscribes = [];
    }
    page._unsubscribes.push(unsubProcessed, unsubLearning);
});
