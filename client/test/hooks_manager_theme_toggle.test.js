import { HookManager } from '../src/hooks/manager.js';
import { jest } from '@jest/globals';

describe('HookManager + ThemeToggle', () => {
    beforeEach(() => {
        localStorage.clear();
        document.documentElement.classList.remove('dark');

        Object.defineProperty(window, 'matchMedia', {
            writable: true,
            value: jest.fn().mockImplementation(() => ({
                matches: false,
                addListener: jest.fn(),
                removeListener: jest.fn(),
            })),
        });
    });

    test('initializes ThemeToggle without data-hid (client-only hook)', () => {
        document.body.innerHTML = `
            <button data-hook="ThemeToggle" data-hook-config='{"storageKey":"theme"}'></button>
        `;

        const client = { options: { debug: false }, sendHookEvent: jest.fn() };
        const hooks = new HookManager(client);
        hooks.initializeFromDOM();

        const btn = document.querySelector('button');
        expect(document.documentElement.classList.contains('dark')).toBe(false);
        btn.click();
        expect(document.documentElement.classList.contains('dark')).toBe(true);
        expect(localStorage.getItem('theme')).toBe('dark');
    });

    test('destroyAll removes ThemeToggle listener for no-hid elements', () => {
        document.body.innerHTML = `
            <button data-hook="ThemeToggle" data-hook-config='{"storageKey":"theme"}'></button>
        `;

        const client = { options: { debug: false }, sendHookEvent: jest.fn() };
        const hooks = new HookManager(client);
        hooks.initializeFromDOM();

        const btn = document.querySelector('button');
        btn.click();
        expect(document.documentElement.classList.contains('dark')).toBe(true);

        hooks.destroyAll();
        btn.click();
        expect(document.documentElement.classList.contains('dark')).toBe(true);
    });
});
