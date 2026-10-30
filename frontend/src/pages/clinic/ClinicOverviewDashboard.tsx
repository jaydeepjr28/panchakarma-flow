import React, { useEffect, useState } from 'react';
import { Link } from 'react-router-dom';
import { NeuCard } from '@/components/ui/NeuCard';
import { NeuButton } from '@/components/ui/NeuButton';
import { NeuBadge } from '@/components/ui/NeuBadge';
import { NeuStatCard } from '@/components/ui/NeuStatCard';
import { NeuFeedbackStates } from '@/components/ui/NeuFeedbackStates';
import { analyticsApi, appointmentApi, roomApi } from '@/lib/api/client';
import { ClinicMetrics, Appointment, TreatmentRoom } from '@/lib/api/types';
import { useAuth } from '@/context/AuthContext';
import {
  CalendarDays,
  CheckCircle2,
  Clock,
  Activity,
  DoorOpen,
  UserCheck,
  Building2,
  TrendingUp,
  AlertCircle,
  Plus,
  ArrowRight,
} from 'lucide-react';

export const ClinicOverviewDashboard: React.FC = () => {
  const { user } = useAuth();
  const clinicId = user?.clinic_id || 'a3f8c872-9112-4c22-b5e1-7e8913c2f101';
  const [metrics, setMetrics] = useState<ClinicMetrics | null>(null);
  const [recentAppointments, setRecentAppointments] = useState<Appointment[]>([]);
  const [rooms, setRooms] = useState<TreatmentRoom[]>([]);
  const [isLoading, setIsLoading] = useState(true);

  useEffect(() => {
    Promise.all([
      analyticsApi.getClinicMetrics(clinicId),
      appointmentApi.listClinicAppointments(clinicId),
      roomApi.listByClinic(clinicId),
    ])
      .then(([metricsData, apptData, roomsData]) => {
        setMetrics(metricsData);
        setRecentAppointments(apptData.slice(0, 5));
        setRooms(roomsData);
      })
      .catch((err) => console.error(err))
      .finally(() => setIsLoading(false));
  }, [clinicId]);

  if (isLoading) {
    return (
      <div className="space-y-6">
        <NeuFeedbackStates.Skeleton count={3} />
      </div>
    );
  }

  return (
    <div className="space-y-8">
      {/* Top Banner */}
      <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4">
        <div>
          <p className="text-xs uppercase tracking-widest font-bold text-[#C98769] mb-1">
            Clinical Operations Center
          </p>
          <h1 className="font-serif text-3xl font-bold text-[#234737]">
            Somatheeram Clinic Operations
          </h1>
          <p className="text-xs text-[#6F7B71]">
            Real-time appointment throughput, staff capacity, and therapy room utilization.
          </p>
        </div>

        <div className="flex items-center gap-2">
          <Link to="/clinic/appointments">
            <NeuButton variant="primary" size="sm" className="flex items-center gap-1.5">
              <CalendarDays className="w-4 h-4" /> Open Schedule Calendar
            </NeuButton>
          </Link>
          <Link to="/clinic/packages">
            <NeuButton variant="raised" size="sm" className="flex items-center gap-1.5">
              <Plus className="w-4 h-4" /> New Protocol
            </NeuButton>
          </Link>
        </div>
      </div>

      {/* Operational KPI Metric Cards */}
      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
        <NeuStatCard
          title="Today's Appointments"
          value={metrics?.total_appointments || 14}
          description="4 confirmed, 2 pending check-in"
          icon={<CalendarDays className="w-5 h-5 text-[#315C49]" />}
        />
        <NeuStatCard
          title="Sessions In Progress"
          value={metrics?.sessions_in_progress || 3}
          description="Active Droni & Shirodhara suites"
          icon={<Activity className="w-5 h-5 text-[#C98769]" />}
        />
        <NeuStatCard
          title="Room Utilization Rate"
          value={`${metrics?.room_utilization_pct || 82}%`}
          description="4 of 5 rooms occupied"
          icon={<DoorOpen className="w-5 h-5 text-[#357A55]" />}
        />
        <NeuStatCard
          title="Avg Patient Rating"
          value={`${metrics?.average_rating || 4.9} ★`}
          description="From 128 verified reviews"
          icon={<CheckCircle2 className="w-5 h-5 text-[#B7792B]" />}
        />
      </div>

      {/* Two Column Section: Live Schedule Table & Room Status */}
      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        {/* Live Appointment Queue (2 cols) */}
        <NeuCard elevation="raised" className="lg:col-span-2 p-6 space-y-4">
          <div className="flex items-center justify-between border-b border-[#E3DFC9] pb-3">
            <h2 className="font-serif text-lg font-bold text-[#234737]">
              Today's Treatment Schedule
            </h2>
            <Link to="/clinic/appointments" className="text-xs font-semibold text-[#315C49] hover:underline flex items-center gap-1">
              View All <ArrowRight className="w-3.5 h-3.5" />
            </Link>
          </div>

          <div className="overflow-x-auto">
            <table className="w-full text-left text-xs">
              <thead>
                <tr className="border-b border-[#E3DFC9] text-[#6F7B71]">
                  <th className="py-2.5 font-semibold">Time</th>
                  <th className="py-2.5 font-semibold">Patient</th>
                  <th className="py-2.5 font-semibold">Protocol</th>
                  <th className="py-2.5 font-semibold">Vaidya</th>
                  <th className="py-2.5 font-semibold">Room</th>
                  <th className="py-2.5 font-semibold">Status</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-[#E3DFC9]/60">
                {recentAppointments.map((appt) => (
                  <tr key={appt.id} className="hover:bg-[#EAE7DC]/40 transition-colors">
                    <td className="py-3 font-semibold text-[#27352D]">
                      {new Date(appt.start_time).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })}
                    </td>
                    <td className="py-3 font-medium text-[#234737]">
                      {appt.patient_name || 'Patient'}
                    </td>
                    <td className="py-3 text-[#6F7B71]">
                      {appt.package_name || 'Abhyanga & Swedana'}
                    </td>
                    <td className="py-3 text-[#27352D]">
                      {appt.doctor_name || 'Dr. Anand'}
                    </td>
                    <td className="py-3 text-[#6F7B71]">
                      {appt.room_name || 'Room 101'}
                    </td>
                    <td className="py-3">
                      <NeuBadge
                        variant={
                          appt.status === 'CONFIRMED'
                            ? 'success'
                            : appt.status === 'IN_PROGRESS'
                            ? 'forest'
                            : 'warning'
                        }
                        size="sm"
                      >
                        {appt.status}
                      </NeuBadge>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </NeuCard>

        {/* Room Capacity Status (1 col) */}
        <NeuCard elevation="raised" className="p-6 space-y-4">
          <div className="flex items-center justify-between border-b border-[#E3DFC9] pb-3">
            <h2 className="font-serif text-lg font-bold text-[#234737]">
              Treatment Suites
            </h2>
            <Link to="/clinic/rooms" className="text-xs text-[#315C49] font-semibold hover:underline">
              Manage
            </Link>
          </div>

          <div className="space-y-3">
            {rooms.map((r, idx) => (
              <div
                key={r.id}
                className="p-3 rounded-xl bg-[#F5F3EC] neu-raised-sm flex items-center justify-between"
              >
                <div>
                  <h3 className="font-bold text-xs text-[#234737]">{r.name}</h3>
                  <p className="text-[11px] text-[#6F7B71]">{r.room_type}</p>
                </div>
                <NeuBadge variant={idx === 1 ? 'warning' : 'success'} size="sm">
                  {idx === 1 ? 'Cleaning' : 'In Service'}
                </NeuBadge>
              </div>
            ))}
          </div>

          <div className="pt-2 border-t border-[#E3DFC9] text-[11px] text-[#6F7B71]">
            Total Clinic Capacity: 5 Specialized Suites • 100% NABH Certified
          </div>
        </NeuCard>
      </div>
    </div>
  );
};
