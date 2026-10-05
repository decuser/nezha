'use strict';

function nezhaEscapeString(value) {
  // Nezha accepts quoted strings with backslash escapes. JSON string quoting
  // gives us the same compact spelling for the characters needed here.
  return JSON.stringify(value);
}

function nezhaEscapeRune(value) {
  const escaped = value
    .replace(/\\/g, '\\\\')
    .replace(/'/g, "\\'")
    .replace(/\n/g, '\\n')
    .replace(/\r/g, '\\r')
    .replace(/\t/g, '\\t');
  return "'" + escaped + "'";
}

function nezhaValue(block) {
  if (!block) return null;

  switch (block.type) {
    case 'nezha_number':
      return block.getFieldValue('VALUE');
    case 'nezha_string':
      return nezhaEscapeString(block.getFieldValue('VALUE'));
    case 'nezha_rune':
      return nezhaEscapeRune(block.getFieldValue('VALUE'));
    case 'nezha_symbol':
      return ':' + block.getFieldValue('VALUE');
    case 'nezha_boolean':
      return block.getFieldValue('VALUE');
    default:
      throw new Error('unsupported value block: ' + block.type);
  }
}

function nezhaStatement(block) {
  if (block.type !== 'nezha_println') {
    throw new Error('unsupported statement block: ' + block.type);
  }

  const value = nezhaValue(block.getInputTargetBlock('VALUE'));
  if (value === null) {
    return '# println has an empty value socket\n';
  }
  return 'println(' + value + ')\n';
}

function nezhaSource(workspace) {
  let source = '';
  const tops = workspace.getTopBlocks(true);

  for (const top of tops) {
    if (top.outputConnection) continue;

    let block = top;
    while (block) {
      source += nezhaStatement(block);
      block = block.getNextBlock();
    }
  }

  return source;
}
