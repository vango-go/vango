import { jest, describe, test, expect, beforeAll, afterAll } from '@jest/globals';

let VangoClient;

beforeAll(async () => {
    if (typeof window !== 'undefined') {
        window.__vango__ = true;
    }
    const mod = await import('../src/index.js');
    VangoClient = mod.default;
});

afterAll(() => {
    if (typeof window !== 'undefined') {
        delete window.__vango__;
    }
});

describe('Auth broadcast handling', () => {
    test('handleAuthBroadcast schedules reload for expired payload', () => {
        const schedule = jest.fn();
        const ctx = {
            _scheduleAuthReload: schedule,
        };

        VangoClient.prototype._handleAuthBroadcast.call(ctx, { type: 'expired', reason: 1 });

        expect(schedule).toHaveBeenCalledTimes(1);
    });

    test('handleAuthBroadcast ignores unrelated payload', () => {
        const schedule = jest.fn();
        const ctx = {
            _scheduleAuthReload: schedule,
        };

        VangoClient.prototype._handleAuthBroadcast.call(ctx, { type: 'other' });

        expect(schedule).not.toHaveBeenCalled();
    });

});
