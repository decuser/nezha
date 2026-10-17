'use strict';

function nezhaTextField(defaultValue) {
  const field = new Blockly.FieldTextInput(defaultValue);
  field.setSpellcheck(false);
  return field;
}

function nezhaNameField(defaultValue) {
  const field = nezhaTextField(defaultValue);
  field.setValidator(function(value) {
    return /^[A-Za-z_][A-Za-z0-9_]*$/.test(value) ? value : null;
  });
  return field;
}

function nezhaContainingStatement(block) {
  let current = block;
  while (current) {
    if (current.previousConnection || current.nextConnection) return current;
    current = current.getParent();
  }
  return null;
}

function nezhaPriorBindings(block) {
  let statement = nezhaContainingStatement(block);
  if (!statement) return [];

  const names = [];
  const seen = new Set();

  while (statement) {
    let previous = statement.getPreviousBlock();
    while (previous) {
      if (previous.type === 'nezha_let') {
        const name = previous.getFieldValue('NAME');
        if (name && !seen.has(name)) {
          names.push(name);
          seen.add(name);
        }
      }
      previous = previous.getPreviousBlock();
    }

    const parent = statement.getParent();
    if (!parent) break;
    const outer = nezhaContainingStatement(parent);
    if (!outer || outer === statement) break;
    statement = outer;
  }

  names.reverse();
  return names;
}

function nezhaBindingOptions() {
  const names = nezhaPriorBindings(this.getSourceBlock());
  if (names.length === 0) return [['(unbound)', '']];
  return names.map(function(name) { return [name, name]; });
}

function nezhaBindingField() {
  return new Blockly.FieldDropdown(nezhaBindingOptions);
}

Blockly.Blocks['nezha_number'] = {
  init: function() {
    const field = nezhaTextField('42');
    field.setValidator(function(value) {
      return /^[0-9]+(?:\.[0-9]+)?(?:\/[0-9]+)?$/.test(value) ? value : null;
    });
    this.appendDummyInput().appendField('number').appendField(field, 'VALUE');
    this.setOutput(true, 'NezhaValue');
    this.setColour(220);
  }
};

Blockly.Blocks['nezha_string'] = {
  init: function() {
    this.appendDummyInput()
      .appendField('string')
      .appendField(nezhaTextField('hello'), 'VALUE');
    this.setOutput(true, 'NezhaValue');
    this.setColour(160);
  }
};

Blockly.Blocks['nezha_rune'] = {
  init: function() {
    const field = nezhaTextField('N');
    field.setValidator(function(value) {
      return Array.from(value).length === 1 ? value : null;
    });
    this.appendDummyInput().appendField('rune').appendField(field, 'VALUE');
    this.setOutput(true, 'NezhaValue');
    this.setColour(40);
  }
};

Blockly.Blocks['nezha_symbol'] = {
  init: function() {
    const field = nezhaTextField('ready');
    field.setValidator(function(value) {
      return /^[A-Za-z_][A-Za-z0-9_]*$/.test(value) ? value : null;
    });
    this.appendDummyInput().appendField('symbol :').appendField(field, 'VALUE');
    this.setOutput(true, 'NezhaValue');
    this.setColour(290);
  }
};

Blockly.Blocks['nezha_boolean'] = {
  init: function() {
    this.appendDummyInput()
      .appendField('boolean')
      .appendField(new Blockly.FieldDropdown([
        ['true', 'true'],
        ['false', 'false']
      ]), 'VALUE');
    this.setOutput(true, 'NezhaValue');
    this.setColour(20);
  }
};

Blockly.Blocks['nezha_read'] = {
  init: function() {
    this.appendDummyInput().appendField('read');
    this.setOutput(true, 'NezhaValue');
    this.setColour(120);
  }
};

Blockly.Blocks['nezha_name'] = {
  init: function() {
    this.appendDummyInput()
      .appendField('name')
      .appendField(nezhaBindingField(), 'NAME');
    this.setOutput(true, 'NezhaValue');
    this.setColour(260);
  }
};

