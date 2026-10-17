'use strict';

const DOCUMENT_FORMAT = 'nezha-blockly';
const DOCUMENT_VERSION = 1;
const MAX_VISIBLE_OUTPUT = 1024 * 1024;

const searchableBlocks = [
  {name: 'assignment', type: 'nezha_assign', category: 'Operations'},
  {name: 'binary', type: 'nezha_binary', category: 'Operations'},
  {name: 'boolean', type: 'nezha_boolean', category: 'Values'},
  {name: 'equal', type: 'nezha_equal', category: 'Operations'},
  {name: 'if', type: 'nezha_if_plain', category: 'Control'},
  {name: 'if / else', type: 'nezha_if', category: 'Control'},
  {name: 'let', type: 'nezha_let', category: 'Operations'},
  {name: 'name', type: 'nezha_name', category: 'Values'},
  {name: 'negative', type: 'nezha_negative', category: 'Operations'},
  {name: 'not', type: 'nezha_not', category: 'Operations'},
  {name: 'number', type: 'nezha_number', category: 'Values'},
  {name: 'print', type: 'nezha_print', category: 'Operations'},
  {name: 'println', type: 'nezha_println', category: 'Operations'},
  {name: 'read', type: 'nezha_read', category: 'Operations'},
  {name: 'rune', type: 'nezha_rune', category: 'Values'},
  {name: 'string', type: 'nezha_string', category: 'Values'},
  {name: 'symbol', type: 'nezha_symbol', category: 'Values'},
  {name: 'while', type: 'nezha_while', category: 'Control'}
];

const baseToolbox = {
  kind: 'categoryToolbox',
  contents: [
    {kind: 'category', name: 'Search', contents: []},
    {
      kind: 'category',
      name: 'Control',
      contents: [
        {kind: 'block', type: 'nezha_if_plain'},
        {kind: 'block', type: 'nezha_if'},
        {kind: 'block', type: 'nezha_while'}
      ]
    },
    {kind: 'category', name: 'Functions', contents: []},
    {
      kind: 'category',
      name: 'Operations',
      contents: [
        {kind: 'block', type: 'nezha_assign'},
        {kind: 'block', type: 'nezha_binary'},
        {kind: 'block', type: 'nezha_equal'},
        {kind: 'block', type: 'nezha_let'},
        {kind: 'block', type: 'nezha_negative'},
        {kind: 'block', type: 'nezha_not'},
        {kind: 'block', type: 'nezha_print'},
        {kind: 'block', type: 'nezha_println'},
        {kind: 'block', type: 'nezha_read'}
      ]
    },
    {
      kind: 'category',
      name: 'Values',
      contents: [
        {kind: 'block', type: 'nezha_boolean'},
        {kind: 'block', type: 'nezha_name'},
        {kind: 'block', type: 'nezha_number'},
        {kind: 'block', type: 'nezha_rune'},
        {kind: 'block', type: 'nezha_string'},
        {kind: 'block', type: 'nezha_symbol'}
      ]
    }
  ]
};

const workspace = Blockly.inject('blocklyDiv', {
  toolbox: baseToolbox,
  media: 'vendor/blockly/media/',
  trashcan: false,
  sounds: false,
  zoom: {controls: false, wheel: true, startScale: 1.0, maxScale: 2.0, minScale: 0.5},
  move: {scrollbars: true, drag: true, wheel: true}
});

const sourceEl = document.getElementById('source');
const inputEl = document.getElementById('input');
const outputEl = document.getElementById('output');
const statusEl = document.getElementById('status');
const runButton = document.getElementById('run');
const stopButton = document.getElementById('stop');
const tabs = Array.from(document.querySelectorAll('.tab'));
const views = Array.from(document.querySelectorAll('.view'));

const searchPanel = document.getElementById('blockSearchPanel');
const searchInput = document.getElementById('blockSearchInput');
const searchResults = document.getElementById('blockSearchResults');
let searchCategoryRow = null;

function toolboxItems() {
  const toolbox = workspace.getToolbox && workspace.getToolbox();
  return toolbox && toolbox.getToolboxItems ? toolbox.getToolboxItems() : [];
}

function toolboxItemName(item) {
  if (!item) return '';
  if (typeof item.getName === 'function') return item.getName();
  return item.name_ || '';
}

function selectToolboxCategory(name) {
  const toolbox = workspace.getToolbox && workspace.getToolbox();
  if (!toolbox) return;
  const item = toolboxItems().find((candidate) => toolboxItemName(candidate) === name);
  if (!item) return;
  if (typeof toolbox.setSelectedItem === 'function') toolbox.setSelectedItem(item);
}

