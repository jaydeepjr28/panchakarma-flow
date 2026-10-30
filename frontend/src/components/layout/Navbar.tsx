import React from 'react';
import { Link, useLocation } from 'react-router-dom';
import { useAuth } from '@/context/AuthContext';
import { NeuButton } from '@/components/ui/NeuButton';
import { UserRole } from '@/lib/api/types';
import {
  LogOut,
  LayoutDashboard,
  CalendarDays,
  Users,
  DoorOpen,
  Package,
  BarChart3,
  ClipboardList,
  Timer,
  HeartHandshake,
} from 'lucide-react';

// Nav item shape
interface NavItem {
  label: string;
  to: string;
  icon?: React.ReactNode;
  matchPrefix?: string;
}

// Returns center nav items based on authenticated role
function getNavItems(role?: UserRole): NavItem[] {
  switch (role) {
    case 'clinic_admin':
    case 'super_admin':
      return [
        { label: 'Dashboard',     to: '/clinic/dashboard',     icon: <LayoutDashboard className="w-3.5 h-3.5" />, matchPrefix: '/clinic/dashboard' },
        { label: 'Appointments',  to: '/clinic/appointments',  icon: <CalendarDays    className="w-3.5 h-3.5" />, matchPrefix: '/clinic/appointments' },
        { label: 'Doctors',       to: '/clinic/doctors',       icon: <Users           className="w-3.5 h-3.5" />, matchPrefix: '/clinic/doctors' },
        { label: 'Rooms',         to: '/clinic/rooms',         icon: <DoorOpen        className="w-3.5 h-3.5" />, matchPrefix: '/clinic/rooms' },
        { label: 'Packages',      to: '/clinic/packages',      icon: <Package         className="w-3.5 h-3.5" />, matchPrefix: '/clinic/packages' },
        { label: 'Reports',       to: '/clinic/reports',       icon: <BarChart3       className="w-3.5 h-3.5" />, matchPrefix: '/clinic/reports' },
      ];

    case 'therapist':
    case 'doctor':
      return [
        { label: 'Dashboard',   to: '/therapist/dashboard',       icon: <LayoutDashboard className="w-3.5 h-3.5" />, matchPrefix: '/therapist/dashboard' },
        { label: 'Sessions',    to: '/therapist/session-tracker', icon: <Timer           className="w-3.5 h-3.5" />, matchPrefix: '/therapist/session-tracker' },
        { label: 'Follow-ups',  to: '/therapist/follow-ups',      icon: <HeartHandshake  className="w-3.5 h-3.5" />, matchPrefix: '/therapist/follow-ups' },
      ];

    case 'patient':
      return [
        { label: 'Overview',           to: '/',              matchPrefix: undefined },
        { label: 'Clinics & Packages', to: '/clinics',       matchPrefix: '/clinics' },
        { label: 'My Workspace',       to: '/patient/dashboard', matchPrefix: '/patient' },
      ];

    default:
      // Unauthenticated visitors
      return [
        { label: 'Overview',           to: '/',        matchPrefix: undefined },
        { label: 'Clinics & Packages', to: '/clinics', matchPrefix: '/clinics' },
      ];
  }
}

