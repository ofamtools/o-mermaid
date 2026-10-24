import { THEMES } from 'beautiful-mermaid';
const result = await Bun.build({ entrypoints: ['renderer/entry.js'], target: 'browser', minify: true });
if (!result.success) throw new AggregateError(result.logs, 'Renderer bundle failed');
await Bun.write('renderer/beautiful-mermaid.js', result.outputs[0]);
await Bun.write('renderer/themes.json', JSON.stringify({
  default: { bg: '#f5f5f5', fg: '#333333' },
  solarized: { bg: '#fdf6e3', fg: '#657b83' },
  ...THEMES,
}, null, 2) + '\n');
for (const [source, destination] of [
  ['beautiful-mermaid/LICENSE', 'beautiful-mermaid.txt'],
  ['elkjs/LICENSE.md', 'elkjs.txt'],
  ['entities/LICENSE', 'entities.txt'],
]) {
  await Bun.write(`renderer/licenses/${destination}`, Bun.file(`node_modules/${source}`));
}
