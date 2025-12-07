/**
 * Jest configuration for Vango client tests
 */
export default {
    testEnvironment: 'node',
    transform: {},
    moduleFileExtensions: ['js', 'mjs'],
    testMatch: ['**/test/**/*.test.js'],
    collectCoverageFrom: ['src/**/*.js'],
};
