import React from 'react';
import { Navigate, useLocation, Link } from 'react-router-dom';
import { useAuth } from '@/context/AuthContext';
import { UserRole } from '@/lib/api/types';
import { NeuCard } from '@/components/ui/NeuCard';
import { NeuButton } from '@/components/ui/NeuButton';
import { ShieldAlert, ArrowLeft, RefreshCw } from 'lucide-react';

interface ProtectedRouteProps {
  children: React.ReactNode;
  allowedRoles?: UserRole[];
}

export const ProtectedRoute: React.FC<ProtectedRouteProps> = ({ children, allowedRoles }) => {
  const { user, isLoading, switchDemoRole } = useAuth();
  const location = useLocation();

  if (isLoading) {
    return (
      <div className="min-h-screen bg-[#F5F3EC] flex items-center justify-center">
        <div className="flex flex-col items-center gap-3">
          <div className="w-12 h-12 rounded-2xl bg-[#F5F3EC] neu-inset flex items-center justify-center animate-pulse">
            <RefreshCw className="w-6 h-6 text-[#315C49] animate-spin" />
          </div>
          <p className="text-sm font-medium text-[#6F7B71]">Validating security credentials...</p>
        </div>
      </div>
    );
  }

  if (!user) {
    return <Navigate to="/login" state={{ from: location }} replace />;
  }

  if (allowedRoles && !allowedRoles.includes(user.role)) {
    return (
      <div className="min-h-[80vh] flex items-center justify-center p-4">
        <NeuCard elevation="raised-lg" className="max-w-md w-full p-8 text-center">
          <div className="w-16 h-16 mx-auto mb-4 rounded-2xl bg-[#B74747]/10 flex items-center justify-center text-[#B74747]">
            <ShieldAlert className="w-8 h-8" />
          </div>
          <h2 className="font-serif text-2xl font-bold text-[#234737] mb-2">Restricted Access</h2>
          <p className="text-sm text-[#6F7B71] mb-6">
            This workspace requires role authorization (
            <span className="font-semibold text-[#27352D]">{allowedRoles.join(', ')}</span>). Your current
            active role is <span className="font-semibold text-[#27352D]">{user.role}</span>.
          </p>

          <div className="space-y-3">
            {allowedRoles[0] && (
              <NeuButton
                variant="primary"
                className="w-full"
                onClick={() => switchDemoRole(allowedRoles[0])}
              >
                Switch Role to {allowedRoles[0].replace('_', ' ')}
              </NeuButton>
            )}
            <Link to="/" className="block">
              <NeuButton variant="raised" className="w-full flex items-center justify-center gap-2">
                <ArrowLeft className="w-4 h-4" /> Return to Homepage
              </NeuButton>
            </Link>
          </div>
        </NeuCard>
      </div>
    );
  }

  return <>{children}</>;
};
