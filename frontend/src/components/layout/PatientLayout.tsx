import React from 'react';
import { NavLink, Outlet, Link } from 'react-router-dom';
import { Navbar } from './Navbar';
import { useAuth } from '@/context/AuthContext';
import {
  LayoutDashboard,
  CalendarPlus,
  CalendarCheck,
  ClipboardList,
  Activity,
  UserCheck,
  Clock,
  Sparkles,
} from 'lucide-react';

export const PatientLayout: React.FC = () => {
  const { user, profile } = useAuth();

  const navItems = [
    { label: 'Overview', to: '/patient/dashboard', icon: LayoutDashboard },
    { label: 'Book Therapy', to: '/patient/book', icon: CalendarPlus },
    { label: 'My Appointments', to: '/patient/appointments', icon: CalendarCheck },
    { label: 'Pre-Care Checklists', to: '/patient/checklists', icon: ClipboardList },
    { label: 'Session Progress', to: '/patient/progress', icon: Activity },
    { label: 'Health Profile', to: '/patient/profile', icon: UserCheck },
  ];

  return (
    <div className="min-h-screen bg-[#F5F3EC] flex flex-col text-[#27352D]">
      <Navbar />

      {/* Patient Sub-Navigation Bar */}
      <section className="bg-[#F5F3EC] border-b border-[#E3DFC9]/70 px-4 sm:px-6 lg:px-8 py-3">
        <div className="max-w-7xl mx-auto flex flex-col md:flex-row md:items-center md:justify-between gap-4">
          <div className="flex items-center gap-3">
            <div className="w-10 h-10 rounded-xl bg-[#315C49]/10 text-[#315C49] flex items-center justify-center font-bold text-sm">
              {profile?.full_name?.charAt(0) || 'P'}
            </div>
            <div>
              <h2 className="text-sm font-bold text-[#234737] flex items-center gap-2">
                Welcome, {profile?.full_name || user?.email?.split('@')[0]}
                <span className="text-[11px] font-normal px-2 py-0.5 rounded-full bg-[#A8B9A0]/25 text-[#234737] neu-inset">
                  Ayur ID: {profile?.id || 'PAT-001'}
                </span>
              </h2>
              <p className="text-xs text-[#6F7B71]">
                Prakriti: <span className="font-semibold text-[#315C49]">{profile?.prakriti_type || 'Vata-Pitta'}</span> • Next Session in 2 days
              </p>
            </div>
          </div>

          {/* Quick Tabs */}
          <nav className="flex items-center gap-1.5 overflow-x-auto pb-1 md:pb-0 scrollbar-none">
            {navItems.map((item) => {
              const Icon = item.icon;
              return (
                <NavLink
                  key={item.to}
                  to={item.to}
                  className={({ isActive }) =>
                    `flex items-center gap-2 px-3.5 py-2 rounded-xl text-xs font-semibold whitespace-nowrap transition-all ${
                      isActive
                        ? 'bg-[#315C49] text-[#F5F3EC] shadow-sm'
                        : 'bg-[#F5F3EC] text-[#6F7B71] hover:text-[#234737] neu-raised-sm hover:neu-pressed'
                    }`
                  }
                >
                  <Icon className="w-3.5 h-3.5" />
                  <span>{item.label}</span>
                </NavLink>
              );
            })}
          </nav>
        </div>
      </section>

      {/* Main Outlet Container */}
      <main className="flex-1 max-w-7xl w-full mx-auto px-4 sm:px-6 lg:px-8 py-8">
        <Outlet />
      </main>

      {/* Footer Note */}
      <footer className="border-t border-[#E3DFC9]/70 py-6 text-center text-xs text-[#6F7B71] bg-[#F5F3EC]">
        <p>PanchakarmaFlow Patient Portal • Classical Shodhana Protocol Management • Ministry of Ayush Compliant</p>
      </footer>
    </div>
  );
};
