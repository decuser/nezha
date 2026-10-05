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
    case 'nezha_read':
      return 'read()';
    case 'nezha_name': {
      const name = block.getFieldValue('NAME');
      return name || null;
    }
    case 'nezha_equal': {
      const left = nezhaValue(block.getInputTargetBlock('LEFT'));
      const right = nezhaValue(block.getInputTargetBlock('RIGHT'));
      if (left === null || right === null) return null;
      return 'equal(' + left + ', ' + right + ')';
    }
    case 'nezha_not': {
      const value = nezhaValue(block.getInputTargetBlock('VALUE'));
      if (value === null) return null;
      return '(not ' + value + ')';
    }
    case 'nezha_binary': {
      const left = nezhaValue(block.getInputTargetBlock('LEFT'));
      const right = nezhaValue(block.getInputTargetBlock('RIGHT'));
      if (left === null || right === null) return null;
      return '(' + left + ' ' + block.getFieldValue('OP') + ' ' + right + ')';
    }
    default:
      throw new Error('unsupported value block: ' + block.type);
  }
}

function nezhaIndent(depth) {
  return '\t'.repeat(depth);
}

function nezhaStatementChain(block, depth) {
  let source = '';
  let current = block;
  while (current) {
    source += nezhaStatement(current, depth);
    current = current.getNextBlock();
  }
  return source;
}

function nezhaStatement(block, depth) {
  const indent = nezhaIndent(depth);

  switch (block.type) {
    case 'nezha_let': {
      const value = nezhaValue(block.getInputTargetBlock('VALUE'));
      if (value === null) {
        return indent + '# let ' + block.getFieldValue('NAME') + ' has an empty value socket\n';
      }
      return indent + 'let ' + block.getFieldValue('NAME') + ' = ' + value + '\n';
    }
    case 'nezha_print': {
      const value = nezhaValue(block.getInputTargetBlock('VALUE'));
      if (value === null) {
        return indent + '# print has an empty value socket\n';
      }
      return indent + 'print(' + value + ')\n';
    }
    case 'nezha_assign': {
      const name = block.getFieldValue('NAME');
      if (!name) {
        return indent + '# assignment has no earlier binding\n';
      }
      const value = nezhaValue(block.getInputTargetBlock('VALUE'));
      if (value === null) {
        return indent + '# assignment to ' + name + ' has an empty value socket\n';
      }
      return indent + name + ' = ' + value + '\n';
    }
    case 'nezha_println': {
      const value = nezhaValue(block.getInputTargetBlock('VALUE'));
      if (value === null) {
        return indent + '# println has an empty value socket\n';
      }
      return indent + 'println(' + value + ')\n';
    }
    case 'nezha_if': {
      const condition = nezhaValue(block.getInputTargetBlock('COND'));
      if (condition === null) {
        return indent + '# if has an empty condition socket\n';
      }
      let source = indent + 'if ' + condition + ' {\n';
      const body = block.getInputTargetBlock('DO');
      if (body) source += nezhaStatementChain(body, depth + 1);
      source += indent + '}\n';
      return source;
    }
    case 'nezha_while': {
      const condition = nezhaValue(block.getInputTargetBlock('COND'));
      if (condition === null) {
        return indent + '# while has an empty condition socket\n';
      }
      let source = indent + 'while ' + condition + ' {\n';
      const body = block.getInputTargetBlock('DO');
      if (body) source += nezhaStatementChain(body, depth + 1);
      source += indent + '}\n';
      return source;
    }
    default:
      throw new Error('unsupported statement block: ' + block.type);
  }
}

function nezhaSource(workspace) {
  let source = '';
  const tops = workspace.getTopBlocks(true);

  for (const top of tops) {
    if (top.outputConnection) continue;
    source += nezhaStatementChain(top, 0);
  }

  return source;
}
