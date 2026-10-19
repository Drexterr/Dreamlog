import React from 'react';
import { act, fireEvent, render, waitFor } from '@testing-library/react-native';
import FollowUpScreen from '../app/followup/[id]';
import { api } from '../src/api/client';
import { T } from '../src/testIDs';
const mockReplace=jest.fn(), mockBack=jest.fn();
jest.mock('expo-router',()=>({useRouter:()=>({replace:mockReplace,back:mockBack}),useLocalSearchParams:()=>({id:'entry'})}));
jest.mock('../src/context/ThemeContext',()=>({useTheme:()=>({colors:require('../src/theme').THEMES.curious})}));
jest.mock('../src/api/client',()=>({api:{getOrCreateConversation:jest.fn(),sendConversationMessage:jest.fn()}}));
const get=api.getOrCreateConversation as jest.Mock,send=api.sendConversationMessage as jest.Mock;
const conversation=(extra={})=>({id:'conversation',turn_count:0,is_closed:false,messages:[],...extra});
beforeEach(()=>{jest.useFakeTimers();jest.clearAllMocks();get.mockReset();send.mockReset();get.mockResolvedValue(conversation());send.mockResolvedValue(conversation({turn_count:1,messages:[{id:'u',role:'user',content:'My message'},{id:'a',role:'assistant',content:'A grounded reply'}]}));});
afterEach(()=>{act(()=>{jest.runOnlyPendingTimers();});jest.useRealTimers();});
async function screen(){const view=render(<FollowUpScreen/>);await waitFor(()=>expect(view.getByTestId(T.followup.closeButton)).toBeTruthy());return view;}
describe('follow-up replies',()=>{
  it('loads conversation and displays seed prompt',async()=>{const view=await screen();expect(get).toHaveBeenCalledWith('entry');expect(view.getByText('What would you like to explore further from your reflection?')).toBeTruthy();});
  it('loads persisted history instead of a seed',async()=>{get.mockResolvedValue(conversation({messages:[{id:'old',role:'assistant',content:'Saved reply'}]}));const view=await screen();expect(view.getByText('Saved reply')).toBeTruthy();});
  it('sends trimmed content and displays returned reply',async()=>{const view=await screen();fireEvent.changeText(view.getByTestId(T.followup.input),' My message ');fireEvent.press(view.getByTestId(T.followup.sendButton));await waitFor(()=>expect(view.getByText('A grounded reply')).toBeTruthy());expect(send).toHaveBeenCalledWith('conversation','My message');expect(view.getByText('2 exchanges left')).toBeTruthy();});
  it('rejects whitespace-only keyboard messages',async()=>{const view=await screen();fireEvent.changeText(view.getByTestId(T.followup.input),'   ');fireEvent(view.getByTestId(T.followup.input),'submitEditing');expect(send).not.toHaveBeenCalled();});
  it('restores draft after provider failure and removes optimistic text',async()=>{send.mockRejectedValueOnce(new Error('offline'));const view=await screen();fireEvent.changeText(view.getByTestId(T.followup.input),'Retry me');fireEvent.press(view.getByTestId(T.followup.sendButton));await waitFor(()=>expect(view.getByTestId(T.followup.input).props.value).toBe('Retry me'));expect(view.queryByText('Retry me')).toBeNull();});
  it('shows closed banner and returns home',async()=>{get.mockResolvedValue(conversation({turn_count:3,is_closed:true}));const view=await screen();expect(view.queryByTestId(T.followup.input)).toBeNull();fireEvent.press(view.getByTestId(T.followup.goodnight));expect(mockReplace).toHaveBeenCalledWith('/(tabs)');});
  it('does not send a fourth turn through keyboard if closure flag is delayed',async()=>{get.mockResolvedValue(conversation({turn_count:3,is_closed:false}));const view=await screen();fireEvent.changeText(view.getByTestId(T.followup.input),'Fourth turn');fireEvent(view.getByTestId(T.followup.input),'submitEditing');expect(send).not.toHaveBeenCalled();});
  it('Done returns to previous screen',async()=>{const view=await screen();fireEvent.press(view.getByTestId(T.followup.closeButton));expect(mockBack).toHaveBeenCalledTimes(1);});
});
