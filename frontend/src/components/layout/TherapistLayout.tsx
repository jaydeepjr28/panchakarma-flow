import React from 'react';
import { NavLink, Outlet } from 'react-router-dom';
import { Navbar } from './Navbar';
import { useAuth } from '@/context/AuthContext';
import { NeuBadge } from '@/components/ui/NeuBadge';
import {
  Activity,
  CalendarClock,
  HeartPulse,
  ClipboardCheck,
  AlertCircle,
  FileSpreadsheet,
} from 'lucide-react';

export const TherapistLayout: React.FC = () => {
  const { user } = useAuth();

  const navItems = [
    { label: "Today's Patient Queue", to: '/therapist/dashboard', icon: CalendarClock },
    { label: 'Active Session Tracker', to: '/therapist/session-tracker', icon: Activity },
    { label: 'Follow-Ups & Complications', to: '/therapist/follow-ups', icon: HeartPulse },
  ];

  return (
    <div className="min-h-screen bg-[#F5F3EC] flex flex-col text-[#27352D]">
      <Navbar />

      {/* Therapist Shift & Status Banner */}
      <section className="bg-[#F5F3EC] border-b border-[#E3DFC9]/70 px-4 sm:px-6 lg:px-8 py-3">
        <div className="max-w-7xl mx-auto flex flex-col sm:flex-row sm:items-center sm:justify-between gap-3">
          <div className="flex items-center gap-3">
            <div className="w-9 h-9 rounded-xl bg-[#357A55]/15 text-[#357A55] flex items-center justify-center font-bold">
              <Activity className="w-5 h-5" />
            </div>
            <div>
              <h2 className="text-sm font-bold text-[#234737] flex items-center gap-2">
                Therapist Workspace
                <NeuBadge variant="sage" size="sm">
                  Ramesh Vaidya
                </NeuBadge>
                <span className="text-[11px] px-2 py-0.5 rounded-full bg-[#357A55]/15 text-[#357A55] font-semibold flex items-center gap-1">
                  <span className="w-1.5 h-1.5 rounded-full bg-[#357A55] animate-ping" />
                  On Duty • Room 201
                </span>
              </h2>
              <p className="text-xs text-[#6F7B71]">
                Somatheeram Clinic • Protocol: Abhyanga & Shirodhara
              </p>
            </div>
          </div>

          {/* Therapist Navigation Links */}
          <nav className="flex items-center gap-1.5 overflow-x-auto">
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

      {/* Main Content Area */}
      <main className="flex-1 max-w-7xl w-full mx-auto px-4 sm:px-6 lg:px-8 py-6">
        <Outlet />
      </main>

      {/* Safety Notice */}
      <footer className="border-t border-[#E3DFC9]/70 py-4 text-center text-xs text-[#6F7B71] bg-[#F5F3EC]">
        <p>Therapy Protocol System: Real-time server-timestamped step progression • Doctor escalation protocol active</p>
      </footer>
    </div>
  );
};
