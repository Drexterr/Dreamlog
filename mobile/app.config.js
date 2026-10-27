// Dynamic Expo config layered on app.json.
//
// google-services.json (Firebase) is deliberately kept out of git, so EAS cloud
// builds don't receive it. EAS provides it as a *file* environment variable
// (GOOGLE_SERVICES_JSON, a path to a temp copy on the build worker): use that
// when present, otherwise keep app.json's local path so local builds and
// `expo start` behave as before.
module.exports = ({ config }) => ({
  ...config,
  android: {
    ...config.android,
    googleServicesFile: process.env.GOOGLE_SERVICES_JSON || config.android?.googleServicesFile,
  },
});
