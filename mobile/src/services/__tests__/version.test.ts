import { Linking, Platform } from 'react-native';
import Constants from 'expo-constants';
import { api } from '../../api/client';
import { checkForceUpdate, isVersionBelow, openStoreListing } from '../version';
jest.mock('../../api/client', () => ({ api: { getVersion: jest.fn() } }));
jest.mock('expo-constants', () => ({ __esModule: true, default: { expoConfig: { version: '1.0.0' } } }));
const getVersion=api.getVersion as jest.Mock;
const info={ minimum_version:'1.1.0', android_store_url:'https://fixture.invalid/android', ios_store_url:'https://fixture.invalid/ios' };
beforeEach(()=>{ jest.clearAllMocks(); getVersion.mockReset(); getVersion.mockResolvedValue(info); (Constants.expoConfig as {version:string}).version='1.0.0'; });
describe('app version gate',()=>{
  it.each([
    ['1.0.0','1.0.0',false],['1.0','1.0.0',false],['1.0.9','1.1',true],['2.0','1.99',false],['1.10','1.9',false],['1','1.0.1',true],['bad','1',true],['1.0.0.1','1.0.0',false],
  ])('compares %s to %s',(current,minimum,want)=>{ expect(isVersionBelow(current,minimum)).toBe(want); });
  it('gates an older installed version',async()=>{ expect(await checkForceUpdate()).toEqual(info); });
  it('does not gate a newer installed version',async()=>{ (Constants.expoConfig as {version:string}).version='2.0.0'; expect(await checkForceUpdate()).toBeNull(); });
  it('fails open on API errors',async()=>{ getVersion.mockRejectedValue(new Error('offline')); expect(await checkForceUpdate()).toBeNull(); });
  it('skips remote lookup if installed version is unavailable',async()=>{
    (Constants.expoConfig as {version?:string}).version=undefined; expect(await checkForceUpdate()).toBeNull(); expect(getVersion).not.toHaveBeenCalled();
  });
  it('uses Android market URL when supported',async()=>{
    jest.replaceProperty(Platform,'OS','android'); const can=jest.spyOn(Linking,'canOpenURL').mockResolvedValue(true); const open=jest.spyOn(Linking,'openURL').mockResolvedValue(undefined);
    await openStoreListing(info); expect(open).toHaveBeenCalledWith('market://details?id=com.talktoode.app'); can.mockRestore();open.mockRestore();jest.restoreAllMocks();
  });
  it('falls back to provided web listing when store app is missing',async()=>{
    jest.replaceProperty(Platform,'OS','android'); const can=jest.spyOn(Linking,'canOpenURL').mockResolvedValue(false); const open=jest.spyOn(Linking,'openURL').mockResolvedValue(undefined);
    await openStoreListing(info); expect(open).toHaveBeenCalledWith(info.android_store_url); can.mockRestore();open.mockRestore();jest.restoreAllMocks();
  });
  it('opens only iOS listing on iOS',async()=>{
    jest.replaceProperty(Platform,'OS','ios'); const open=jest.spyOn(Linking,'openURL').mockResolvedValue(undefined);
    await openStoreListing(info); expect(open).toHaveBeenCalledWith(info.ios_store_url);open.mockRestore();jest.restoreAllMocks();
  });
  it('store launch rejection does not crash the update screen',async()=>{
    jest.replaceProperty(Platform,'OS','ios'); const open=jest.spyOn(Linking,'openURL').mockRejectedValue(new Error('no handler'));
    await expect(openStoreListing(info)).resolves.toBeUndefined(); open.mockRestore();jest.restoreAllMocks();
  });
});
