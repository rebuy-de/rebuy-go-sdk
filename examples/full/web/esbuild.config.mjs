import * as esbuild from 'esbuild'
import fs from 'node:fs'

await esbuild.build({
  entryPoints: [
     'src/index.js', 'src/index.css',
  ],
  bundle: true,
  minify: true,
  sourcemap: true,
  outdir: 'dist/',
  format: 'esm',
  loader: {
    '.woff2': 'file',
    '.ttf': 'file'
  },
})

fs.cpSync('src/www', 'dist', {recursive: true});

// The HTMX stuff does not deal well with ESM bundling. It is not needed tho,
// therefore we copy the assets manually and link them directly in the <head>.
const scripts = [
  'hyperscript.org/dist/_hyperscript.min.js',
  'hyperscript.org/dist/template.js',
  'htmx.org/dist/htmx.min.js',
  'idiomorph/dist/idiomorph-ext.min.js',
];

scripts.forEach((file) => {
    fs.cpSync(`node_modules/${file}`, `dist/${file}`, {recursive: true});
});
