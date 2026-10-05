'use strict';

Blockly.Blocks['nezha_number'] = {
  init: function() {
    const field = new Blockly.FieldTextInput('42');
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
      .appendField(new Blockly.FieldTextInput('hello'), 'VALUE');
    this.setOutput(true, 'NezhaValue');
    this.setColour(160);
  }
};

Blockly.Blocks['nezha_rune'] = {
  init: function() {
    const field = new Blockly.FieldTextInput('N');
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
    const field = new Blockly.FieldTextInput('ready');
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
