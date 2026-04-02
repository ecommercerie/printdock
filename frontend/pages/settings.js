registerPage('#settings', function(app) {
    const page = document.createElement('div');
    page.className = 'page';

    const title = document.createElement('h1');
    title.textContent = 'Configuration';
    page.appendChild(title);

    // ===== General Config Section =====
    const generalSection = document.createElement('div');
    generalSection.className = 'card mb-6';

    const generalTitle = document.createElement('h2');
    generalTitle.className = 'section-title';
    generalTitle.textContent = 'Paramètres généraux';
    generalSection.appendChild(generalTitle);

    // Watch directory
    const watchGroup = document.createElement('div');
    watchGroup.className = 'form-group';

    const watchLabel = document.createElement('label');
    watchLabel.textContent = 'Répertoire de surveillance';
    watchGroup.appendChild(watchLabel);

    const watchInputContainer = document.createElement('div');
    watchInputContainer.style.display = 'flex';
    watchInputContainer.style.gap = '8px';

    const watchInput = document.createElement('input');
    watchInput.type = 'text';
    watchInput.id = 'watch-dir-input';
    watchInput.placeholder = 'Chemin du répertoire';
    watchInput.style.flex = '1';
    watchInputContainer.appendChild(watchInput);

    const watchBtn = document.createElement('button');
    watchBtn.className = 'btn btn-secondary btn-sm';
    watchBtn.textContent = 'Parcourir';
    watchBtn.onclick = async () => {
        const path = await window.go.main.App.PickDirectory('Sélectionner le répertoire de surveillance');
        if (path) {
            watchInput.value = path;
        }
    };
    watchInputContainer.appendChild(watchBtn);
    watchGroup.appendChild(watchInputContainer);
    generalSection.appendChild(watchGroup);

    // Archive directory
    const archiveGroup = document.createElement('div');
    archiveGroup.className = 'form-group';

    const archiveLabel = document.createElement('label');
    archiveLabel.textContent = 'Répertoire d\'archivage';
    archiveGroup.appendChild(archiveLabel);

    const archiveInputContainer = document.createElement('div');
    archiveInputContainer.style.display = 'flex';
    archiveInputContainer.style.gap = '8px';

    const archiveInput = document.createElement('input');
    archiveInput.type = 'text';
    archiveInput.id = 'archive-dir-input';
    archiveInput.placeholder = 'Chemin du répertoire';
    archiveInput.style.flex = '1';
    archiveInputContainer.appendChild(archiveInput);

    const archiveBtn = document.createElement('button');
    archiveBtn.className = 'btn btn-secondary btn-sm';
    archiveBtn.textContent = 'Parcourir';
    archiveBtn.onclick = async () => {
        const path = await window.go.main.App.PickDirectory('Sélectionner le répertoire d\'archivage');
        if (path) {
            archiveInput.value = path;
        }
    };
    archiveInputContainer.appendChild(archiveBtn);
    archiveGroup.appendChild(archiveInputContainer);
    generalSection.appendChild(archiveGroup);

    // Stability delay
    const delayGroup = document.createElement('div');
    delayGroup.className = 'form-group';

    const delayLabel = document.createElement('label');
    delayLabel.textContent = 'Délai de stabilité (secondes)';
    delayGroup.appendChild(delayLabel);

    const delayInput = document.createElement('input');
    delayInput.type = 'number';
    delayInput.id = 'stability-delay-input';
    delayInput.placeholder = '2';
    delayInput.min = '0';
    delayGroup.appendChild(delayInput);
    generalSection.appendChild(delayGroup);

    // Worker count
    const workerGroup = document.createElement('div');
    workerGroup.className = 'form-group';

    const workerLabel = document.createElement('label');
    workerLabel.textContent = 'Nombre de travailleurs';
    workerGroup.appendChild(workerLabel);

    const workerInput = document.createElement('input');
    workerInput.type = 'number';
    workerInput.id = 'worker-count-input';
    workerInput.placeholder = '4';
    workerInput.min = '1';
    workerGroup.appendChild(workerInput);
    generalSection.appendChild(workerGroup);

    // Auto-cleanup days
    const cleanupGroup = document.createElement('div');
    cleanupGroup.className = 'form-group';

    const cleanupLabel = document.createElement('label');
    cleanupLabel.textContent = 'Nettoyage automatique (jours)';
    cleanupGroup.appendChild(cleanupLabel);

    const cleanupRow = document.createElement('div');
    cleanupRow.style.display = 'flex';
    cleanupRow.style.gap = '8px';
    cleanupRow.style.alignItems = 'center';

    const cleanupInput = document.createElement('input');
    cleanupInput.type = 'number';
    cleanupInput.id = 'cleanup-days-input';
    cleanupInput.placeholder = '0 = désactivé';
    cleanupInput.min = '0';
    cleanupInput.style.width = '120px';
    cleanupRow.appendChild(cleanupInput);

    const cleanupHint = document.createElement('span');
    cleanupHint.style.fontSize = '12px';
    cleanupHint.style.color = 'var(--color-text-muted)';
    cleanupHint.textContent = '0 = désactivé — supprime historique et fichiers archivés';
    cleanupRow.appendChild(cleanupHint);

    cleanupGroup.appendChild(cleanupRow);
    generalSection.appendChild(cleanupGroup);

    // Keep original files
    const keepOriginalGroup = document.createElement('div');
    keepOriginalGroup.className = 'form-group';
    keepOriginalGroup.style.display = 'flex';
    keepOriginalGroup.style.alignItems = 'center';
    keepOriginalGroup.style.gap = '12px';

    const keepOriginalCheckbox = document.createElement('input');
    keepOriginalCheckbox.type = 'checkbox';
    keepOriginalCheckbox.id = 'keep-original-checkbox';
    keepOriginalCheckbox.style.width = '18px';
    keepOriginalCheckbox.style.height = '18px';
    keepOriginalCheckbox.style.cursor = 'pointer';
    keepOriginalGroup.appendChild(keepOriginalCheckbox);

    const keepOriginalLabel = document.createElement('label');
    keepOriginalLabel.htmlFor = 'keep-original-checkbox';
    keepOriginalLabel.style.cursor = 'pointer';
    keepOriginalLabel.style.margin = '0';
    keepOriginalLabel.textContent = 'Conserver les fichiers originaux';
    keepOriginalGroup.appendChild(keepOriginalLabel);
    generalSection.appendChild(keepOriginalGroup);

    // Notification sound
    const notifSoundGroup = document.createElement('div');
    notifSoundGroup.className = 'form-group';
    notifSoundGroup.style.display = 'flex';
    notifSoundGroup.style.alignItems = 'center';
    notifSoundGroup.style.gap = '12px';

    const notifSoundCheckbox = document.createElement('input');
    notifSoundCheckbox.type = 'checkbox';
    notifSoundCheckbox.id = 'notification-sound-checkbox';
    notifSoundCheckbox.style.width = '18px';
    notifSoundCheckbox.style.height = '18px';
    notifSoundCheckbox.style.cursor = 'pointer';
    notifSoundGroup.appendChild(notifSoundCheckbox);

    const notifSoundLabel = document.createElement('label');
    notifSoundLabel.htmlFor = 'notification-sound-checkbox';
    notifSoundLabel.style.cursor = 'pointer';
    notifSoundLabel.style.margin = '0';
    notifSoundLabel.textContent = 'Son de notification';
    notifSoundGroup.appendChild(notifSoundLabel);
    generalSection.appendChild(notifSoundGroup);

    // Rescan on startup
    const rescanGroup = document.createElement('div');
    rescanGroup.className = 'form-group';
    rescanGroup.style.display = 'flex';
    rescanGroup.style.alignItems = 'center';
    rescanGroup.style.gap = '12px';

    const rescanCheckbox = document.createElement('input');
    rescanCheckbox.type = 'checkbox';
    rescanCheckbox.id = 'rescan-on-startup-checkbox';
    rescanCheckbox.style.width = '18px';
    rescanCheckbox.style.height = '18px';
    rescanCheckbox.style.cursor = 'pointer';
    rescanGroup.appendChild(rescanCheckbox);

    const rescanLabel = document.createElement('label');
    rescanLabel.htmlFor = 'rescan-on-startup-checkbox';
    rescanLabel.style.cursor = 'pointer';
    rescanLabel.style.margin = '0';
    rescanLabel.textContent = 'Scanner les fichiers existants au démarrage';
    rescanGroup.appendChild(rescanLabel);
    generalSection.appendChild(rescanGroup);

    // Webhook URL
    const webhookGroup = document.createElement('div');
    webhookGroup.className = 'form-group';

    const webhookLabel = document.createElement('label');
    webhookLabel.textContent = 'URL Webhook';
    webhookGroup.appendChild(webhookLabel);

    const webhookInput = document.createElement('input');
    webhookInput.type = 'text';
    webhookInput.id = 'webhook-url-input';
    webhookInput.placeholder = 'https://...';
    webhookGroup.appendChild(webhookInput);
    generalSection.appendChild(webhookGroup);

    // Extra watch directories
    const extraWatchGroup = document.createElement('div');
    extraWatchGroup.className = 'form-group';

    const extraWatchLabel = document.createElement('label');
    extraWatchLabel.textContent = 'Dossiers de surveillance supplémentaires';
    extraWatchGroup.appendChild(extraWatchLabel);

    const extraWatchHint = document.createElement('span');
    extraWatchHint.style.fontSize = '12px';
    extraWatchHint.style.color = 'var(--color-text-muted)';
    extraWatchHint.style.display = 'block';
    extraWatchHint.style.marginBottom = '4px';
    extraWatchHint.textContent = 'Un chemin par ligne';
    extraWatchGroup.appendChild(extraWatchHint);

    const extraWatchTextarea = document.createElement('textarea');
    extraWatchTextarea.id = 'extra-watch-dirs-input';
    extraWatchTextarea.placeholder = 'C:\\chemin\\vers\\dossier';
    extraWatchTextarea.rows = 4;
    extraWatchGroup.appendChild(extraWatchTextarea);
    generalSection.appendChild(extraWatchGroup);

    // Allowed printers
    const allowedPrinterGroup = document.createElement('div');
    allowedPrinterGroup.className = 'form-group';

    const allowedPrinterLabel = document.createElement('label');
    allowedPrinterLabel.textContent = 'Imprimantes autorisées';
    allowedPrinterGroup.appendChild(allowedPrinterLabel);

    const allowedPrinterHint = document.createElement('span');
    allowedPrinterHint.style.fontSize = '12px';
    allowedPrinterHint.style.color = 'var(--color-text-muted)';
    allowedPrinterHint.style.display = 'block';
    allowedPrinterHint.style.marginBottom = '4px';
    allowedPrinterHint.textContent = 'Cochez les imprimantes que les utilisateurs peuvent sélectionner. Aucune cochée = toutes autorisées.';
    allowedPrinterGroup.appendChild(allowedPrinterHint);

    const allowedPrinterList = document.createElement('div');
    allowedPrinterList.id = 'allowed-printer-list';
    allowedPrinterList.style.maxHeight = '150px';
    allowedPrinterList.style.overflowY = 'auto';
    allowedPrinterList.style.border = '1px solid var(--color-border)';
    allowedPrinterList.style.borderRadius = '4px';
    allowedPrinterList.style.padding = '8px';
    allowedPrinterGroup.appendChild(allowedPrinterList);
    generalSection.appendChild(allowedPrinterGroup);

    // Load all printers for the allowed list
    (async () => {
        try {
            const allPrinters = await window.go.main.App.GetAllPrinters();
            const cfg = await window.go.main.App.GetConfig();
            const allowed = cfg.allowedPrinters || [];
            allowedPrinterList.innerHTML = '';
            if (!allPrinters || allPrinters.length === 0) {
                allowedPrinterList.textContent = 'Aucune imprimante détectée';
                return;
            }
            allPrinters.forEach(p => {
                const row = document.createElement('div');
                row.style.display = 'flex';
                row.style.alignItems = 'center';
                row.style.gap = '8px';
                row.style.padding = '4px 0';
                const cb = document.createElement('input');
                cb.type = 'checkbox';
                cb.value = p;
                cb.checked = allowed.length === 0 || allowed.includes(p);
                cb.className = 'allowed-printer-cb';
                row.appendChild(cb);
                const lbl = document.createElement('span');
                lbl.textContent = p;
                lbl.style.fontSize = '13px';
                row.appendChild(lbl);
                allowedPrinterList.appendChild(row);
            });
        } catch(e) {}
    })();

    // Save button
    const saveBtn = document.createElement('button');
    saveBtn.className = 'btn btn-primary';
    saveBtn.textContent = 'Sauvegarder';
    saveBtn.onclick = async () => {
        // Collect allowed printers
        const allCbs = document.querySelectorAll('.allowed-printer-cb');
        const checkedPrinters = [];
        let allChecked = true;
        allCbs.forEach(cb => {
            if (cb.checked) checkedPrinters.push(cb.value);
            else allChecked = false;
        });

        const config = {
            watchDir: watchInput.value,
            archiveDir: archiveInput.value,
            stabilityDelaySeconds: parseInt(delayInput.value) || 0,
            workerCount: parseInt(workerInput.value) || 1,
            autoCleanupDays: parseInt(cleanupInput.value) || 0,
            keepOriginal: keepOriginalCheckbox.checked,
            notificationSound: notifSoundCheckbox.checked,
            rescanOnStartup: rescanCheckbox.checked,
            webhookUrl: webhookInput.value,
            extraWatchDirs: extraWatchTextarea.value.split('\n').map(s => s.trim()).filter(s => s.length > 0),
            allowedPrinters: allChecked ? [] : checkedPrinters
        };

        const error = await window.go.main.App.SaveConfig(config);
        if (error) {
            showToast('Erreur lors de la sauvegarde: ' + error, 'error');
        } else {
            showToast('Configuration sauvegardée avec succès', 'success');
        }
    };
    generalSection.appendChild(saveBtn);
    page.appendChild(generalSection);

    // Load initial config
    (async () => {
        const config = await window.go.main.App.GetConfig();
        if (config) {
            watchInput.value = config.watchDir || '';
            archiveInput.value = config.archiveDir || '';
            delayInput.value = config.stabilityDelaySeconds || 2;
            workerInput.value = config.workerCount || 4;
            cleanupInput.value = config.autoCleanupDays || 0;
            keepOriginalCheckbox.checked = !!config.keepOriginal;
            notifSoundCheckbox.checked = !!config.notificationSound;
            rescanCheckbox.checked = !!config.rescanOnStartup;
            webhookInput.value = config.webhookUrl || '';
            extraWatchTextarea.value = (config.extraWatchDirs || []).join('\n');
        }
    })();

    // ===== Startup Section =====
    const startupSection = document.createElement('div');
    startupSection.className = 'card mb-6';

    const startupTitle = document.createElement('h2');
    startupTitle.className = 'section-title';
    startupTitle.textContent = 'Démarrage automatique';
    startupSection.appendChild(startupTitle);

    const startupGroup = document.createElement('div');
    startupGroup.className = 'form-group';
    startupGroup.style.display = 'flex';
    startupGroup.style.alignItems = 'center';
    startupGroup.style.gap = '12px';

    const startupCheckbox = document.createElement('input');
    startupCheckbox.type = 'checkbox';
    startupCheckbox.id = 'startup-checkbox';
    startupCheckbox.style.width = '18px';
    startupCheckbox.style.height = '18px';
    startupCheckbox.style.cursor = 'pointer';
    startupGroup.appendChild(startupCheckbox);

    const startupLabel = document.createElement('label');
    startupLabel.htmlFor = 'startup-checkbox';
    startupLabel.style.cursor = 'pointer';
    startupLabel.style.margin = '0';
    startupLabel.textContent = 'Lancer PrintDock au démarrage de Windows';
    startupGroup.appendChild(startupLabel);

    startupSection.appendChild(startupGroup);

    const startupStatus = document.createElement('div');
    startupStatus.id = 'startup-status';
    startupStatus.style.fontSize = '13px';
    startupStatus.style.color = 'var(--color-text-muted)';
    startupStatus.style.marginTop = '8px';
    startupSection.appendChild(startupStatus);

    page.appendChild(startupSection);

    // Load startup status
    (async () => {
        try {
            const enabled = await window.go.main.App.IsStartupEnabled();
            startupCheckbox.checked = enabled;
            startupStatus.textContent = enabled
                ? 'PrintDock est configuré pour démarrer automatiquement.'
                : 'PrintDock ne démarre pas automatiquement.';
            startupStatus.style.color = enabled ? 'var(--color-success)' : 'var(--color-text-muted)';
        } catch (e) {
            startupStatus.textContent = 'Impossible de vérifier le statut du démarrage automatique.';
        }
    })();

    startupCheckbox.addEventListener('change', async function() {
        try {
            await window.go.main.App.SetStartupEnabled(startupCheckbox.checked);
            if (startupCheckbox.checked) {
                startupStatus.textContent = 'PrintDock est configuré pour démarrer automatiquement.';
                startupStatus.style.color = 'var(--color-success)';
                showToast('Démarrage automatique activé', 'success');
            } else {
                startupStatus.textContent = 'PrintDock ne démarre pas automatiquement.';
                startupStatus.style.color = 'var(--color-text-muted)';
                showToast('Démarrage automatique désactivé', 'success');
            }
        } catch (e) {
            startupCheckbox.checked = !startupCheckbox.checked;
            showToast('Erreur lors de la modification du démarrage automatique', 'error');
        }
    });

    // ===== Maintenance Section =====
    const maintenanceSection = document.createElement('div');
    maintenanceSection.className = 'card mb-6';

    const maintenanceTitle = document.createElement('h2');
    maintenanceTitle.className = 'section-title';
    maintenanceTitle.textContent = 'Maintenance';
    maintenanceSection.appendChild(maintenanceTitle);

    const maintenanceBtns = document.createElement('div');
    maintenanceBtns.style.display = 'flex';
    maintenanceBtns.style.gap = '12px';
    maintenanceBtns.style.flexWrap = 'wrap';

    // Import config button
    const importBtn = document.createElement('button');
    importBtn.className = 'btn btn-secondary';
    importBtn.textContent = 'Importer une configuration';
    importBtn.onclick = async () => {
        try {
            await window.go.main.App.ImportConfig();
            showToast('Configuration importée', 'success');
            // Reload page to reflect changes
            window.location.hash = '#settings';
            navigate();
        } catch(e) { showToast('Erreur: ' + e, 'error'); }
    };
    maintenanceBtns.appendChild(importBtn);

    // Export config button
    const exportBtn = document.createElement('button');
    exportBtn.className = 'btn btn-secondary';
    exportBtn.textContent = 'Exporter la configuration';
    exportBtn.onclick = async () => {
        try {
            await window.go.main.App.SaveExportFile();
            showToast('Configuration exportée', 'success');
        } catch (e) {
            showToast('Erreur lors de l\'export: ' + e, 'error');
        }
    };
    maintenanceBtns.appendChild(exportBtn);

    // Manual cleanup button
    const cleanupBtn = document.createElement('button');
    cleanupBtn.className = 'btn btn-secondary';
    cleanupBtn.textContent = 'Nettoyer maintenant';
    cleanupBtn.onclick = async () => {
        try {
            const count = await window.go.main.App.RunCleanupNow();
            showToast(count + ' entrée(s) supprimée(s)', 'success');
        } catch (e) {
            showToast('Erreur: ' + e, 'error');
        }
    };
    maintenanceBtns.appendChild(cleanupBtn);

    // Reset database button
    const resetBtn = document.createElement('button');
    resetBtn.className = 'btn btn-danger';
    resetBtn.textContent = 'Réinitialiser l\'historique';
    resetBtn.onclick = async () => {
        if (!confirm('Êtes-vous sûr de vouloir supprimer tout l\'historique ?\nCette action est irréversible.')) {
            return;
        }
        try {
            await window.go.main.App.ResetHistory();
            showToast('Historique réinitialisé', 'success');
        } catch (e) {
            showToast('Erreur: ' + e, 'error');
        }
    };
    maintenanceBtns.appendChild(resetBtn);

    maintenanceSection.appendChild(maintenanceBtns);
    page.appendChild(maintenanceSection);

    // ===== Rules Section =====
    const rulesSection = document.createElement('div');
    rulesSection.className = 'card';

    const rulesTitle = document.createElement('h2');
    rulesTitle.className = 'section-title';
    rulesTitle.textContent = 'Règles d\'impression';
    rulesSection.appendChild(rulesTitle);

    // Add rule button
    const addRuleBtn = document.createElement('button');
    addRuleBtn.className = 'btn btn-primary mb-4';
    addRuleBtn.textContent = 'Ajouter une règle';
    addRuleBtn.onclick = () => showRuleForm(null);
    rulesSection.appendChild(addRuleBtn);

    // Rules table
    const tableWrap = document.createElement('div');
    tableWrap.className = 'table-wrap';
    tableWrap.id = 'rules-table-wrap';
    rulesSection.appendChild(tableWrap);

    page.appendChild(rulesSection);
    app.appendChild(page);

    // Function to refresh rules list
    async function refreshRules() {
        const rules = await window.go.main.App.GetRules();
        const tableWrap = document.getElementById('rules-table-wrap');
        tableWrap.innerHTML = '';

        if (!rules || rules.length === 0) {
            const empty = document.createElement('div');
            empty.style.padding = '20px';
            empty.style.textAlign = 'center';
            empty.style.color = 'var(--color-text-muted)';
            empty.textContent = 'Aucune règle définie';
            tableWrap.appendChild(empty);
            return;
        }

        const table = document.createElement('table');
        const thead = document.createElement('thead');
        const headerRow = document.createElement('tr');

        const headers = ['Ordre', 'Nom', 'Regex', 'Largeur (mm)', 'Hauteur (mm)', 'Tolérance (mm)', 'Pages', 'Imprimante', 'Sous-répertoire', 'Actions'];
        headers.forEach(h => {
            const th = document.createElement('th');
            th.textContent = h;
            headerRow.appendChild(th);
        });
        thead.appendChild(headerRow);
        table.appendChild(thead);

        const tbody = document.createElement('tbody');
        rules.forEach((rule, index) => {
            const tr = document.createElement('tr');

            // Order cell with Up/Down buttons
            const orderTd = document.createElement('td');
            orderTd.style.whiteSpace = 'nowrap';

            const upBtn = document.createElement('button');
            upBtn.className = 'btn btn-secondary btn-sm';
            upBtn.textContent = '↑';
            upBtn.disabled = index === 0;
            upBtn.onclick = async () => {
                const names = rules.map(r => r.name);
                names.splice(index - 1, 0, names.splice(index, 1)[0]);
                await window.go.main.App.ReorderRules(names);
                refreshRules();
            };
            orderTd.appendChild(upBtn);

            const downBtn = document.createElement('button');
            downBtn.className = 'btn btn-secondary btn-sm';
            downBtn.textContent = '↓';
            downBtn.disabled = index === rules.length - 1;
            downBtn.style.marginLeft = '4px';
            downBtn.onclick = async () => {
                const names = rules.map(r => r.name);
                names.splice(index + 1, 0, names.splice(index, 1)[0]);
                await window.go.main.App.ReorderRules(names);
                refreshRules();
            };
            orderTd.appendChild(downBtn);
            tr.appendChild(orderTd);

            const cells = [
                rule.name,
                rule.match.filenameRegex || '-',
                rule.match.pageWidthMm || '-',
                rule.match.pageHeightMm || '-',
                rule.match.dimensionToleranceMm || '-',
                rule.match.pageCount || '-',
                rule.action.printer || '-',
                rule.action.archiveSubdir || '-'
            ];

            cells.forEach(cellText => {
                const td = document.createElement('td');
                td.textContent = cellText;
                tr.appendChild(td);
            });

            // Action buttons: Edit + Delete
            const actionTd = document.createElement('td');
            actionTd.style.whiteSpace = 'nowrap';

            const editBtn = document.createElement('button');
            editBtn.className = 'btn btn-primary btn-sm';
            editBtn.textContent = 'Éditer';
            editBtn.style.marginRight = '4px';
            editBtn.onclick = () => showRuleForm(rule);
            actionTd.appendChild(editBtn);

            const deleteBtn = document.createElement('button');
            deleteBtn.className = 'btn btn-danger btn-sm';
            deleteBtn.textContent = 'Supprimer';
            deleteBtn.onclick = async () => {
                if (confirm('Êtes-vous sûr de vouloir supprimer cette règle?')) {
                    await window.go.main.App.DeleteRule(rule.name);
                    refreshRules();
                }
            };
            actionTd.appendChild(deleteBtn);
            tr.appendChild(actionTd);

            tbody.appendChild(tr);
        });
        table.appendChild(tbody);
        tableWrap.appendChild(table);
    }

    // Function to show add/edit rule form. Pass null to add, or an existing rule object to edit.
    async function showRuleForm(existingRule) {
        const isEdit = !!existingRule;
        const overlay = document.createElement('div');
        overlay.className = 'modal-overlay';
        overlay.onclick = (e) => { if (e.target === overlay) overlay.remove(); };

        const modal = document.createElement('div');
        modal.className = 'modal';

        const header = document.createElement('div');
        header.className = 'modal-header';
        const headerTitle = document.createElement('div');
        headerTitle.className = 'modal-title';
        headerTitle.textContent = isEdit ? 'Éditer la règle' : 'Ajouter une règle';
        header.appendChild(headerTitle);
        const closeBtn = document.createElement('button');
        closeBtn.className = 'modal-close';
        closeBtn.textContent = '×';
        closeBtn.onclick = () => overlay.remove();
        header.appendChild(closeBtn);
        modal.appendChild(header);

        const body = document.createElement('div');
        body.className = 'modal-body flex-col';

        // Helper to create form groups
        function addField(label, type, placeholder, value) {
            const group = document.createElement('div');
            group.className = 'form-group';
            const lbl = document.createElement('label');
            lbl.textContent = label;
            group.appendChild(lbl);
            const input = document.createElement('input');
            input.type = type;
            input.placeholder = placeholder;
            if (value) input.value = value;
            group.appendChild(input);
            body.appendChild(group);
            return input;
        }

        const nameInput = addField('Nom', 'text', 'Nom de la règle', isEdit ? existingRule.name : '');
        if (isEdit) nameInput.disabled = true; // can't rename

        // Hint: regex is optional if dimensions are set
        const regexHintGroup = document.createElement('div');
        regexHintGroup.className = 'form-group';
        const regexHintLabel = document.createElement('label');
        regexHintLabel.textContent = 'Expression régulière du nom de fichier';
        regexHintGroup.appendChild(regexHintLabel);
        const regexHint = document.createElement('span');
        regexHint.style.fontSize = '11px';
        regexHint.style.color = 'var(--color-text-muted)';
        regexHint.style.display = 'block';
        regexHint.style.marginBottom = '4px';
        regexHint.textContent = 'Optionnel — laissez vide pour matcher uniquement par dimensions';
        regexHintGroup.appendChild(regexHint);
        const regexInput = document.createElement('input');
        regexInput.type = 'text';
        regexInput.placeholder = 'ex: (?i)label|expedition';
        if (isEdit && existingRule.match.filenameRegex) regexInput.value = existingRule.match.filenameRegex;
        regexHintGroup.appendChild(regexInput);
        body.appendChild(regexHintGroup);

        const widthInput = addField('Largeur de page (mm)', 'number', '100 pour étiquette, 210 pour A4',
            isEdit && existingRule.match.pageWidthMm ? existingRule.match.pageWidthMm : '');
        const heightInput = addField('Hauteur de page (mm)', 'number', '150 pour étiquette, 297 pour A4',
            isEdit && existingRule.match.pageHeightMm ? existingRule.match.pageHeightMm : '');
        const toleranceInput = addField('Tolérance de dimension (mm)', 'number', '5',
            isEdit && existingRule.match.dimensionToleranceMm ? existingRule.match.dimensionToleranceMm : '');
        const countInput = addField('Nombre de pages (optionnel)', 'number', '',
            isEdit && existingRule.match.pageCount ? existingRule.match.pageCount : '');

        // Printer dropdown
        const printerGroup = document.createElement('div');
        printerGroup.className = 'form-group';
        const printerLabel = document.createElement('label');
        printerLabel.textContent = 'Imprimante';
        printerGroup.appendChild(printerLabel);
        const printerSelect = document.createElement('select');
        printerGroup.appendChild(printerSelect);
        body.appendChild(printerGroup);

        const subdirInput = addField('Sous-répertoire d\'archivage', 'text', 'ex: thermal, a4',
            isEdit ? (existingRule.action.archiveSubdir || '') : '');

        modal.appendChild(body);

        const footer = document.createElement('div');
        footer.className = 'modal-footer';
        const cancelBtn = document.createElement('button');
        cancelBtn.className = 'btn btn-secondary';
        cancelBtn.textContent = 'Annuler';
        cancelBtn.onclick = () => overlay.remove();
        footer.appendChild(cancelBtn);

        const saveBtn = document.createElement('button');
        saveBtn.className = 'btn btn-primary';
        saveBtn.textContent = 'Enregistrer';
        saveBtn.onclick = async () => {
            const rule = {
                name: nameInput.value,
                match: {
                    filenameRegex: regexInput.value || '',
                    pageWidthMm: parseFloat(widthInput.value) || 0,
                    pageHeightMm: parseFloat(heightInput.value) || 0,
                    dimensionToleranceMm: parseFloat(toleranceInput.value) || 0,
                    pageCount: parseInt(countInput.value) || 0
                },
                action: {
                    printer: printerSelect.value,
                    archiveSubdir: subdirInput.value || ''
                }
            };
            if (!rule.name) {
                showToast('Le nom de la règle est obligatoire', 'error');
                return;
            }
            try {
                if (isEdit) {
                    await window.go.main.App.UpdateRule(existingRule.name, rule);
                    showToast('Règle mise à jour', 'success');
                } else {
                    await window.go.main.App.SaveRule(rule);
                    showToast('Règle ajoutée', 'success');
                }
                overlay.remove();
                refreshRules();
            } catch (e) {
                showToast('Erreur: ' + e, 'error');
            }
        };
        footer.appendChild(saveBtn);
        modal.appendChild(footer);
        overlay.appendChild(modal);
        document.body.appendChild(overlay);

        // Load printers (filtered by allowed list)
        const printers = await window.go.main.App.GetPrinters();
        if (printers) {
            printers.forEach(p => {
                const option = document.createElement('option');
                option.value = p;
                option.textContent = p;
                printerSelect.appendChild(option);
            });
        }
        if (isEdit && existingRule.action.printer) {
            printerSelect.value = existingRule.action.printer;
        }
    }

    // Initial load
    refreshRules();

    // Listen for editRule event from learning page
    function onEditRule(e) {
        if (e.detail) showRuleForm(e.detail);
    }
    window.addEventListener('editRule', onEditRule);
});
