registerPage('#learning', function(app) {
    const page = document.createElement('div');
    page.className = 'page';

    // Title
    const title = document.createElement('h1');
    title.textContent = 'Apprentissage';
    page.appendChild(title);

    // Drop zone + file selector
    const dropZone = document.createElement('div');
    dropZone.style.border = '2px dashed var(--color-border)';
    dropZone.style.borderRadius = '8px';
    dropZone.style.padding = '20px';
    dropZone.style.textAlign = 'center';
    dropZone.style.marginBottom = '16px';
    dropZone.style.cursor = 'pointer';
    dropZone.style.transition = 'border-color 0.2s, background 0.2s';
    dropZone.style.color = 'var(--color-text-muted)';
    dropZone.innerHTML = 'Glissez un PDF ici ou <strong>cliquez pour sélectionner</strong> un fichier pour créer une règle';
    dropZone.setAttribute('data-wails-drop-target', 'drop');

    dropZone.addEventListener('click', function() {
        handleAnalyzePDFDialog();
    });
    page.appendChild(dropZone);

    // Wails native file drop handler
    if (window.runtime && window.runtime.OnFileDrop) {
        window.runtime.OnFileDrop(function(x, y, paths) {
            if (!paths || paths.length === 0) return;
            // Only process if we're on the learning page
            if (window.location.hash !== '#learning') return;
            const pdfPath = paths.find(function(p) { return p.toLowerCase().endsWith('.pdf'); });
            if (pdfPath) {
                handleAnalyzePDFPath(pdfPath);
            } else {
                showToast('Seuls les fichiers PDF sont acceptés', 'error');
            }
        }, true);
    }

    // Handle drop highlight via Wails drag events
    if (window.runtime && window.runtime.OnFileDragStatusChange) {
        window.runtime.OnFileDragStatusChange(function(entering) {
            if (window.location.hash !== '#learning') return;
            if (entering) {
                dropZone.style.borderColor = 'var(--color-primary)';
                dropZone.style.background = 'rgba(59,130,246,0.05)';
            } else {
                dropZone.style.borderColor = 'var(--color-border)';
                dropZone.style.background = '';
            }
        });
    }

    // Analyze via file dialog (click)
    async function handleAnalyzePDFDialog() {
        try {
            const result = await window.go.main.App.AnalyzePDF();
            handleAnalysisResult(result);
        } catch (e) {
            showToast('Erreur: ' + e, 'error');
        }
    }

    // Analyze a specific file path (drag & drop)
    async function handleAnalyzePDFPath(path) {
        try {
            const result = await window.go.main.App.AnalyzePDFPath(path);
            handleAnalysisResult(result);
        } catch (e) {
            showToast('Erreur: ' + e, 'error');
        }
    }

    // Handle analysis result (shared between dialog and drop)
    function handleAnalysisResult(result) {
        if (!result) return;
        if (result.matchedRule) {
            var msg = 'Ce document correspond à la règle "' + result.matchedRule.name + '" '
                + '(imprimante: ' + result.matchedRule.action.printer + ').\n\n'
                + 'Voulez-vous modifier cette règle ?';
            if (confirm(msg)) {
                window.location.hash = '#settings';
                setTimeout(function() {
                    window.dispatchEvent(new CustomEvent('editRule', { detail: result.matchedRule }));
                }, 300);
            }
        } else {
            showToast('Document ajouté à la file d\'apprentissage', 'success');
            loadLearningQueue();
        }
    }

    // Container for queue items
    const queueContainer = document.createElement('div');
    queueContainer.id = 'learning-queue-container';
    page.appendChild(queueContainer);

    app.appendChild(page);

    // Helper: extract a generic base name from a filename by stripping
    // the extension, dates (YYYY-MM-DD, YYYYMMDD), trailing sequence numbers,
    // and trailing separators.
    function extractBaseName(filename) {
        let name = filename.replace(/\.[^/.]+$/, ''); // strip extension
        // Remove date patterns: 2026-03-20, 20260320, 2026_03_20
        name = name.replace(/[_\-]?\d{4}[-_]?\d{2}[-_]?\d{2}/g, '');
        // Remove trailing sequence numbers like -1, _2, _001
        name = name.replace(/[_\-]\d+$/g, '');
        // Remove trailing separators
        name = name.replace(/[_\-]+$/g, '');
        return name;
    }

    // Helper: generate a case-insensitive regex from a filename
    function generateRegex(filename) {
        const base = extractBaseName(filename);
        if (!base) {
            // Fallback: escape the whole filename
            return '(?i)' + filename.replace(/\.[^/.]+$/, '').replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
        }
        // Escape regex special chars in the base name
        const escaped = base.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
        return '(?i)' + escaped;
    }

    // Helper: generate rule name from filename
    function generateRuleName(filename) {
        const base = extractBaseName(filename);
        const clean = (base || filename.replace(/\.[^/.]+$/, ''))
            .replace(/[^a-zA-Z0-9_-]/g, '-')
            .replace(/-+/g, '-')
            .replace(/^-|-$/g, '')
            .toLowerCase();
        return 'rule-' + clean;
    }

    // Load and render learning queue
    async function loadLearningQueue() {
        try {
            const queue = await window.go.main.App.GetLearningQueue();
            const printers = await window.go.main.App.GetPrinters();

            // Update badge
            const badge = document.getElementById('learning-badge');
            if (queue && queue.length > 0) {
                badge.textContent = String(queue.length);
                badge.style.display = 'inline-block';
            } else {
                badge.style.display = 'none';
            }

            // Render queue
            queueContainer.innerHTML = '';

            if (!queue || queue.length === 0) {
                const emptyMsg = document.createElement('p');
                emptyMsg.className = 'placeholder';
                emptyMsg.textContent = 'Aucun document en attente';
                queueContainer.appendChild(emptyMsg);
                return;
            }

            queue.forEach(item => {
                const card = document.createElement('div');
                card.className = 'card';
                card.style.marginBottom = '16px';

                // Filename (bold)
                const filenameEl = document.createElement('div');
                filenameEl.style.fontWeight = 'bold';
                filenameEl.style.marginBottom = '8px';
                filenameEl.style.fontSize = '15px';
                filenameEl.textContent = item.filename;
                card.appendChild(filenameEl);

                // Metadata: dimensions and page count
                const metadataEl = document.createElement('div');
                metadataEl.style.fontSize = '13px';
                metadataEl.style.color = 'var(--color-text-muted)';
                metadataEl.style.marginBottom = '12px';
                metadataEl.textContent = `${item.pageWidthMm} x ${item.pageHeightMm} mm • ${item.pageCount} page${item.pageCount > 1 ? 's' : ''}`;
                card.appendChild(metadataEl);

                // Printer dropdown
                const printerLabel = document.createElement('label');
                printerLabel.style.display = 'block';
                printerLabel.style.marginBottom = '8px';
                printerLabel.style.fontSize = '12px';
                printerLabel.style.color = 'var(--color-text-muted)';
                printerLabel.textContent = 'Imprimante';
                card.appendChild(printerLabel);

                const printerSelect = document.createElement('select');
                printerSelect.style.width = '100%';
                printerSelect.style.padding = '6px 8px';
                printerSelect.style.marginBottom = '12px';
                printerSelect.style.border = '1px solid var(--color-border)';
                printerSelect.style.borderRadius = '4px';
                printerSelect.style.backgroundColor = 'var(--color-bg)';
                printerSelect.style.color = 'var(--color-text)';
                printerSelect.style.fontSize = '13px';

                if (printers && printers.length > 0) {
                    printers.forEach(printer => {
                        const option = document.createElement('option');
                        option.value = printer;
                        option.textContent = printer;
                        printerSelect.appendChild(option);
                    });
                } else {
                    const option = document.createElement('option');
                    option.textContent = 'Aucune imprimante';
                    option.disabled = true;
                    printerSelect.appendChild(option);
                }
                printerSelect.addEventListener('change', function() {
                    subdirInput.value = printerSelect.value;
                });
                card.appendChild(printerSelect);

                // Archive subdir input
                const subdirLabel = document.createElement('label');
                subdirLabel.style.display = 'block';
                subdirLabel.style.marginBottom = '8px';
                subdirLabel.style.fontSize = '12px';
                subdirLabel.style.color = 'var(--color-text-muted)';
                subdirLabel.textContent = 'Sous-dossier d\'archive';
                card.appendChild(subdirLabel);

                const subdirInput = document.createElement('input');
                subdirInput.type = 'text';
                subdirInput.value = (printers && printers.length > 0) ? printers[0] : 'unsorted';
                subdirInput.style.width = '100%';
                subdirInput.style.padding = '6px 8px';
                subdirInput.style.marginBottom = '12px';
                subdirInput.style.border = '1px solid var(--color-border)';
                subdirInput.style.borderRadius = '4px';
                subdirInput.style.backgroundColor = 'var(--color-bg)';
                subdirInput.style.color = 'var(--color-text)';
                subdirInput.style.fontSize = '13px';
                subdirInput.style.boxSizing = 'border-box';
                card.appendChild(subdirInput);

                // Buttons container
                const buttonsContainer = document.createElement('div');
                buttonsContainer.style.display = 'flex';
                buttonsContainer.style.gap = '8px';
                buttonsContainer.style.marginBottom = '12px';
                card.appendChild(buttonsContainer);

                // "Imprimer" button
                const printBtn = document.createElement('button');
                printBtn.textContent = 'Imprimer';
                printBtn.style.flex = '1';
                printBtn.style.padding = '8px 12px';
                printBtn.style.backgroundColor = 'var(--color-primary)';
                printBtn.style.color = 'white';
                printBtn.style.border = 'none';
                printBtn.style.borderRadius = '4px';
                printBtn.style.cursor = 'pointer';
                printBtn.style.fontSize = '13px';
                printBtn.onclick = async () => {
                    try {
                        await window.go.main.App.HandleLearning(item.id, {
                            action: 'print',
                            printer: printerSelect.value,
                            archiveSubdir: subdirInput.value
                        });
                        showToast('Document imprimé', 'success');
                        loadLearningQueue();
                    } catch (error) {
                        console.error('Error printing:', error);
                        showToast('Erreur lors de l\'impression', 'error');
                    }
                };
                buttonsContainer.appendChild(printBtn);

                // "Ignorer" button
                const ignoreBtn = document.createElement('button');
                ignoreBtn.textContent = 'Ignorer';
                ignoreBtn.style.flex = '1';
                ignoreBtn.style.padding = '8px 12px';
                ignoreBtn.style.backgroundColor = 'var(--color-danger)';
                ignoreBtn.style.color = 'white';
                ignoreBtn.style.border = 'none';
                ignoreBtn.style.borderRadius = '4px';
                ignoreBtn.style.cursor = 'pointer';
                ignoreBtn.style.fontSize = '13px';
                ignoreBtn.onclick = async () => {
                    try {
                        await window.go.main.App.HandleLearning(item.id, {
                            action: 'ignore'
                        });
                        showToast('Document ignoré', 'success');
                        loadLearningQueue();
                    } catch (error) {
                        console.error('Error ignoring:', error);
                        showToast('Erreur lors de l\'ignorance du document', 'error');
                    }
                };
                buttonsContainer.appendChild(ignoreBtn);

                // "Imprimer + Mémoriser" button
                const memorizeBtn = document.createElement('button');
                memorizeBtn.textContent = 'Imprimer + Mémoriser';
                memorizeBtn.style.flex = '1.5';
                memorizeBtn.style.padding = '8px 12px';
                memorizeBtn.style.backgroundColor = 'var(--color-success)';
                memorizeBtn.style.color = 'white';
                memorizeBtn.style.border = 'none';
                memorizeBtn.style.borderRadius = '4px';
                memorizeBtn.style.cursor = 'pointer';
                memorizeBtn.style.fontSize = '13px';
                buttonsContainer.appendChild(memorizeBtn);

                // Hidden form for rule editing (shown when "Imprimer + Mémoriser" is clicked)
                const ruleFormContainer = document.createElement('div');
                ruleFormContainer.style.display = 'none';
                ruleFormContainer.style.marginTop = '12px';
                ruleFormContainer.style.padding = '12px';
                ruleFormContainer.style.backgroundColor = 'var(--color-bg-secondary)';
                ruleFormContainer.style.borderRadius = '4px';
                ruleFormContainer.style.border = '1px solid var(--color-border)';
                card.appendChild(ruleFormContainer);

                // Rule name input
                const ruleNameLabel = document.createElement('label');
                ruleNameLabel.style.display = 'block';
                ruleNameLabel.style.marginBottom = '8px';
                ruleNameLabel.style.fontSize = '12px';
                ruleNameLabel.style.color = 'var(--color-text-muted)';
                ruleNameLabel.textContent = 'Nom de la règle';
                ruleFormContainer.appendChild(ruleNameLabel);

                const ruleNameInput = document.createElement('input');
                ruleNameInput.type = 'text';
                ruleNameInput.value = generateRuleName(item.filename);
                ruleNameInput.style.width = '100%';
                ruleNameInput.style.padding = '6px 8px';
                ruleNameInput.style.marginBottom = '12px';
                ruleNameInput.style.border = '1px solid var(--color-border)';
                ruleNameInput.style.borderRadius = '4px';
                ruleNameInput.style.backgroundColor = 'var(--color-bg)';
                ruleNameInput.style.color = 'var(--color-text)';
                ruleNameInput.style.fontSize = '13px';
                ruleNameInput.style.boxSizing = 'border-box';
                ruleFormContainer.appendChild(ruleNameInput);

                // Filename regex input
                const filenameRegexLabel = document.createElement('label');
                filenameRegexLabel.style.display = 'block';
                filenameRegexLabel.style.marginBottom = '8px';
                filenameRegexLabel.style.fontSize = '12px';
                filenameRegexLabel.style.color = 'var(--color-text-muted)';
                filenameRegexLabel.textContent = 'Expression régulière du nom de fichier';
                ruleFormContainer.appendChild(filenameRegexLabel);

                const filenameRegexInput = document.createElement('input');
                filenameRegexInput.type = 'text';
                filenameRegexInput.value = generateRegex(item.filename);
                filenameRegexInput.style.width = '100%';
                filenameRegexInput.style.padding = '6px 8px';
                filenameRegexInput.style.marginBottom = '12px';
                filenameRegexInput.style.border = '1px solid var(--color-border)';
                filenameRegexInput.style.borderRadius = '4px';
                filenameRegexInput.style.backgroundColor = 'var(--color-bg)';
                filenameRegexInput.style.color = 'var(--color-text)';
                filenameRegexInput.style.fontSize = '13px';
                filenameRegexInput.style.boxSizing = 'border-box';
                ruleFormContainer.appendChild(filenameRegexInput);

                // Save and cancel buttons for rule form
                const ruleFormButtonsContainer = document.createElement('div');
                ruleFormButtonsContainer.style.display = 'flex';
                ruleFormButtonsContainer.style.gap = '8px';
                ruleFormContainer.appendChild(ruleFormButtonsContainer);

                const saveMemoizeBtn = document.createElement('button');
                saveMemoizeBtn.textContent = 'Enregistrer et imprimer';
                saveMemoizeBtn.style.flex = '1';
                saveMemoizeBtn.style.padding = '8px 12px';
                saveMemoizeBtn.style.backgroundColor = 'var(--color-success)';
                saveMemoizeBtn.style.color = 'white';
                saveMemoizeBtn.style.border = 'none';
                saveMemoizeBtn.style.borderRadius = '4px';
                saveMemoizeBtn.style.cursor = 'pointer';
                saveMemoizeBtn.style.fontSize = '13px';
                saveMemoizeBtn.onclick = async () => {
                    try {
                        await window.go.main.App.HandleLearning(item.id, {
                            action: 'print_and_memorize',
                            printer: printerSelect.value,
                            ruleName: ruleNameInput.value,
                            filenameRegex: filenameRegexInput.value,
                            archiveSubdir: subdirInput.value
                        });
                        showToast('Règle créée et document imprimé', 'success');
                        loadLearningQueue();
                    } catch (error) {
                        console.error('Error memorizing:', error);
                        showToast('Erreur lors de la mémorisation', 'error');
                    }
                };
                ruleFormButtonsContainer.appendChild(saveMemoizeBtn);

                const cancelMemoizeBtn = document.createElement('button');
                cancelMemoizeBtn.textContent = 'Annuler';
                cancelMemoizeBtn.style.flex = '1';
                cancelMemoizeBtn.style.padding = '8px 12px';
                cancelMemoizeBtn.style.backgroundColor = 'var(--color-border)';
                cancelMemoizeBtn.style.color = 'var(--color-text)';
                cancelMemoizeBtn.style.border = 'none';
                cancelMemoizeBtn.style.borderRadius = '4px';
                cancelMemoizeBtn.style.cursor = 'pointer';
                cancelMemoizeBtn.style.fontSize = '13px';
                cancelMemoizeBtn.onclick = () => {
                    ruleFormContainer.style.display = 'none';
                    memorizeBtn.style.display = 'block';
                };
                ruleFormButtonsContainer.appendChild(cancelMemoizeBtn);

                // "Imprimer + Mémoriser" button click handler
                memorizeBtn.onclick = () => {
                    ruleFormContainer.style.display = 'block';
                    memorizeBtn.style.display = 'none';
                };

                queueContainer.appendChild(card);
            });
        } catch (error) {
            console.error('Error loading learning queue:', error);
            queueContainer.innerHTML = '<p class="placeholder">Erreur lors du chargement de la file d\'attente</p>';
        }
    }

    // Initial load
    loadLearningQueue();

    // Listen for learning:new and learning:resolved events
    const unsubscribeNew = window.runtime.EventsOn('learning:new', () => {
        loadLearningQueue();
    });

    const unsubscribeResolved = window.runtime.EventsOn('learning:resolved', () => {
        loadLearningQueue();
    });

    // Store unsubscribe functions on page for cleanup if needed
    if (!page._unsubscribes) {
        page._unsubscribes = [];
    }
    page._unsubscribes.push(unsubscribeNew, unsubscribeResolved);
});
