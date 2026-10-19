import ts from '../../mobile/node_modules/typescript/lib/typescript.js';
import { readdir, readFile, mkdir, writeFile } from 'node:fs/promises';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '../..');
const out = path.join(root, 'test-results/control-inventory');
await mkdir(out, { recursive: true });
const registry = ts.createSourceFile('testIDs.ts', await readFile(path.join(root, 'mobile/src/testIDs.ts'), 'utf8'), ts.ScriptTarget.Latest, true);
const ids = new Map();
function readObject(object, prefix) {
  for (const property of object.properties) {
    if (!ts.isPropertyAssignment(property)) continue;
    const key = `${prefix}.${property.name.getText(registry).replaceAll("'", '').replaceAll('"', '')}`;
    if (ts.isStringLiteral(property.initializer)) ids.set(key, property.initializer.text);
    if (ts.isObjectLiteralExpression(property.initializer)) readObject(property.initializer, key);
  }
}
function findRegistry(node) {
  if (ts.isVariableDeclaration(node) && node.name.getText(registry) === 'T' && ts.isObjectLiteralExpression(node.initializer)) readObject(node.initializer, 'T');
  ts.forEachChild(node, findRegistry);
}
findRegistry(registry);
const controls = [];
async function scan(dir) {
  for (const entry of await readdir(dir, { withFileTypes: true })) {
    if (entry.name === '__tests__') continue;
    const file = path.join(dir, entry.name);
    if (entry.isDirectory()) { await scan(file); continue; }
    if (!file.endsWith('.tsx')) continue;
    const source = ts.createSourceFile(file, await readFile(file, 'utf8'), ts.ScriptTarget.Latest, true, ts.ScriptKind.TSX);
    function visit(node) {
      if (ts.isJsxOpeningElement(node) || ts.isJsxSelfClosingElement(node)) {
        const attrs = node.attributes.properties.filter(ts.isJsxAttribute);
        const events = attrs.filter(a => ['onPress', 'onLongPress', 'onValueChange', 'onSubmitEditing', 'onChangeText'].includes(a.name.getText(source)));
        if (events.length) {
          const id = attrs.find(a => a.name.getText(source) === 'testID');
          const expression = id?.initializer && ts.isJsxExpression(id.initializer) ? id.initializer.expression?.getText(source) : null;
          const literal = id?.initializer && ts.isStringLiteral(id.initializer) ? id.initializer.text : null;
          controls.push({
            file: path.relative(root, file).replaceAll('\\', '/'),
            line: source.getLineAndCharacterOfPosition(node.getStart(source)).line + 1,
            component: node.tagName.getText(source),
            events: events.map(a => a.name.getText(source)),
            testID: literal || ids.get(expression) || null,
            testIDExpression: expression,
            accessibilityLabel: attrs.find(a => a.name.getText(source) === 'accessibilityLabel')?.initializer?.getText(source) || null,
            deviceStatus: 'not_executed',
          });
        }
      }
      ts.forEachChild(node, visit);
    }
    visit(source);
  }
}
await scan(path.join(root, 'mobile/app'));
await scan(path.join(root, 'mobile/src/components'));
const missing = controls.filter(c => !c.testID && !c.testIDExpression);
await writeFile(path.join(out, 'controls.json'), JSON.stringify({ controls, counts: { total: controls.length, missingTestID: missing.length }, note: 'Source controls are not runtime button instances. Conditional states, repeated rows and platform-specific controls require separate device validation.' }, null, 2));
await writeFile(path.join(out, 'SUMMARY.md'), `# Control Inventory\n\n${controls.length} interactive source controls; ${missing.length} have no explicit testID. No control is marked device-tested by static inspection.\n\n| File | Line | Component | ID |\n| --- | --- | --- | --- |\n${missing.map(c => `| ${c.file} | ${c.line} | ${c.component} | missing |`).join('\n')}\n`);
console.log(`Controls: ${controls.length}; missing IDs: ${missing.length}. Report: ${out}`);