function findSearchCategoryRow() {
  const candidates = document.querySelectorAll('#blocklyDiv .blocklyToolboxCategory');
  for (const row of candidates) {
    const label = row.textContent.trim();
    if (label === 'Search') return row;
  }
  return null;
}

function renderSearchResults() {
  const query = searchInput.value.trim().toLowerCase();
  searchResults.replaceChildren();
  const matches = searchableBlocks.filter((block) => !query || block.name.includes(query));

  if (matches.length === 0) {
    const empty = document.createElement('div');
    empty.className = 'blockSearchEmpty';
    empty.textContent = 'No matching blocks';
    searchResults.appendChild(empty);
    return;
  }

  for (const block of matches) {
    const button = document.createElement('button');
    button.type = 'button';
    button.className = 'blockSearchResult';
    button.innerHTML = `<span>${block.name}</span><small>${block.category}</small>`;
    button.addEventListener('click', () => {
      closeBlockSearch();
      selectToolboxCategory(block.category);
    });
    searchResults.appendChild(button);
  }
}

function positionSearchPanel() {
  searchCategoryRow = findSearchCategoryRow();
  if (!searchCategoryRow) return false;
  const rowRect = searchCategoryRow.getBoundingClientRect();
  const editorRect = document.getElementById('view-editor').getBoundingClientRect();
  searchPanel.style.left = `${Math.max(0, rowRect.right - editorRect.left + 6)}px`;
  searchPanel.style.top = `${Math.max(0, rowRect.top - editorRect.top)}px`;
  return true;
}

function openBlockSearch() {
  if (!positionSearchPanel()) return;
  searchPanel.hidden = false;
  searchInput.value = '';
  renderSearchResults();
  requestAnimationFrame(() => searchInput.focus());
}

function closeBlockSearch() {
  searchPanel.hidden = true;
}

function installSearchCategoryClickHook() {
  const div = document.getElementById('blocklyDiv');
  div.addEventListener('pointerdown', (event) => {
    const row = event.target.closest && event.target.closest('.blocklyToolboxCategory');
    if (!row || row.textContent.trim() !== 'Search') return;
    event.preventDefault();
    event.stopPropagation();
    openBlockSearch();
  }, true);

  document.addEventListener('pointerdown', (event) => {
    if (searchPanel.hidden || searchPanel.contains(event.target)) return;
    const row = event.target.closest && event.target.closest('.blocklyToolboxCategory');
    if (row && row.textContent.trim() === 'Search') return;
    closeBlockSearch();
  }, true);
}

searchInput.addEventListener('input', renderSearchResults);
searchInput.addEventListener('keydown', (event) => {
  if (event.key === 'Escape') {
    closeBlockSearch();
  } else if (event.key === 'Enter') {
    const first = searchResults.querySelector('.blockSearchResult');
    if (first) first.click();
  }
});
installSearchCategoryClickHook();



let currentPath = '';
let dirty = false;
let loading = false;
function appBridge() {
  if (!window.go || !window.go.main || !window.go.main.App) {
    throw new Error('Wails bridge is not available');
  }
  return window.go.main.App;
}

function baseName(path) {
  if (!path) return 'Untitled';
  const parts = path.split(/[\\/]/);
  return parts[parts.length - 1] || 'Untitled';
}

function updateWindowTitle() {
  const mark = dirty ? ' *' : '';
  const title = `${baseName(currentPath)}${mark} — Nezha Blockly`;
  if (window.runtime && window.runtime.WindowSetTitle) {
    window.runtime.WindowSetTitle(title);
  }
}

function setDirty(value) {
  dirty = value;
  updateWindowTitle();
}

function switchView(name) {
  for (const tab of tabs) tab.classList.toggle('active', tab.dataset.view === name);
  for (const view of views) view.classList.toggle('active', view.id === `view-${name}`);
  if (name === 'editor') {
    requestAnimationFrame(() => Blockly.svgResize(workspace));
  }
}

for (const tab of tabs) {
  tab.addEventListener('click', () => switchView(tab.dataset.view));
}


function refreshBindingFields() {
  for (const block of workspace.getAllBlocks(false)) {
    if (block.type !== 'nezha_name' && block.type !== 'nezha_assign') continue;
    const field = block.getField('NAME');
    const available = nezhaPriorBindings(block);
    const current = field.getValue();
    if (available.includes(current)) continue;
    field.setValue(available.length > 0 ? available[0] : '');
  }
}

