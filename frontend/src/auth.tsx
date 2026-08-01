import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useState,
  type ReactNode,
} from 'react';
import * as api from './api';

type AuthState = {
  user: api.AuthUser | null;
  accessToken: string | null;
  loading: boolean;
  login: (email: string, password: string) => Promise<void>;
  register: (name: string, email: string, password: string) => Promise<void>;
  logout: () => Promise<void>;
};

const AuthContext = createContext<AuthState | null>(null);

export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<api.AuthUser | null>(null);
  const [accessToken, setAccessToken] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    let active = true;
    (async () => {
      try {
        const result = await api.refresh();
        if (!active) {
          return;
        }
        setAccessToken(result.accessToken);
        const current = await api.me(result.accessToken);
        if (active) {
          setUser(current.user);
        }
      } catch {
        if (active) {
          setAccessToken(null);
          setUser(null);
        }
      } finally {
        if (active) {
          setLoading(false);
        }
      }
    })();
    return () => {
      active = false;
    };
  }, []);

  const login = useCallback(async (email: string, password: string) => {
    const result = await api.login(email, password);
    setAccessToken(result.accessToken);
    setUser(result.user);
  }, []);

  const register = useCallback(async (name: string, email: string, password: string) => {
    await api.register(name, email, password);
    const result = await api.login(email, password);
    setAccessToken(result.accessToken);
    setUser(result.user);
  }, []);

  const logout = useCallback(async () => {
    if (accessToken) {
      try {
        await api.logout(accessToken);
      } catch {
        // ログアウトは冪等なので失敗してもローカル状態はクリアする
      }
    }
    setAccessToken(null);
    setUser(null);
  }, [accessToken]);

  const value = useMemo(
    () => ({ user, accessToken, loading, login, register, logout }),
    [user, accessToken, loading, login, register, logout],
  );

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}

export function useAuth() {
  const context = useContext(AuthContext);
  if (!context) {
    throw new Error('useAuth must be used within AuthProvider');
  }
  return context;
}
