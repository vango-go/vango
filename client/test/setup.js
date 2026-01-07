/**
 * Jest Setup File
 *
 * Provides necessary polyfills for jsdom environment.
 */

import { TextEncoder, TextDecoder } from 'util';

// Polyfill TextEncoder/TextDecoder for jsdom
global.TextEncoder = TextEncoder;
global.TextDecoder = TextDecoder;
