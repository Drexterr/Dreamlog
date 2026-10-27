#!/usr/bin/env node
/**
 * EAS "eas-build-post-install" hook: puts Firebase's google-services.json where
 * Gradle looks for it.
 *
 * mobile/android is a committed (bare) project, so EAS does not regenerate it
 * from app.json, and google-services.json is deliberately gitignored (Firebase
 * config stays out of the repo). EAS cloud builds therefore had no file and the
 * Android build failed in Gradle (:app:processReleaseGoogleServices). The file
 * is stored in EAS as the protected *file* variable GOOGLE_SERVICES_JSON, whose
 * value on the build worker is the path of a temporary copy; this copies it to
 * android/app/google-services.json.
 *
 * Local runs (no EAS_BUILD) and iOS builds are left alone. An Android cloud
 * build with no file and no variable fails here with a clear message instead
 * of four minutes later inside Gradle.
 */
const fs = require('fs');
const path = require('path');

const DEST = path.join(__dirname, '..', 'android', 'app', 'google-services.json');

function copyGoogleServices(env = process.env, dest = DEST, log = console.log) {
  if (env.EAS_BUILD !== 'true' || env.EAS_BUILD_PLATFORM === 'ios') {
    return { status: 'skipped' };
  }
  const source = env.GOOGLE_SERVICES_JSON;
  if (source) {
    if (!fs.existsSync(source)) {
      throw new Error(`GOOGLE_SERVICES_JSON points to ${source}, which does not exist`);
    }
    // Fail early on a broken upload instead of inside Gradle.
    JSON.parse(fs.readFileSync(source, 'utf8'));
    fs.mkdirSync(path.dirname(dest), { recursive: true });
    fs.copyFileSync(source, dest);
    log('google-services.json copied from the GOOGLE_SERVICES_JSON file variable');
    return { status: 'copied' };
  }
  if (fs.existsSync(dest)) return { status: 'present' };
  throw new Error(
    'android/app/google-services.json is missing and the GOOGLE_SERVICES_JSON EAS file variable is not set. ' +
      'Create it with: eas env:create --name GOOGLE_SERVICES_JSON --type file --value ./google-services.json --environment <env>',
  );
}

module.exports = { copyGoogleServices };

if (require.main === module) {
  try {
    copyGoogleServices();
  } catch (err) {
    console.error(err.message);
    process.exit(1);
  }
}