function refreshSource() {
  try {
    sourceEl.textContent = nezhaSource(workspace);
    statusEl.textContent = 'ready';
  } catch (err) {
    sourceEl.textContent = '';
    statusEl.textContent = err.message;
  }
}

workspace.addChangeListener(function(event) {
  if (event.isUiEvent) return;
  refreshBindingFields();
  refreshSource();
  if (!loading) setDirty(true);
});

inputEl.addEventListener('input', function() {
  if (!loading) setDirty(true);
});

function appendRunOutput(chunk) {
  if (typeof chunk !== 'string' || chunk.length === 0) return;
  outputEl.textContent += chunk;
  if (outputEl.textContent.length > MAX_VISIBLE_OUTPUT) {
    outputEl.textContent = outputEl.textContent.slice(-MAX_VISIBLE_OUTPUT);
  }
  outputEl.scrollTop = outputEl.scrollHeight;
}

function finishRun(result) {
  runButton.disabled = false;
  stopButton.disabled = true;

  if (result && result.error) {
    outputEl.classList.add('error');
    outputEl.textContent += (outputEl.textContent ? '\n' : '') + result.error + '\n';
    statusEl.textContent = 'error';
  } else if (result && result.stopped) {
    statusEl.textContent = 'stopped';
  } else {
    statusEl.textContent = 'done';
  }
}

runButton.addEventListener('click', async function() {
  switchView('editor');
  const source = sourceEl.textContent;
  outputEl.classList.remove('error');
  outputEl.textContent = '';
  statusEl.textContent = 'running';
  runButton.disabled = true;
  stopButton.disabled = false;

  try {
    const result = await appBridge().Start(source, inputEl.value);
    if (result.error) finishRun(result);
  } catch (err) {
    finishRun({error: err.message});
  }
});

stopButton.addEventListener('click', function() {
  if (!window.runtime || !window.runtime.EventsEmit) {
    outputEl.classList.add('error');
    outputEl.textContent += (outputEl.textContent ? '\n' : '') + 'Wails event bridge is not available\n';
    statusEl.textContent = 'error';
    return;
  }

  stopButton.disabled = true;
  statusEl.textContent = 'stopping';
  window.runtime.EventsEmit('run:stop');
});

function makeBlock(type) {
  const block = workspace.newBlock(type);
  block.initSvg();
  return block;
}

function connectValue(statement, inputName, value) {
  statement.getInput(inputName).connection.connect(value.outputConnection);
}

function connectStatements(blocks) {
  for (let i = 0; i + 1 < blocks.length; i += 1) {
    blocks[i].nextConnection.connect(blocks[i + 1].previousConnection);
  }
}

function stringValue(value) {
  const block = makeBlock('nezha_string');
  block.setFieldValue(value, 'VALUE');
  return block;
}

function printlnValue(value) {
  const statement = makeBlock('nezha_println');
  connectValue(statement, 'VALUE', value);
  return statement;
}

function printValue(value) {
  const statement = makeBlock('nezha_print');
  connectValue(statement, 'VALUE', value);
  return statement;
}

function buildHelloSample() {
  inputEl.value = '';
  return [printlnValue(stringValue('Hello, world!'))];
}

function buildInputOutputSample() {
  inputEl.value = 'Max';

  const binding = makeBlock('nezha_let');
  binding.setFieldValue('name', 'NAME');
  connectValue(binding, 'VALUE', makeBlock('nezha_read'));

  const hello = printValue(stringValue('hello, '));
  const output = makeBlock('nezha_println');
  connectStatements([binding, hello, output]);

  const name = makeBlock('nezha_name');
  connectValue(output, 'VALUE', name);
  refreshBindingFields();
  name.setFieldValue('name', 'NAME');
  return [binding, hello, output];
}

function resetVisualState() {
  outputEl.classList.remove('error');
  outputEl.textContent = '';
}

function loadSample(name) {
  loading = true;
  Blockly.Events.disable();
  let blocks = [];
  try {
    workspace.clear();
    resetVisualState();
    currentPath = '';

    switch (name) {
      case 'hello':
        blocks = buildHelloSample();
        break;
      case 'io':
      default:
        blocks = buildInputOutputSample();
        break;
    }

    for (const block of workspace.getAllBlocks(false)) block.render();
    if (blocks.length > 0) blocks[0].moveBy(80, 60);
  } finally {
    Blockly.Events.enable();
    loading = false;
  }

  refreshBindingFields();
  refreshSource();
  setDirty(false);
  statusEl.textContent = 'sample loaded';
  switchView('editor');
}

for (const button of document.querySelectorAll('[data-sample]')) {
  button.addEventListener('click', () => loadSample(button.dataset.sample));
}