Blockly.Blocks['nezha_let'] = {
  init: function() {
    this.appendValueInput('VALUE')
      .setCheck('NezhaValue')
      .appendField('let')
      .appendField(nezhaNameField('x'), 'NAME')
      .appendField('=');
    this.setPreviousStatement(true, 'NezhaStatement');
    this.setNextStatement(true, 'NezhaStatement');
    this.setColour(260);
  }
};

Blockly.Blocks['nezha_println'] = {
  init: function() {
    this.appendValueInput('VALUE')
      .setCheck('NezhaValue')
      .appendField('println');
    this.setPreviousStatement(true, 'NezhaStatement');
    this.setNextStatement(true, 'NezhaStatement');
    this.setColour(120);
  }
};

Blockly.Blocks['nezha_print'] = {
  init: function() {
    this.appendValueInput('VALUE')
      .setCheck('NezhaValue')
      .appendField('print');
    this.setPreviousStatement(true, 'NezhaStatement');
    this.setNextStatement(true, 'NezhaStatement');
    this.setColour(120);
  }
};

Blockly.Blocks['nezha_assign'] = {
  init: function() {
    this.appendValueInput('VALUE')
      .setCheck('NezhaValue')
      .appendField(nezhaBindingField(), 'NAME')
      .appendField('=');
    this.setPreviousStatement(true, 'NezhaStatement');
    this.setNextStatement(true, 'NezhaStatement');
    this.setColour(260);
  }
};

Blockly.Blocks['nezha_if_plain'] = {
  init: function() {
    this.appendValueInput('COND')
      .setCheck('NezhaValue')
      .appendField('if');
    this.appendStatementInput('DO')
      .setCheck('NezhaStatement');
    this.setPreviousStatement(true, 'NezhaStatement');
    this.setNextStatement(true, 'NezhaStatement');
    this.setColour(120);
  }
};

Blockly.Blocks['nezha_if'] = {
  init: function() {
    this.appendValueInput('COND')
      .setCheck('NezhaValue')
      .appendField('if');
    this.appendStatementInput('DO')
      .setCheck('NezhaStatement');
    this.appendStatementInput('ELSE')
      .setCheck('NezhaStatement')
      .appendField('else');
    this.setPreviousStatement(true, 'NezhaStatement');
    this.setNextStatement(true, 'NezhaStatement');
    this.setColour(120);
  }
};

Blockly.Blocks['nezha_equal'] = {
  init: function() {
    this.appendValueInput('LEFT')
      .setCheck('NezhaValue')
      .appendField('equal');
    this.appendValueInput('RIGHT')
      .setCheck('NezhaValue');
    this.setInputsInline(true);
    this.setOutput(true, 'NezhaValue');
    this.setColour(210);
  }
};

Blockly.Blocks['nezha_not'] = {
  init: function() {
    this.appendValueInput('VALUE')
      .setCheck('NezhaValue')
      .appendField('not');
    this.setOutput(true, 'NezhaValue');
    this.setColour(210);
  }
};

Blockly.Blocks['nezha_negative'] = {
  init: function() {
    this.appendValueInput('VALUE')
      .setCheck('NezhaValue')
      .appendField('-');
    this.setOutput(true, 'NezhaValue');
    this.setColour(210);
  }
};

Blockly.Blocks['nezha_while'] = {
  init: function() {
    this.appendValueInput('COND')
      .setCheck('NezhaValue')
      .appendField('while');
    this.appendStatementInput('DO')
      .setCheck('NezhaStatement');
    this.setPreviousStatement(true, 'NezhaStatement');
    this.setNextStatement(true, 'NezhaStatement');
    this.setColour(120);
  }
};

Blockly.Blocks['nezha_binary'] = {
  init: function() {
    this.appendValueInput('LEFT').setCheck('NezhaValue');
    this.appendDummyInput()
      .appendField(new Blockly.FieldDropdown([
        ['+', '+'],
        ['-', '-'],
        ['*', '*'],
        ['/', '/'],
        ['<', '<'],
        ['>', '>'],
        ['<=', '<='],
        ['>=', '>='],
        ['and', 'and'],
        ['or', 'or']
      ]), 'OP');
    this.appendValueInput('RIGHT').setCheck('NezhaValue');
    this.setInputsInline(true);
    this.setOutput(true, 'NezhaValue');
    this.setColour(210);
  }
};
