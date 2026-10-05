'use strict';

const toolbox = {
  kind: 'flyoutToolbox',
  contents: [
    {kind: 'block', type: 'nezha_println'},
    {kind: 'sep', gap: '24'},
    {kind: 'block', type: 'nezha_number'},
    {kind: 'block', type: 'nezha_string'},
    {kind: 'block', type: 'nezha_rune'},
    {kind: 'block', type: 'nezha_symbol'},
    {kind: 'block', type: 'nezha_boolean'}
  ]
};

const workspace = Blockly.inject('blocklyDiv', {
  toolbox: toolbox,
  media: 'vendor/blockly/media/',
  trashcan: false,
  sounds: false,
  zoom: {controls: false, wheel: true, startScale: 1.0, maxScale: 2.0, minScale: 0.5},
  move: {scrollbars: true, drag: true, wheel: true}
});

const sourceEl = document.getElementById('source');
const outputEl = document.getElementById('output');
const statusEl = document.getElementById('status');
const runButton = document.getElementById('run');

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
  refreshSource();
});

runButton.addEventListener('click', async function() {
  const source = sourceEl.textContent;
  outputEl.classList.remove('error');
  outputEl.textContent = '';
  statusEl.textContent = 'running';
  runButton.disabled = true;

  try {
    const response = await fetch('/run', {
      method: 'POST',
      headers: {'Content-Type': 'text/plain; charset=utf-8'},
      body: source
    });

    if (!response.ok) {
      throw new Error(await response.text());
    }

    const result = await response.json();
    outputEl.textContent = result.output || '';
    if (result.error) {
      outputEl.classList.add('error');
      outputEl.textContent += (outputEl.textContent ? '\n' : '') + result.error + '\n';
      statusEl.textContent = 'error';
    } else {
      statusEl.textContent = 'done';
    }
  } catch (err) {
    outputEl.classList.add('error');
    outputEl.textContent = err.message + '\n';
    statusEl.textContent = 'error';
  } finally {
    runButton.disabled = false;
  }
});

// Seed the first workspace with one complete, runnable program.
const printBlock = workspace.newBlock('nezha_println');
const stringBlock = workspace.newBlock('nezha_string');
printBlock.initSvg();
stringBlock.initSvg();
printBlock.render();
stringBlock.render();
printBlock.getInput('VALUE').connection.connect(stringBlock.outputConnection);
printBlock.moveBy(80, 60);
refreshSource();
