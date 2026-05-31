/**
 * PDF import review: edit draft problems one-by-one with PATCH /update-problem on approve or question switch.
 */
(function () {
    const conceptsPlaceholderHtml = '<option value="" disabled>Select a topic first</option>';

    // show loading state and reset topic when chapters reload
    window.beforeImportReviewChaptersRequest = function () {
        const chapter = document.getElementById('review-chapter-dropdown');
        if (chapter) {
            chapter.innerHTML = '<option value="" disabled selected>Loading chapters...</option>';
        }
        const topic = document.getElementById('review-topic-dropdown');
        if (topic) {
            topic.innerHTML = '<option value="">Select Topic</option>';
        }
        const concepts = document.getElementById('review-concepts-dropdown');
        if (concepts) {
            concepts.innerHTML = conceptsPlaceholderHtml;
        }
    };

    // show a loading state while topics load
    window.beforeImportReviewTopicsRequest = function () {
        const topic = document.getElementById('review-topic-dropdown');
        if (topic) {
            topic.innerHTML = '<option value="" disabled selected>Loading topics...</option>';
        }
        const concepts = document.getElementById('review-concepts-dropdown');
        if (concepts) {
            concepts.innerHTML = conceptsPlaceholderHtml;
        }
    };

    const STATUS_PENDING = 'pending';
    const STATUS_APPROVED = 'approved';
    const STATUS_EXCLUDED = 'excluded';

    const labelPool = 'ABCDEFGHIJ';
    const bootstrapEl = document.getElementById('import-review-bootstrap');
    if (!bootstrapEl) return;

    let bootstrap;
    try {
        bootstrap = JSON.parse(bootstrapEl.textContent);
    } catch (e) {
        console.error('Invalid import review bootstrap JSON', e);
        return;
    }

    const testId = bootstrap.testId;
    const problems = bootstrap.problems || [];
    const problemById = new Map(problems.map((p) => [p.id, p]));
    const statusById = new Map(problems.map((p) => [p.id, STATUS_PENDING]));

    let activeIndex = 0;
    let saving = false;

    const sidebar = document.getElementById('import-review-sidebar');
    const progressEl = document.getElementById('import-review-progress');
    const continueBtn = document.getElementById('import-review-continue-btn');
    const saveStatusEl = document.getElementById('review-save-status');
    const typeSelect = document.getElementById('review-problem-type');
    const questionDiv = document.getElementById('review-question-div');
    const optionsDiv = document.getElementById('review-options-div');
    const optionTabs = document.getElementById('review-option-tabs');
    const addTabBtn = document.getElementById('review-add-tab');
    const answerDiv = document.getElementById('review-answer-div');
    const answerContent = document.getElementById('review-answer-content');
    const solutionDiv = document.getElementById('review-solution-div');
    const chapterDropdown = document.getElementById('review-chapter-dropdown');
    const topicDropdown = document.getElementById('review-topic-dropdown');
    const skillsDropdown = document.getElementById('review-skills-dropdown');
    const conceptsDropdown = document.getElementById('review-concepts-dropdown');

    const questionEditor = questionDiv.querySelector('.editor');
    const optionsEditor = optionsDiv.querySelector('.editor');
    const solutionEditor = solutionDiv.querySelector('.editor');

    let tabs = [];
    let tabContent = [];
    let activeTabIndex = 0;
    const answerLabels = new Set();

    function activeProblem() {
        return problems[activeIndex];
    }

    function activeId() {
        return activeProblem()?.id;
    }

    function statusIcon(status) {
        switch (status) {
            case STATUS_APPROVED:
                return '✓';
            case STATUS_EXCLUDED:
                return '—';
            default:
                return '○';
        }
    }

    function statusClass(status) {
        switch (status) {
            case STATUS_APPROVED:
                return 'text-green-600';
            case STATUS_EXCLUDED:
                return 'text-gray-400 line-through';
            default:
                return 'text-amber-500';
        }
    }

    function refreshSidebar() {
        sidebar.querySelectorAll('.import-review-nav-btn').forEach((btn) => {
            const id = parseInt(btn.dataset.problemId, 10);
            const st = statusById.get(id) || STATUS_PENDING;
            const icon = btn.querySelector('.import-review-status-icon');
            if (icon) {
                icon.textContent = statusIcon(st);
                icon.className = 'import-review-status-icon shrink-0 w-5 text-center ' + statusClass(st);
            }
            btn.classList.toggle('bg-white', problems[activeIndex]?.id === id);
            btn.classList.toggle('border-blue-300', problems[activeIndex]?.id === id);
            btn.classList.toggle('font-medium', problems[activeIndex]?.id === id);
        });
        updateProgress();
    }

    function updateProgress() {
        const active = problems.filter((p) => statusById.get(p.id) !== STATUS_EXCLUDED);
        const approved = active.filter((p) => statusById.get(p.id) === STATUS_APPROVED).length;
        progressEl.textContent = `${approved} / ${active.length} approved`;
        const allApproved = active.length > 0 && approved === active.length;
        continueBtn.disabled = !allApproved;
    }

    function setSaveStatus(msg, isError) {
        if (!saveStatusEl) return;
        saveStatusEl.textContent = msg || '';
        saveStatusEl.classList.toggle('text-red-600', !!isError);
        saveStatusEl.classList.toggle('text-gray-500', !isError);
    }

    function isMcq(subtype) {
        return subtype === 'mcq_single_answer' || subtype === 'mcq_multiple_answer' || subtype === 'matrix_match';
    }

    function isNumerical(subtype) {
        return subtype === 'numerical_answer';
    }

    function onTypeVisibility(subtype) {
        const mcq = isMcq(subtype);
        optionsDiv.classList.toggle('hidden', !mcq);
        if (!mcq) {
            renderIntegerOrNumericalAnswer(subtype);
        }
    }

    function renderIntegerOrNumericalAnswer(subtype) {
        if (isNumerical(subtype)) {
            answerContent.innerHTML = `
                <div class="mb-2">
                    <label class="mr-4"><input type="radio" name="review-numType" value="single" checked> Single Value</label>
                    <label><input type="radio" name="review-numType" value="range"> Range</label>
                </div>
                <div id="review-single-answer">
                    <input type="text" name="review-singleAnswer" class="border p-2 rounded w-full" placeholder="Answer" />
                </div>
                <div id="review-range-answer" class="hidden flex gap-2">
                    <input type="text" name="review-minAnswer" class="border p-2 rounded w-full" placeholder="Min" />
                    <input type="text" name="review-maxAnswer" class="border p-2 rounded w-full" placeholder="Max" />
                </div>`;
            answerContent.querySelectorAll('input[name="review-numType"]').forEach((radio) => {
                radio.addEventListener('change', () => {
                    const range = radio.value === 'range';
                    document.getElementById('review-single-answer').classList.toggle('hidden', range);
                    document.getElementById('review-range-answer').classList.toggle('hidden', !range);
                });
            });
        } else {
            answerContent.innerHTML = `<input type="text" id="review-integer-answer" class="border p-2 rounded w-full" placeholder="Integer answer" />`;
        }
    }

    function getLabel(index) {
        return labelPool[index];
    }

    function saveCurrentTabContent() {
        if (!isMcq(typeSelect.value)) return;
        tabContent[activeTabIndex] = {
            html: optionsEditor.innerHTML,
        };
    }

    function renderMcqTabs() {
        optionTabs.querySelectorAll('.mcq-tab').forEach((t) => t.remove());
        answerContent.querySelectorAll('label').forEach((l) => l.remove());

        const subtype = typeSelect.value;
        const isSingle = subtype === 'mcq_single_answer' || subtype === 'matrix_match';

        tabs.forEach((_, index) => {
            const label = getLabel(index);
            const tab = document.createElement('div');
            tab.className = 'mcq-tab' + (index === activeTabIndex ? ' active' : '');
            tab.dataset.index = index;
            tab.innerHTML = `<span class="mr-2">${label}</span><button type="button" title="Remove" class="remove-btn">&times;</button>`;
            tab.addEventListener('click', () => {
                if (index === activeTabIndex) return;
                saveCurrentTabContent();
                activeTabIndex = index;
                renderMcqTabs();
                optionsEditor.innerHTML = tabContent[activeTabIndex]?.html || '';
                if (window.initializeRichTextEditors) {
                    window.initializeRichTextEditors(optionsDiv);
                }
            });
            tab.querySelector('.remove-btn').addEventListener('click', (e) => {
                e.stopPropagation();
                if (tabs.length <= 1) {
                    alert('At least one option is required.');
                    return;
                }
                saveCurrentTabContent();
                tabs.splice(index, 1);
                tabContent.splice(index, 1);
                answerLabels.delete(getLabel(index));
                if (activeTabIndex >= tabs.length) activeTabIndex = tabs.length - 1;
                renderMcqTabs();
                optionsEditor.innerHTML = tabContent[activeTabIndex]?.html || '';
            });
            optionTabs.insertBefore(tab, addTabBtn);

            const checkboxLabel = document.createElement('label');
            checkboxLabel.className = 'mr-4';
            checkboxLabel.innerHTML = `<input type="${isSingle ? 'radio' : 'checkbox'}" name="review-answer" value="${label}"> ${label}`;
            const checkbox = checkboxLabel.querySelector('input');
            checkbox.checked = answerLabels.has(label);
            checkbox.addEventListener('change', (e) => {
                if (isSingle) {
                    answerLabels.clear();
                    if (e.target.checked) answerLabels.add(label);
                } else if (e.target.checked) {
                    answerLabels.add(label);
                } else {
                    answerLabels.delete(label);
                }
            });
            answerContent.appendChild(checkboxLabel);
        });
        addTabBtn.style.display = tabs.length >= 10 ? 'none' : 'flex';
    }

    addTabBtn.addEventListener('click', () => {
        if (tabs.length >= labelPool.length) return;
        saveCurrentTabContent();
        tabs.push(null);
        tabContent.push({ html: '' });
        activeTabIndex = tabs.length - 1;
        renderMcqTabs();
        optionsEditor.innerHTML = '';
        if (window.initializeRichTextEditors) {
            window.initializeRichTextEditors(optionsDiv);
        }
    });

    function parsePositiveId(value) {
        const id = parseInt(value, 10);
        return Number.isInteger(id) && id > 0 ? id : null;
    }

    function fetchText(url) {
        return fetch(url, { headers: { 'HX-Request': 'true' } }).then((res) => {
            if (!res.ok) throw new Error(res.statusText);
            return res.text();
        });
    }

    function getConceptDisplayName(concept) {
        if (!concept?.name?.length) return String(concept?.id ?? '');
        const en = concept.name.find((n) => n.lang_code === 'en');
        return en?.concept ?? concept.name[0].concept ?? '';
    }

    function setMultiSelect(dropdown, ids) {
        const idSet = new Set((ids || []).map(Number));
        Array.from(dropdown.options).forEach((opt) => {
            if (opt.value) opt.selected = idSet.has(parseInt(opt.value, 10));
        });
    }

    function getSelectedConceptsFromDropdown() {
        return Array.from(conceptsDropdown.selectedOptions)
            .map((opt) => {
                const id = parsePositiveId(opt.value);
                if (!id) return null;
                return {
                    id,
                    name: [{ lang_code: 'en', concept: opt.textContent }],
                };
            })
            .filter(Boolean);
    }

    function normalizeProblemConcepts(concepts) {
        return (concepts || [])
            .map((c) => {
                const id = parsePositiveId(c?.id);
                if (!id) return null;
                return { ...c, id };
            })
            .filter(Boolean);
    }

    function loadSkillsForProblem(p) {
        const skillIds = p.skill_ids || [];
        const apply = () => {
            const loadingOnly = skillsDropdown.options.length === 1
                && skillsDropdown.querySelector('option[disabled]');
            if (loadingOnly) return false;
            setMultiSelect(skillsDropdown, skillIds);
            return true;
        };
        if (apply()) return;
        const observer = new MutationObserver(() => {
            if (apply()) observer.disconnect();
        });
        observer.observe(skillsDropdown, { childList: true, subtree: true });
        setTimeout(() => observer.disconnect(), 8000);
    }

    function loadConceptsForProblem(p) {
        const topicId = parsePositiveId(topicDropdown.value);
        const selectedConcepts = p?.concepts?.length
            ? normalizeProblemConcepts(p.concepts)
            : getSelectedConceptsFromDropdown();

        if (!topicId) {
            conceptsDropdown.innerHTML = conceptsPlaceholderHtml;
            return;
        }

        conceptsDropdown.innerHTML = '<option value="" disabled>Loading concepts...</option>';
        const excludeIds = selectedConcepts.map((c) => c.id).join(',');
        let url = `/api/concepts?topic_id=${topicId}`;
        if (excludeIds) url += `&exclude=${encodeURIComponent(excludeIds)}`;

        fetchText(url)
            .then((html) => {
                conceptsDropdown.innerHTML = '';
                selectedConcepts.forEach((c) => {
                    const opt = document.createElement('option');
                    opt.value = c.id;
                    opt.textContent = getConceptDisplayName(c);
                    opt.selected = true;
                    conceptsDropdown.appendChild(opt);
                });
                if (html.trim()) {
                    conceptsDropdown.insertAdjacentHTML('beforeend', html);
                }
                const selectedIds = new Set(selectedConcepts.map((c) => c.id));
                Array.from(conceptsDropdown.options).forEach((opt) => {
                    const id = parsePositiveId(opt.value);
                    if (id && selectedIds.has(id)) {
                        opt.selected = true;
                    }
                });
            })
            .catch((err) => {
                console.error('Error loading concepts:', err);
                conceptsDropdown.innerHTML = '<option disabled>Error loading concepts</option>';
            });
    }

    function tryLoadConceptsForTopic(problem) {
        if (!parsePositiveId(topicDropdown.value)) return;
        loadConceptsForProblem(problem ?? { concepts: getSelectedConceptsFromDropdown() });
    }

    function loadChapterTopic(p) {
        const chapterId = p.chapter_id ? String(p.chapter_id) : '';
        const topicId = p.topic_id ? String(p.topic_id) : '';

        const finish = () => tryLoadConceptsForTopic(p);

        const setWhenReady = (dropdown, value, eventName) => {
            if (!value) return Promise.resolve(false);
            const apply = () => {
                const opt = dropdown.querySelector(`option[value="${value}"]`);
                if (opt) {
                    dropdown.value = value;
                    if (eventName) htmx.trigger(dropdown, eventName);
                    return true;
                }
                return false;
            };
            if (apply()) return Promise.resolve(true);
            return new Promise((resolve) => {
                const observer = new MutationObserver(() => {
                    if (apply()) {
                        observer.disconnect();
                        resolve(true);
                    }
                });
                observer.observe(dropdown, { childList: true, subtree: true });
                setTimeout(() => {
                    observer.disconnect();
                    resolve(false);
                }, 8000);
            });
        };

        if (chapterId) {
            setWhenReady(chapterDropdown, chapterId, 'change').then(() => {
                if (topicId) {
                    setWhenReady(topicDropdown, topicId, null).then(finish);
                } else {
                    finish();
                }
            });
        } else {
            chapterDropdown.value = '';
            topicDropdown.innerHTML = '<option value="">Select Topic</option>';
            conceptsDropdown.innerHTML = conceptsPlaceholderHtml;
        }
    }

    function loadProblemIntoForm(index) {
        const p = problems[index];
        if (!p) return;

        activeIndex = index;
        typeSelect.value = p.subtype || 'mcq_single_answer';
        onTypeVisibility(typeSelect.value);

        const diff = p.difficulty_level || 'medium';
        document.querySelectorAll('input[name="review-difficulty"]').forEach((r) => {
            r.checked = r.value === diff;
        });

        questionEditor.innerHTML = p.meta_data?.text || '';
        solutionEditor.innerHTML = (p.meta_data?.solutions && p.meta_data.solutions[0]?.value) || '';

        answerLabels.clear();
        if (isMcq(p.subtype)) {
            const opts = p.meta_data?.options || [];
            tabs = opts.length ? new Array(opts.length).fill(null) : [null, null, null, null];
            tabContent = opts.map((html) => ({ html: html || '' }));
            if (!tabContent.length) {
                tabContent = [{ html: '' }, { html: '' }, { html: '' }, { html: '' }];
                tabs = tabContent.slice();
            }
            activeTabIndex = 0;
            (p.meta_data?.answer || []).forEach((ans) => {
                const idx = parseInt(ans, 10);
                if (!isNaN(idx) && idx > 0) answerLabels.add(getLabel(idx - 1));
            });
            renderMcqTabs();
            optionsEditor.innerHTML = tabContent[0]?.html || '';
        } else {
            renderIntegerOrNumericalAnswer(p.subtype);
            const answers = p.meta_data?.answer || [];
            if (isNumerical(p.subtype)) {
                if (answers.length === 2) {
                    const rangeRadio = answerContent.querySelector('input[value="range"]');
                    if (rangeRadio) rangeRadio.checked = true;
                    document.getElementById('review-single-answer')?.classList.add('hidden');
                    document.getElementById('review-range-answer')?.classList.remove('hidden');
                    const min = answerContent.querySelector('[name="review-minAnswer"]');
                    const max = answerContent.querySelector('[name="review-maxAnswer"]');
                    if (min) min.value = answers[0];
                    if (max) max.value = answers[1];
                } else if (answers.length === 1) {
                    const single = answerContent.querySelector('[name="review-singleAnswer"]');
                    if (single) single.value = answers[0];
                }
            } else {
                const intInput = document.getElementById('review-integer-answer');
                if (intInput && answers[0]) intInput.value = answers[0];
            }
        }

        loadChapterTopic(p);
        loadSkillsForProblem(p);
        refreshSidebar();

        if (window.initializeRichTextEditors) {
            window.initializeRichTextEditors(document.getElementById('import-review-editor-panel'));
        }
        if (window.MathJax?.typesetPromise) {
            MathJax.typesetPromise([questionDiv, optionsDiv, solutionDiv]).catch(() => {});
        }
    }

    function collectAnswers(subtype) {
        if (isMcq(subtype)) {
            saveCurrentTabContent();
            return {
                options: tabContent.map((t) => t.html),
                answers: Array.from(answerLabels)
                    .map((label) => labelPool.indexOf(label) + 1)
                    .sort((a, b) => a - b)
                    .map(String),
            };
        }
        if (isNumerical(subtype)) {
            const numType = answerContent.querySelector('input[name="review-numType"]:checked')?.value;
            if (numType === 'range') {
                const min = answerContent.querySelector('[name="review-minAnswer"]')?.value ?? '';
                const max = answerContent.querySelector('[name="review-maxAnswer"]')?.value ?? '';
                return { options: null, answers: [min, max] };
            }
            const val = answerContent.querySelector('[name="review-singleAnswer"]')?.value ?? '';
            return { options: null, answers: [val] };
        }
        const val = document.getElementById('review-integer-answer')?.value ?? '';
        return { options: null, answers: [val] };
    }

    function buildPayload() {
        const subtype = typeSelect.value;
        const chapterId = parseInt(chapterDropdown.value, 10) || 0;
        const topicId = parseInt(topicDropdown.value, 10) || 0;
        const curriculumId = parseInt(document.getElementById('curriculum-dropdown')?.value, 10) || 0;
        const gradeId = parseInt(document.getElementById('grade-dropdown')?.value, 10) || 0;
        const subjectId = parseInt(document.getElementById('subject-dropdown')?.value, 10) || 0;
        const difficulty = document.querySelector('input[name="review-difficulty"]:checked')?.value || 'medium';
        const { options, answers } = collectAnswers(subtype);
        const skillIds = Array.from(skillsDropdown.selectedOptions).map((opt) => parseInt(opt.value, 10));
        const conceptIds = Array.from(conceptsDropdown.selectedOptions)
            .map((opt) => parsePositiveId(opt.value))
            .filter(Boolean);

        return {
            lang_code: 'en',
            type: 'problem',
            subtype,
            skill_ids: skillIds,
            type_params: { test_ids: [testId] },
            curriculum_grades: [{ curriculum_id: curriculumId, grade_id: gradeId }],
            subject_id: subjectId,
            topic_id: topicId,
            chapter_id: chapterId,
            difficulty_level: difficulty,
            meta_data: {
                text: questionEditor.innerHTML.trim(),
                options,
                answer: answers,
                solutions: [{ type: 'text', value: solutionEditor.innerHTML.trim() }],
            },
            concept_ids: conceptIds,
            tags: [],
        };
    }

    function validateCurrent() {
        const chapterId = parseInt(chapterDropdown.value, 10);
        const topicId = parseInt(topicDropdown.value, 10);
        if (!chapterId || !topicId) {
            alert('Please select chapter and topic.');
            return false;
        }
        const text = questionEditor.innerHTML.trim();
        if (!text || text === '<br>') {
            alert('Please enter the question text.');
            return false;
        }
        const subtype = typeSelect.value;
        if (isMcq(subtype)) {
            saveCurrentTabContent();
            for (let i = 0; i < tabContent.length; i++) {
                if (!tabContent[i]?.html?.trim()) {
                    alert(`Option ${getLabel(i)} is empty.`);
                    return false;
                }
            }
            if (answerLabels.size === 0) {
                alert('Please select at least one correct answer.');
                return false;
            }
        }
        return true;
    }

    async function saveCurrentProblem() {
        const id = activeId();
        if (!id || statusById.get(id) === STATUS_EXCLUDED) return true;

        const payload = buildPayload();
        saving = true;
        setSaveStatus('Saving…');
        try {
            const res = await fetch(`/update-problem?id=${id}`, {
                method: 'PATCH',
                headers: { 'Content-Type': 'application/json', 'HX-Request': 'true' },
                body: JSON.stringify(payload),
            });
            if (!res.ok) {
                const msg = await res.text();
                setSaveStatus('Save failed', true);
                console.error(msg);
                return false;
            }
            Object.assign(problemById.get(id), {
                chapter_id: payload.chapter_id,
                topic_id: payload.topic_id,
                difficulty_level: payload.difficulty_level,
                skill_ids: payload.skill_ids,
                concepts: getSelectedConceptsFromDropdown(),
                meta_data: payload.meta_data,
            });
            setSaveStatus('Saved');
            return true;
        } catch (err) {
            setSaveStatus('Save failed', true);
            console.error(err);
            return false;
        } finally {
            saving = false;
        }
    }

    async function switchToIndex(index) {
        if (index === activeIndex) return;
        if (statusById.get(activeId()) !== STATUS_EXCLUDED) {
            await saveCurrentProblem();
        }
        loadProblemIntoForm(index);
    }

    sidebar.addEventListener('click', (e) => {
        const btn = e.target.closest('.import-review-nav-btn');
        if (!btn) return;
        const idx = parseInt(btn.dataset.index, 10);
        switchToIndex(idx);
    });

    document.getElementById('review-approve-btn').addEventListener('click', async () => {
        if (!validateCurrent()) return;
        const ok = await saveCurrentProblem();
        if (!ok) return;
        statusById.set(activeId(), STATUS_APPROVED);
        refreshSidebar();
        const next = problems.findIndex((p, i) => i > activeIndex && statusById.get(p.id) !== STATUS_EXCLUDED);
        if (next !== -1) switchToIndex(next);
    });

    document.getElementById('review-exclude-btn').addEventListener('click', async () => {
        if (!confirm('Exclude this question from the import? It will be archived.')) return;
        const id = activeId();
        try {
            const res = await fetch(`/archive-problem?id=${id}`, {
                method: 'PATCH',
                headers: { 'HX-Request': 'true' },
            });
            if (!res.ok) {
                alert(await res.text() || 'Failed to exclude question.');
                return;
            }
            statusById.set(id, STATUS_EXCLUDED);
            refreshSidebar();
            const next = problems.findIndex((p, i) => i > activeIndex && statusById.get(p.id) !== STATUS_EXCLUDED);
            if (next !== -1) switchToIndex(next);
        } catch (err) {
            console.error(err);
            alert('Failed to exclude question.');
        }
    });

    continueBtn.addEventListener('htmx:responseError', (evt) => {
        const xhr = evt.detail?.xhr;
        alert(xhr?.responseText || 'Cannot continue yet. Approve all questions and set chapter/topic for each.');
    });

    if (typeof htmx !== 'undefined') {
        htmx.defineExtension('import-review-cascade', {
            onEvent(name, event) {
                const elt = event.detail.elt;
                const target = event.detail.target;

                if (name === 'htmx:afterRequest' && elt?.id === 'review-topic-dropdown') {
                    tryLoadConceptsForTopic();
                    return;
                }

                if (name === 'htmx:afterSwap') {
                    if (target?.id === 'review-chapter-dropdown') {
                        const topic = document.getElementById('review-topic-dropdown');
                        if (topic) {
                            topic.innerHTML = '<option value="">Select Topic</option>';
                        }
                        const concepts = document.getElementById('review-concepts-dropdown');
                        if (concepts) {
                            concepts.innerHTML = conceptsPlaceholderHtml;
                        }
                    }
                } else if (name === 'htmx:configRequest') {
                    if (elt?.id === 'review-chapter-dropdown') {
                        const curriculum = document.getElementById('curriculum-dropdown');
                        const grade = document.getElementById('grade-dropdown');
                        const subject = document.getElementById('subject-dropdown');
                        if (!curriculum?.value || !grade?.value || !subject?.value) {
                            event.preventDefault();
                        }
                    }
                }
            },
        });
    }

    topicDropdown.addEventListener('change', () => tryLoadConceptsForTopic());

    if (problems.length) {
        loadProblemIntoForm(0);
    }
    updateProgress();
})();
