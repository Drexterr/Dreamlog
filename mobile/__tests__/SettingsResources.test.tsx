import React from 'react';
import { fireEvent, render, waitFor } from '@testing-library/react-native';
import SettingsScreen from '../app/(tabs)/settings';
import { detectUserCountry } from '../src/services/region';
import { T } from '../src/testIDs';
jest.mock('expo-router',()=>({useRouter:()=>({push:jest.fn(),replace:jest.fn()})}));
jest.mock('../src/context/ThemeContext',()=>({useTheme:()=>({colors:require('../src/theme').THEMES.curious,theme:'curious',setTheme:jest.fn()})}));
jest.mock('../src/context/AuthContext',()=>({useAuth:()=>({isAuthenticated:false,requestAuth:jest.fn()})}));
jest.mock('../src/api/client',()=>({api:{},clearToken:jest.fn()}));
jest.mock('../src/lib/supabase',()=>({supabase:{auth:{signOut:jest.fn()}}}));
jest.mock('../src/services/region',()=>({cacheUserCountry:jest.fn(),detectUserCountry:jest.fn()}));
jest.mock('../src/services/updates',()=>({getUpdateSnapshot:jest.fn(),checkAndApplyUpdate:jest.fn()}));
const detect=detectUserCountry as jest.Mock;
beforeEach(()=>{jest.clearAllMocks();detect.mockReset();detect.mockResolvedValue('IN');});
describe('Settings resource localization',()=>{
  it('uses the resolved onboarding country when no signed-in profile exists',async()=>{
    const view=render(<SettingsScreen/>);fireEvent.press(view.getByTestId(T.settings.helpRow));
    await waitFor(()=>expect(view.getByText('iCall')).toBeTruthy());expect(detect).toHaveBeenCalled();
    fireEvent.press(view.getByText('Close'));expect(view.queryByText('iCall')).toBeNull();
  });
  it('keeps international resources available if country resolution fails',async()=>{
    detect.mockRejectedValueOnce(new Error('offline'));const view=render(<SettingsScreen/>);fireEvent.press(view.getByTestId(T.settings.helpRow));
    await waitFor(()=>expect(detect).toHaveBeenCalled());expect(view.getByText('Find A Helpline')).toBeTruthy();
    expect(view.queryByText('iCall')).toBeNull();
  });
  it('does not block opening resources while country detection is pending',()=>{
    detect.mockReturnValue(new Promise(()=>{}));const view=render(<SettingsScreen/>);fireEvent.press(view.getByTestId(T.settings.helpRow));
    expect(view.getByText('Find A Helpline')).toBeTruthy();expect(view.getByText('Close')).toBeTruthy();
  });
});
