/**
 * Jest configuration for Vango client tests
 */
export default {
    testEnvironment: 'jsdom',
    transform: {},
    moduleFileExtensions: ['js', 'mjs'],
    testMatch: ['**/test/**/*.test.js'],
    collectCoverageFrom: ['src/**/*.js'],
    injectGlobals: true,
    setupFilesAfterEnv: ['./test/setup.js'],
};