function documentState() {
  return {
    format: DOCUMENT_FORMAT,
    version: DOCUMENT_VERSION,
    input: inputEl.value,
    workspace: Blockly.serialization.workspaces.save(workspace)
  };
}

function serializeDocument() {
  return JSON.stringify(documentState(), null, 2) + '\n';
}

function loadDocumentState(doc) {
  if (!doc || doc.format !== DOCUMENT_FORMAT) {
    throw new Error('not a Nezha Blockly document');
  }
  if (doc.version !== DOCUMENT_VERSION) {
    throw new Error(`unsupported Nezha Blockly document version: ${doc.version}`);
  }
  if (!doc.workspace || typeof doc.workspace !== 'object') {
    throw new Error('document has no Blockly workspace');
  }

  loading = true;
  Blockly.Events.disable();
  try {
    workspace.clear();
    Blockly.serialization.workspaces.load(doc.workspace, workspace);
    inputEl.value = typeof doc.input === 'string' ? doc.input : '';
    resetVisualState();
  } finally {
    Blockly.Events.enable();
    loading = false;
  }

  refreshBindingFields();
  refreshSource();
  setDirty(false);
  switchView('editor');
}

async function confirmDiscardIfNeeded() {
  if (!dirty) return true;
  try {
    return await appBridge().ConfirmDiscard();
  } catch (_) {
    return window.confirm('Discard unsaved changes?');
  }
}

function newDocument() {
  loading = true;
  Blockly.Events.disable();
  try {
    workspace.clear();
    inputEl.value = '';
    resetVisualState();
    currentPath = '';
  } finally {
    Blockly.Events.enable();
    loading = false;
  }
  refreshSource();
  setDirty(false);
  statusEl.textContent = 'new document';
  switchView('editor');
}

async function requestNew() {
  if (!(await confirmDiscardIfNeeded())) return;
  newDocument();
}

async function requestOpen() {
  if (!(await confirmDiscardIfNeeded())) return;
  try {
    const result = await appBridge().OpenDocument();
    if (result.error) throw new Error(result.error);
    if (!result.path) return;
    const doc = JSON.parse(result.content);
    loadDocumentState(doc);
    currentPath = result.path;
    setDirty(false);
    statusEl.textContent = `opened ${baseName(currentPath)}`;
  } catch (err) {
    statusEl.textContent = 'open failed';
    outputEl.classList.add('error');
    outputEl.textContent = err.message + '\n';
    switchView('editor');
  }
}

async function requestSave(saveAs) {
  try {
    const result = await appBridge().SaveDocument(currentPath, serializeDocument(), saveAs);
    if (result.error) throw new Error(result.error);
    if (!result.path) return false;
    currentPath = result.path;
    setDirty(false);
    statusEl.textContent = `saved ${baseName(currentPath)}`;
    return true;
  } catch (err) {
    statusEl.textContent = 'save failed';
    outputEl.classList.add('error');
    outputEl.textContent = err.message + '\n';
    switchView('editor');
    return false;
  }
}

async function requestExport() {
  try {
    const result = await appBridge().ExportSource(sourceEl.textContent);
    if (result.error) throw new Error(result.error);
    if (result.path) statusEl.textContent = `exported ${baseName(result.path)}`;
  } catch (err) {
    statusEl.textContent = 'export failed';
    outputEl.classList.add('error');
    outputEl.textContent = err.message + '\n';
    switchView('editor');
  }
}

async function requestQuit() {
  if (!(await confirmDiscardIfNeeded())) return;
  await appBridge().Quit();
}

function registerMenuEvents() {
  if (!window.runtime || !window.runtime.EventsOn) return;
  window.runtime.EventsOn('file:new', requestNew);
  window.runtime.EventsOn('file:open', requestOpen);
  window.runtime.EventsOn('file:save', () => requestSave(false));
  window.runtime.EventsOn('file:save-as', () => requestSave(true));
  window.runtime.EventsOn('file:export', requestExport);
  window.runtime.EventsOn('file:quit', requestQuit);
  window.runtime.EventsOn('edit:undo', () => workspace.undo(false));
  window.runtime.EventsOn('edit:redo', () => workspace.undo(true));
  window.runtime.EventsOn('view:editor', () => switchView('editor'));
  window.runtime.EventsOn('view:samples', () => switchView('samples'));
  window.runtime.EventsOn('view:help', () => switchView('help'));
  window.runtime.EventsOn('run:output', appendRunOutput);
  window.runtime.EventsOn('run:finished', finishRun);
}

registerMenuEvents();

// Begin with the I/O sample as the working document.
loadSample('io');