export const Navbar: React.FC = () => {
  const { user, profile, logout } = useAuth();
  const location = useLocation();

  const getPortalLink = (role?: UserRole) => {
    switch (role) {
      case 'clinic_admin':  return '/clinic/dashboard';
      case 'therapist':     return '/therapist/dashboard';
      case 'doctor':        return '/clinic/doctors';
      case 'patient':
      default:              return '/patient/dashboard';
    }
  };

  const navItems = getNavItems(user?.role);

  const isActive = (item: NavItem) => {
    if (item.matchPrefix) return location.pathname.startsWith(item.matchPrefix);
    return location.pathname === item.to;
  };

  return (
    <header className="sticky top-0 z-40 bg-[#F5F3EC]/90 backdrop-blur-md border-b border-[#E3DFC9]/70 shadow-sm transition-all">
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 h-20 flex items-center justify-between">

        {/* Brand Logo */}
        <Link to={user ? getPortalLink(user.role) : '/'} className="flex items-center gap-3 group flex-shrink-0">
          <div className="w-12 h-12 rounded-2xl bg-[#F5F3EC] neu-raised flex items-center justify-center text-[#315C49] group-hover:neu-pressed transition-all">
            <svg className="w-7 h-7" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
              <path d="M12 2C6.5 2 2 6.5 2 12c0 5 3.5 9 8 10V14H8v-2h2V9.5C10 7.5 11.5 6 13.5 6H16v2h-2.5C13.2 8 13 8.2 13 8.5V12h3l-.5 2H13v8c4.5-1 8-5 8-10 0-5.5-4.5-10-10-10z" className="opacity-0" />
              <path d="M12 3c-1.5 3-4.5 6-7 7 2.5 1 5.5 4 7 7 1.5-3 4.5-6 7-7-2.5-1-5.5-4-7-7z" fill="#315C49" fillOpacity="0.15" />
              <path d="M12 3v18" />
              <path d="M12 14c-2.5-1.5-5-2-8-2 3-2 5.5-2.5 8-2" />
              <path d="M12 14c2.5-1.5 5-2 8-2-3-2-5.5-2.5-8-2" />
            </svg>
          </div>
          <div className="flex flex-col">
            <span className="font-serif font-bold text-xl tracking-tight text-[#234737] flex items-center gap-1.5">
              Panchakarma<span className="text-[#C98769] font-sans font-light">Flow</span>
            </span>
            <span className="text-[10px] tracking-widest uppercase font-medium text-[#6F7B71]">
              Therapy Management Protocol
            </span>
          </div>
        </Link>

        {/* Center Navigation — role-aware */}
        <nav className="hidden md:flex items-center gap-1">
          {navItems.map((item) => (
            <Link
              key={item.to}
              to={item.to}
              className={`flex items-center gap-1.5 px-3 py-2 rounded-xl text-sm font-medium transition-all ${
                isActive(item)
                  ? 'text-[#234737] neu-inset bg-[#F5F3EC]'
                  : 'text-[#6F7B71] hover:text-[#234737] hover:bg-[#EAE7DC]/60'
              }`}
            >
              {item.icon}
              {item.label}
            </Link>
          ))}
        </nav>

        {/* Right — Auth actions */}
        <div className="flex items-center gap-2 flex-shrink-0">
          {user ? (
            <>
              <Link
                to={getPortalLink(user.role)}
                className="flex items-center gap-2 px-3 py-1.5 rounded-xl bg-[#F5F3EC] neu-raised-sm hover:neu-pressed transition-all"
              >
                <div className="w-7 h-7 rounded-lg bg-[#315C49] text-[#F5F3EC] flex items-center justify-center font-bold text-xs">
                  {profile?.full_name
                    ? profile.full_name.charAt(0).toUpperCase()
                    : user.email.charAt(0).toUpperCase()}
                </div>
                <div className="hidden sm:block text-left">
                  <p className="text-xs font-semibold text-[#27352D] leading-tight truncate max-w-[120px]">
                    {profile?.full_name || user.email.split('@')[0]}
                  </p>
                  <p className="text-[10px] text-[#6F7B71] capitalize leading-tight">
                    {user.role.replace('_', ' ')}
                  </p>
                </div>
              </Link>
              <button
                onClick={logout}
                className="w-9 h-9 rounded-xl bg-[#F5F3EC] neu-raised-sm hover:neu-pressed text-[#B74747] flex items-center justify-center transition-all"
                title="Log out"
              >
                <LogOut className="w-4 h-4" />
              </button>
            </>
          ) : (
            <>
              <Link to="/login">
                <NeuButton variant="raised" size="sm">Sign In</NeuButton>
              </Link>
              <Link to="/register">
                <NeuButton variant="primary" size="sm">Register</NeuButton>
              </Link>
            </>
          )}
        </div>

      </div>
    </header>
  );
};
