/**
 * Vango Client Build Script
 *
 * Uses esbuild to bundle the thin client into a single file.
 */

import * as esbuild from 'esbuild';
import fs from 'fs';
import { gzipSync } from 'zlib';

const isDev = process.argv.includes('--dev');
const isWatch = process.argv.includes('--watch');

const options = {
    entryPoints: ['src/index.js'],
    bundle: true,
    outfile: isDev ? 'dist/vango.js' : 'dist/vango.min.js',
    minify: !isDev,
    sourcemap: isDev,
    target: ['es2018'],
    format: 'iife',
    globalName: 'Vango',
    // Ensure we don't include Node.js built-ins
    platform: 'browser',
    // Keep names readable in dev mode
    keepNames: isDev,
};

async function build() {
    try {
        if (isWatch) {
            const ctx = await esbuild.context(options);
            await ctx.watch();
            console.log('Watching for changes...');
        } else {
            const result = await esbuild.build(options);

            const stat = fs.statSync(options.outfile);
            const content = fs.readFileSync(options.outfile);
            const gzipped = gzipSync(content);

            console.log(`Built ${options.outfile}`);
            console.log(`  Raw size:     ${(stat.size / 1024).toFixed(2)} KB`);
            console.log(`  Gzipped size: ${(gzipped.length / 1024).toFixed(2)} KB`);

            if (!isDev && gzipped.length > 15 * 1024) {
                console.warn('\n⚠️  WARNING: Gzipped size exceeds 15KB target!');
            } else if (!isDev) {
                console.log('\n✅ Size is within 15KB target');
            }
        }
    } catch (error) {
        console.error('Build failed:', error);
        process.exit(1);
    }
}

build();
