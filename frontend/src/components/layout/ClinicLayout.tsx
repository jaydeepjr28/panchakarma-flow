import React, { useState } from 'react';
import { NavLink, Outlet } from 'react-router-dom';
import { Navbar } from './Navbar';
import { useAuth } from '@/context/AuthContext';
import { NeuBadge } from '@/components/ui/NeuBadge';
import {
  LayoutDashboard,
  CalendarDays,
  UserCog,
  DoorOpen,
  Package,
  BarChart3,
  Building2,
  Menu,
  X,
  Clock,
  ShieldCheck,
} from 'lucide-react';

export const ClinicLayout: React.FC = () => {
  const { user } = useAuth();
  const [isSidebarOpen, setIsSidebarOpen] = useState(false);

  const navItems = [
    { label: 'Overview Dashboard', to: '/clinic/dashboard', icon: LayoutDashboard },
    { label: 'Appointment Calendar', to: '/clinic/appointments', icon: CalendarDays },
    { label: 'Doctors & Schedules', to: '/clinic/doctors', icon: UserCog },
    { label: 'Treatment Rooms', to: '/clinic/rooms', icon: DoorOpen },
    { label: 'Packages & Protocols', to: '/clinic/packages', icon: Package },
    { label: 'Reports & Analytics', to: '/clinic/reports', icon: BarChart3 },
  ];

  return (
    <div className="min-h-screen bg-[#F5F3EC] flex flex-col text-[#27352D]">
      <Navbar />

      {/* Main Container with Sidebar */}
      <div className="flex-1 flex max-w-7xl w-full mx-auto px-4 sm:px-6 lg:px-8 py-6 gap-6">
        {/* Mobile Sidebar Toggle Button */}
        <div className="lg:hidden fixed bottom-6 right-6 z-50">
          <button
            onClick={() => setIsSidebarOpen(!isSidebarOpen)}
            className="w-12 h-12 rounded-2xl bg-[#315C49] text-white neu-raised flex items-center justify-center shadow-lg"
          >
            {isSidebarOpen ? <X className="w-6 h-6" /> : <Menu className="w-6 h-6" />}
          </button>
        </div>

        {/* Sidebar */}
        <aside
          className={`fixed inset-y-0 left-0 z-40 w-64 bg-[#F5F3EC] p-4 border-r border-[#E3DFC9] transition-transform duration-200 ease-in-out lg:static lg:translate-x-0 lg:border-none lg:p-0 ${
            isSidebarOpen ? 'translate-x-0' : '-translate-x-full'
          }`}
        >
          <div className="w-full lg:w-64 rounded-2xl bg-[#F5F3EC] neu-raised p-5 space-y-6">
            {/* Clinic Info Header */}
            <div className="border-b border-[#E3DFC9] pb-4">
              <div className="flex items-center gap-2.5 mb-1">
                <Building2 className="w-5 h-5 text-[#C98769]" />
                <h2 className="font-serif font-bold text-sm text-[#234737] leading-tight">
                  Somatheeram Hospital
                </h2>
              </div>
              <p className="text-[11px] text-[#6F7B71]">Kovalam, Kerala • NABH Accredited</p>
              <div className="mt-2 flex items-center gap-1.5">
                <NeuBadge variant="terracotta" size="sm">
                  Clinic Admin
                </NeuBadge>
                <NeuBadge variant="success" size="sm">
                  4/5 Rooms Active
                </NeuBadge>
              </div>
            </div>

            {/* Navigation Links */}
            <nav className="space-y-1.5">
              {navItems.map((item) => {
                const Icon = item.icon;
                return (
                  <NavLink
                    key={item.to}
                    to={item.to}
                    onClick={() => setIsSidebarOpen(false)}
                    className={({ isActive }) =>
                      `flex items-center gap-3 px-3.5 py-2.5 rounded-xl text-xs font-semibold transition-all ${
                        isActive
                          ? 'bg-[#315C49] text-[#F5F3EC] shadow-sm'
                          : 'text-[#6F7B71] hover:text-[#234737] hover:bg-[#EAE7DC]'
                      }`
                    }
                  >
                    <Icon className="w-4 h-4" />
                    <span>{item.label}</span>
                  </NavLink>
                );
              })}
            </nav>

            {/* System Status Pill */}
            <div className="pt-4 border-t border-[#E3DFC9] text-[11px] space-y-1 text-[#6F7B71]">
              <div className="flex items-center justify-between">
                <span>Microservices Gate:</span>
                <span className="text-[#357A55] font-semibold flex items-center gap-1">
                  <span className="w-1.5 h-1.5 rounded-full bg-[#357A55] animate-pulse" /> Live
                </span>
              </div>
              <div className="flex items-center justify-between">
                <span>Auto Hold Expiry:</span>
                <span className="font-semibold text-[#27352D]">10 min rule</span>
              </div>
            </div>
          </div>
        </aside>

        {/* Content Outlet Area */}
        <main className="flex-1 min-w-0">
          <Outlet />
        </main>
      </div>
    </div>
  );
};
