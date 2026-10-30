import React, { createContext, useContext, useState, useEffect } from 'react';
import { User, UserRole, PatientProfile } from '@/lib/api/types';
import { authApi, tokenStorage } from '@/lib/api/client';

interface AuthContextType {
  user: User | null;
  profile: PatientProfile | null;
  token: string | null;
  isAuthenticated: boolean;
  isLoading: boolean;
  login: (email: string, password?: string) => Promise<User>;
  register: (payload: {
    email: string;
    password?: string;
    full_name: string;
    phone?: string;
    date_of_birth?: string;
    biological_sex?: string;
    blood_group?: string;
    emergency_contact_name?: string;
    emergency_contact_phone?: string;
  }) => Promise<boolean>;
  logout: () => void;
  switchDemoRole: (role: UserRole) => void;
  updateProfile: (profile: Partial<PatientProfile>) => void;
}

const AuthContext = createContext<AuthContextType | undefined>(undefined);

export const AuthProvider: React.FC<{ children: React.ReactNode }> = ({ children }) => {
  const [user, setUser] = useState<User | null>(null);
  const [profile, setProfile] = useState<PatientProfile | null>(null);
  const [token, setToken] = useState<string | null>(null);
  const [isLoading, setIsLoading] = useState(true);

  useEffect(() => {
    // Check initial token and user state
    const existingToken = tokenStorage.getAccessToken();
    const existingUser = tokenStorage.getUser();

    // Clean up any old cached default demo patient
    if (existingUser?.id === 'usr-patient-001' || existingUser?.email === 'aarav.sharma@example.com') {
      tokenStorage.clearAuth();
      localStorage.removeItem('panchakarma_patient_profile');
      setUser(null);
      setProfile(null);
      setToken(null);
    } else if (existingToken && existingUser) {
      setToken(existingToken);
      setUser(existingUser);
      if (existingUser.role === 'patient') {
        const storedProfile = localStorage.getItem('panchakarma_patient_profile');
        if (storedProfile) {
          try {
            setProfile(JSON.parse(storedProfile));
          } catch {
            setProfile(null);
          }
        }
      }
    } else {
      // Start unauthenticated - no default patient account
      setUser(null);
      setProfile(null);
      setToken(null);
    }
    setIsLoading(false);
  }, []);

  const login = async (email: string, password = ''): Promise<User> => {
    setIsLoading(true);
    try {
      const res = await authApi.login(email, password);
      setUser(res.user);
      setToken(res.access_token);
      if (res.profile) {
        setProfile(res.profile);
      }
      setIsLoading(false);
      return res.user;
    } catch (err: any) {
      setIsLoading(false);
      throw err;
    }
  };

  const register = async (payload: {
    email: string;
    password?: string;
    full_name: string;
    phone?: string;
    date_of_birth?: string;
    biological_sex?: string;
    blood_group?: string;
    emergency_contact_name?: string;
    emergency_contact_phone?: string;
  }): Promise<boolean> => {
    setIsLoading(true);
    try {
      const res = await authApi.register({
        email: payload.email,
        password: payload.password || 'SecurePassword123!',
        full_name: payload.full_name,
        phone: payload.phone || '+919876543210',
        date_of_birth: payload.date_of_birth,
        biological_sex: payload.biological_sex,
        blood_group: payload.blood_group,
        emergency_contact_name: payload.emergency_contact_name,
        emergency_contact_phone: payload.emergency_contact_phone,
      });
      setUser(res.user);
      setToken(res.access_token);
      if (res.profile) {
        setProfile(res.profile);
      }
      setIsLoading(false);
      return true;
    } catch (err: any) {
      setIsLoading(false);
      throw err;
    }
  };

  const logout = () => {
    authApi.logout().catch(() => {});
    tokenStorage.clearAuth();
    localStorage.removeItem('panchakarma_patient_profile');
    setUser(null);
    setProfile(null);
    setToken(null);
  };

  const switchDemoRole = async (_role: UserRole) => {
    // Deprecated: Role switching requires explicit login with valid credentials
  };

  const updateProfile = (updated: Partial<PatientProfile>) => {
    setProfile((prev) => {
      const next = prev ? { ...prev, ...updated } : (updated as PatientProfile);
      localStorage.setItem('panchakarma_patient_profile', JSON.stringify(next));
      return next;
    });
  };

  return (
    <AuthContext.Provider
      value={{
        user,
        profile,
        token,
        isAuthenticated: !!user,
        isLoading,
        login,
        register,
        logout,
        switchDemoRole,
        updateProfile,
      }}
    >
      {children}
    </AuthContext.Provider>
  );
};

export const useAuth = () => {
  const context = useContext(AuthContext);
  if (!context) {
    throw new Error('useAuth must be used within an AuthProvider');
  }
  return context;
};
