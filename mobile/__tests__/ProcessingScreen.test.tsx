import React from 'react';
import { act, fireEvent, render } from '@testing-library/react-native';
import { Animated } from 'react-native';
import ProcessingScreen from '../app/processing/[id]';
import { api } from '../src/api/client';
import { writeMindfulSession } from '../src/services/health';
const mockReplace=jest.fn();
jest.mock('expo-router',()=>({useRouter:()=>({replace:mockReplace}),useLocalSearchParams:()=>({id:'entry'})}));
jest.mock('../src/context/ThemeContext',()=>({useTheme:()=>({colors:require('../src/theme').THEMES.curious})}));
jest.mock('../src/api/client',()=>({api:{getEntry:jest.fn()}}));
jest.mock('../src/services/health',()=>({writeMindfulSession:jest.fn().mockResolvedValue(undefined)}));
const entry=api.getEntry as jest.Mock;
beforeEach(()=>{jest.useFakeTimers();jest.clearAllMocks();entry.mockReset();entry.mockResolvedValue({status:'pending'});jest.spyOn(Animated,'loop').mockReturnValue({start:jest.fn(),stop:jest.fn(),reset:jest.fn()});});
afterEach(()=>{jest.useRealTimers();jest.restoreAllMocks();});
describe('journal processing screen',()=>{
  it('waits for the first poll interval and keeps pending UI',async()=>{
    const view=render(<ProcessingScreen/>);expect(entry).not.toHaveBeenCalled();await act(async()=>{await jest.advanceTimersByTimeAsync(3000);});
    expect(entry).toHaveBeenCalledWith('entry');expect(view.queryByText('Something went wrong')).toBeNull();
  });
  it('routes completed entry to reflection and writes recording duration',async()=>{
    entry.mockResolvedValue({status:'completed',duration_sec:15});render(<ProcessingScreen/>);
    await act(async()=>{await jest.advanceTimersByTimeAsync(3000);});expect(mockReplace).toHaveBeenCalledWith('/reflection/entry');
    const session=(writeMindfulSession as jest.Mock).mock.calls[0][0];expect(session.endDate-session.startDate).toBe(15000);
  });
  it('shows failed processing and returns home',async()=>{
    entry.mockResolvedValue({status:'failed'});const view=render(<ProcessingScreen/>);await act(async()=>{await jest.advanceTimersByTimeAsync(3000);});
    expect(view.getByText('Something went wrong')).toBeTruthy();fireEvent.press(view.getByText('Back home'));expect(mockReplace).toHaveBeenCalledWith('/(tabs)');
  });
  it('recovers from a transient poll failure',async()=>{
    entry.mockRejectedValueOnce(new Error('offline')).mockResolvedValue({status:'completed'});render(<ProcessingScreen/>);
    await act(async()=>{await jest.advanceTimersByTimeAsync(6000);});expect(mockReplace).toHaveBeenCalledTimes(1);
  });
  it('times out after the configured poll budget',async()=>{
    const view=render(<ProcessingScreen/>);await act(async()=>{await jest.advanceTimersByTimeAsync(183000);});
    expect(entry).toHaveBeenCalledTimes(60);expect(view.getByText('Something went wrong')).toBeTruthy();
  });
  it('does not overlap slow polling requests',async()=>{
    let finish!: (value:unknown)=>void;entry.mockImplementation(()=>new Promise(r=>{finish=r;}));render(<ProcessingScreen/>);
    await act(async()=>{await jest.advanceTimersByTimeAsync(6000);});expect(entry).toHaveBeenCalledTimes(1);
    await act(async()=>{finish({status:'pending'});});
  });
  it('ignores a late completed response after leaving the screen',async()=>{
    let finish!: (value:unknown)=>void;entry.mockImplementation(()=>new Promise(r=>{finish=r;}));const view=render(<ProcessingScreen/>);
    await act(async()=>{await jest.advanceTimersByTimeAsync(3000);});view.unmount();await act(async()=>{finish({status:'completed'});});
    expect(mockReplace).not.toHaveBeenCalled();expect(writeMindfulSession).not.toHaveBeenCalled();
  });
});
