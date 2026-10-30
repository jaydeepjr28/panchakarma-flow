import React, { useState } from 'react';
import { useNavigate, useLocation, Link } from 'react-router-dom';
import { useAuth } from '@/context/AuthContext';
import { NeuCard } from '@/components/ui/NeuCard';
import { NeuButton } from '@/components/ui/NeuButton';
import { NeuInput } from '@/components/ui/NeuInput';
import { UserRole } from '@/lib/api/types';
import {
  Mail,
  Lock,
  Eye,
  EyeOff,
  Sparkles,
  ArrowRight,
  ShieldCheck,
} from 'lucide-react';

export const LoginPage: React.FC = () => {
  const { login, isLoading } = useAuth();
  const navigate = useNavigate();
  const location = useLocation();

  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [showPassword, setShowPassword] = useState(false);
  const [errorMsg, setErrorMsg] = useState('');

  function getRoleHome(role: UserRole) {
    switch (role) {
      case 'clinic_admin':
      case 'super_admin':
        return '/clinic/dashboard';
      case 'therapist':
      case 'doctor':
        return '/therapist/dashboard';
      case 'patient':
      default:
        return '/patient/dashboard';
    }
  }

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setErrorMsg('');
    try {
      const loggedUser = await login(email, password);
      const destination = (location.state as any)?.from?.pathname || getRoleHome(loggedUser.role);
      navigate(destination, { replace: true });
    } catch (err: any) {
      setErrorMsg(err.message || 'Login failed. Please check credentials.');
    }
  };

  return (
    <div className="min-h-screen bg-[#F5F3EC] flex flex-col justify-center py-12 px-4 sm:px-6 lg:px-8">
      {/* Brand Header */}
      <div className="sm:mx-auto sm:w-full sm:max-w-md text-center mb-8">
        <Link to="/" className="inline-flex items-center gap-2 mb-2">
          <div className="w-12 h-12 rounded-2xl bg-[#F5F3EC] neu-raised flex items-center justify-center text-[#315C49]">
            <Sparkles className="w-6 h-6 text-[#C98769]" />
          </div>
        </Link>
        <h2 className="font-serif text-3xl font-bold text-[#234737] tracking-tight">
          Sign In to PanchakarmaFlow
        </h2>
        <p className="mt-1 text-xs text-[#6F7B71]">
          Enter your authorized credentials to access your workspace.
        </p>
      </div>

      <div className="sm:mx-auto sm:w-full sm:max-w-md">
        <NeuCard elevation="raised-lg" className="p-8">
          {errorMsg && (
            <div className="mb-4 p-3 rounded-xl bg-[#B74747]/10 border border-[#B74747]/20 text-xs text-[#B74747]">
              {errorMsg}
            </div>
          )}

          <form onSubmit={handleSubmit} className="space-y-4">
            <NeuInput
              label="Email Address"
              type="email"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              placeholder="name@example.com"
              leftIcon={<Mail className="w-4 h-4" />}
              required
            />

            <div>
              <div className="flex items-center justify-between mb-1.5">
                <label className="text-xs font-semibold text-[#27352D]">Password</label>
              </div>
              <div className="relative">
                <input
                  type={showPassword ? 'text' : 'password'}
                  value={password}
                  onChange={(e) => setPassword(e.target.value)}
                  className="w-full px-4 py-2.5 rounded-xl bg-[#F5F3EC] neu-inset text-xs text-[#27352D] placeholder-[#6F7B71]/60 focus:outline-none focus:ring-2 focus:ring-[#315C49]/40"
                  placeholder="Enter your password"
                  required
                />
                <button
                  type="button"
                  onClick={() => setShowPassword(!showPassword)}
                  className="absolute right-3 top-2.5 text-[#6F7B71] hover:text-[#27352D]"
                >
                  {showPassword ? <EyeOff className="w-4 h-4" /> : <Eye className="w-4 h-4" />}
                </button>
              </div>
            </div>

            <div className="pt-2">
              <NeuButton
                type="submit"
                variant="primary"
                size="md"
                isLoading={isLoading}
                className="w-full flex items-center justify-center gap-2"
              >
                Sign In to Workspace
                <ArrowRight className="w-4 h-4" />
              </NeuButton>
            </div>
          </form>

          <div className="mt-6 pt-4 border-t border-[#E3DFC9] text-center text-xs text-[#6F7B71]">
            Don't have a patient account?{' '}
            <Link to="/register" className="font-bold text-[#315C49] hover:underline">
              Create an account
            </Link>
          </div>
        </NeuCard>

        {/* Security Note */}
        <div className="mt-6 text-center text-xs text-[#6F7B71] flex items-center justify-center gap-1.5">
          <ShieldCheck className="w-4 h-4 text-[#357A55]" />
          <span>JWT Rotating Refresh Token Protected • Microservices Gateway</span>
        </div>
      </div>
    </div>
  );
};
