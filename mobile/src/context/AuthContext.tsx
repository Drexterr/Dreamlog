import { createContext, useContext } from 'react';

export type AuthContextValue = {
  isAuthenticated: boolean;
  requestAuth: (afterAuth: () => void) => void;
};

export const AuthContext = createContext<AuthContextValue>({
  isAuthenticated: false,
  requestAuth: () => {},
});

export function useAuth(): AuthContextValue {
  return useContext(AuthContext);
}
