// getSentryExpoConfig wraps expo/metro-config and adds source-map settings
// so release stack traces are symbolicated in Sentry.
const { getSentryExpoConfig } = require('@sentry/react-native/metro');

const config = getSentryExpoConfig(__dirname);

module.exports = config;
