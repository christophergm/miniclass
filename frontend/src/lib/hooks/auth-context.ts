import { createContext, type ReactNode } from "react";

import type { AuthClient, AuthError, Session, SessionEndedReason } from "@/lib/auth";

export type AuthActionResult = {
  session: Session | null;
};

export type AuthContextValue = {
  authConfigured: boolean;
  authError: AuthError | null;
  isLoading: boolean;
  session: Session | null;
  sessionEndedReason: SessionEndedReason | null;
  signIn: (email: string, password: string) => Promise<AuthActionResult>;
  signUp: (email: string, password: string) => Promise<AuthActionResult>;
  resetPassword: (email: string) => Promise<void>;
  signOut: () => Promise<void>;
};

export const AuthContext = createContext<AuthContextValue | null>(null);

export type AuthProviderProps = {
  client?: AuthClient | null;
  children: ReactNode;
};
