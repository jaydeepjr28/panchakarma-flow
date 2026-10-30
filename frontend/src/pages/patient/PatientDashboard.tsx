import React, { useEffect, useState } from 'react';
import { Link } from 'react-router-dom';
import { useAuth } from '@/context/AuthContext';
import { NeuCard } from '@/components/ui/NeuCard';
import { NeuButton } from '@/components/ui/NeuButton';
import { NeuBadge } from '@/components/ui/NeuBadge';
import { NeuFeedbackStates } from '@/components/ui/NeuFeedbackStates';
import { appointmentApi, therapyApi } from '@/lib/api/client';
import { Appointment, TherapySession } from '@/lib/api/types';
import {
  Calendar,
  Clock,
  Sparkles,
  CheckCircle2,
  AlertCircle,
  Activity,
  ArrowRight,
  ClipboardList,
  Heart,
  UserCheck,
  Building2,
  CalendarPlus,
} from 'lucide-react';

export const PatientDashboard: React.FC = () => {
  const { user, profile } = useAuth();
  const [appointments, setAppointments] = useState<Appointment[]>([]);
  const [activeSession, setActiveSession] = useState<TherapySession | null>(null);
  const [isLoading, setIsLoading] = useState(true);

  useEffect(() => {
    Promise.all([
      appointmentApi.listPatientAppointments(),
      therapyApi.listPatientSessions(),
    ])
      .then(([appts, sessions]) => {
        setAppointments(appts);
        if (sessions.length > 0) {
          setActiveSession(sessions[0]);
        }
      })
      .catch((err) => console.error(err))
      .finally(() => setIsLoading(false));
  }, []);

  const nextAppointment = appointments.find(
    (a) => a.status === 'CONFIRMED' || a.status === 'PENDING'
  );

  return (
    <div className="space-y-8">
      {/* Welcome & Prakriti Banner */}
      <NeuCard elevation="raised-lg" className="p-8">
        <div className="flex flex-col md:flex-row md:items-center md:justify-between gap-6">
          <div className="space-y-2">
            <div className="inline-flex items-center gap-2 px-3 py-1 rounded-full bg-[#315C49]/10 text-[#315C49] text-xs font-semibold">
              <Sparkles className="w-3.5 h-3.5 text-[#C98769]" />
              Panchakarma Care Portal
            </div>
            <h1 className="font-serif text-3xl sm:text-4xl font-bold text-[#234737]">
              Namaste, {profile?.full_name || user?.email?.split('@')[0] || 'Seeker'}
            </h1>
            <p className="text-xs sm:text-sm text-[#6F7B71] max-w-2xl">
              Your Ayurvedic constitution is categorized as{' '}
              <span className="font-bold text-[#315C49]">{profile?.prakriti_type || 'Holistic Care Pending'}</span>.{' '}
              {profile?.prakriti_type
                ? 'Your prescribed cleansing protocol focuses on pacifying aggravated dosha through gentle oleation and steam.'
                : 'Book a consultation or schedule a therapy package to begin your clinical regimen.'}
            </p>
          </div>

          <div className="flex flex-col sm:flex-row md:flex-col gap-3 shrink-0">
            <Link to="/patient/book">
              <NeuButton variant="primary" size="md" className="w-full flex items-center justify-center gap-2">
                <CalendarPlus className="w-4 h-4" /> Book New Session
              </NeuButton>
            </Link>
            <Link to="/patient/profile">
              <NeuButton variant="raised" size="md" className="w-full flex items-center justify-center gap-2">
                <UserCheck className="w-4 h-4" /> Update Health Profile
              </NeuButton>
            </Link>
          </div>
        </div>
      </NeuCard>

      {/* Two-Column Grid: Next Appointment & Checklist Alert */}
      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        {/* Next Appointment Card (Takes 2 cols) */}
        <NeuCard elevation="raised" className="lg:col-span-2 p-6 space-y-6">
          <div className="flex items-center justify-between border-b border-[#E3DFC9] pb-4">
            <div className="flex items-center gap-2.5">
              <Calendar className="w-5 h-5 text-[#315C49]" />
              <h2 className="font-serif text-xl font-bold text-[#234737]">Next Upcoming Therapy</h2>
            </div>
            {nextAppointment && (
              <NeuBadge
                variant={
                  nextAppointment.status === 'CONFIRMED'
                    ? 'success'
                    : nextAppointment.status === 'IN_PROGRESS'
                    ? 'forest'
                    : 'warning'
                }
              >
                {nextAppointment.status}
              </NeuBadge>
            )}
          </div>

          {isLoading ? (
            <NeuFeedbackStates.Skeleton count={2} />
          ) : !nextAppointment ? (
            <NeuFeedbackStates.EmptyState
              title="No Upcoming Appointments"
              description="You have no confirmed therapy sessions scheduled. Select a package to begin."
              actionText="Book an Appointment"
              onAction={() => window.location.assign('/patient/book')}
            />
          ) : (
            <div className="space-y-4">
              <div className="grid grid-cols-1 sm:grid-cols-2 gap-4 bg-[#F5F3EC] p-5 rounded-2xl neu-inset">
                <div className="space-y-2 text-xs">
                  <div>
                    <span className="text-[#6F7B71] block">Therapy Protocol:</span>
                    <span className="font-bold text-[#234737] text-sm">
                      {nextAppointment.package_name || 'Abhyanga & Swedana'}
                    </span>
                  </div>
                  <div>
                    <span className="text-[#6F7B71] block">Supervising Vaidya:</span>
                    <span className="font-semibold text-[#27352D]">
                      {nextAppointment.doctor_name || 'Dr. Anand Vaidya'}
                    </span>
                  </div>
                  <div>
                    <span className="text-[#6F7B71] block">Treatment Room:</span>
                    <span className="font-semibold text-[#27352D]">
                      {nextAppointment.room_name || 'Room 101 (Droni Equipped)'}
                    </span>
                  </div>
                </div>

                <div className="space-y-2 text-xs">
                  <div>
                    <span className="text-[#6F7B71] block">Scheduled Date & Time:</span>
                    <span className="font-bold text-[#315C49] text-sm">
                      {new Date(nextAppointment.start_time).toLocaleDateString([], {
                        weekday: 'short',
                        month: 'short',
                        day: 'numeric',
                      })}{' '}
                      at{' '}
                      {new Date(nextAppointment.start_time).toLocaleTimeString([], {
                        hour: '2-digit',
                        minute: '2-digit',
                      })}
                    </span>
                  </div>
                  <div>
                    <span className="text-[#6F7B71] block">Duration:</span>
                    <span className="font-semibold text-[#27352D]">
                      {nextAppointment.duration_minutes || 90} minutes
                    </span>
                  </div>
                  <div>
                    <span className="text-[#6F7B71] block">Appointment ID:</span>
                    <span className="font-mono text-[#6F7B71] text-[11px]">{nextAppointment.id}</span>
                  </div>
                </div>
              </div>

              <div className="flex flex-wrap items-center justify-between gap-3 pt-2">
                <Link to="/patient/checklists">
                  <NeuButton variant="raised" size="sm" className="flex items-center gap-1.5 text-xs">
                    <ClipboardList className="w-3.5 h-3.5 text-[#C98769]" />
                    Review Pre-Care Checklist
                  </NeuButton>
                </Link>
                <Link to="/patient/progress">
                  <NeuButton variant="primary" size="sm" className="flex items-center gap-1.5 text-xs">
                    <Activity className="w-3.5 h-3.5" />
                    Live Session Tracker
                  </NeuButton>
                </Link>
              </div>
            </div>
          )}
        </NeuCard>

        {/* Pre-Care Checklist Status Card (Takes 1 col) */}
        <NeuCard elevation="raised" className="p-6 flex flex-col justify-between space-y-4">
          <div>
            <div className="flex items-center justify-between mb-3">
              <span className="text-[11px] font-bold uppercase tracking-wider text-[#C98769]">
                {nextAppointment ? 'Action Required' : 'Care Guidelines'}
              </span>
              <NeuBadge variant={nextAppointment ? 'warning' : 'neutral'} size="sm">
                {nextAppointment ? 'Pending' : 'No Action'}
              </NeuBadge>
            </div>
            <h3 className="font-serif text-lg font-bold text-[#234737] mb-2">
              Pre-Therapy Checklist
            </h3>
            {nextAppointment ? (
              <>
                <p className="text-xs text-[#6F7B71] leading-relaxed mb-4">
                  Classical Panchakarma safety guidelines require verifying 12-hour fasting, light liquid diet, and hydration before arrival.
                </p>

                <div className="space-y-2 text-xs">
                  <div className="flex items-center gap-2 text-[#27352D]">
                    <span className="w-2 h-2 rounded-full bg-[#357A55]" />
                    <span>Hydration intake completed (Warm water)</span>
                  </div>
                  <div className="flex items-center gap-2 text-[#6F7B71]">
                    <span className="w-2 h-2 rounded-full bg-[#B7792B]" />
                    <span>3-hour fast verification pending</span>
                  </div>
                  <div className="flex items-center gap-2 text-[#6F7B71]">
                    <span className="w-2 h-2 rounded-full bg-[#B7792B]" />
                    <span>Loose cotton clothing confirmed</span>
                  </div>
                </div>
              </>
            ) : (
              <p className="text-xs text-[#6F7B71] leading-relaxed">
                You have no scheduled therapy sessions. Pre-procedure guidelines, fasting verification, and dietary instructions will activate here once your therapy is booked.
              </p>
            )}
          </div>

          {nextAppointment ? (
            <Link to="/patient/checklists" className="block pt-2">
              <NeuButton variant="primary" size="sm" className="w-full flex items-center justify-center gap-2">
                Complete Checklist Now <ArrowRight className="w-4 h-4" />
              </NeuButton>
            </Link>
          ) : (
            <Link to="/patient/book" className="block pt-2">
              <NeuButton variant="raised" size="sm" className="w-full flex items-center justify-center gap-2">
                Book a Session <ArrowRight className="w-4 h-4" />
              </NeuButton>
            </Link>
          )}
        </NeuCard>
      </div>

      {/* Therapy Progress Highlights */}
      <NeuCard elevation="raised" className="p-6">
        <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between mb-6 gap-2">
          <div>
            <p className="text-xs uppercase tracking-widest font-bold text-[#315C49]">Protocol Continuity</p>
            <h2 className="font-serif text-xl font-bold text-[#234737]">
              Therapy Course Progress
            </h2>
          </div>
          <Link to="/patient/progress">
            <NeuButton variant="raised" size="sm">
              View Detailed Protocol Timeline
            </NeuButton>
          </Link>
        </div>

        {activeSession ? (
          <div className="space-y-4">
            <div className="flex items-center justify-between text-xs font-semibold">
              <span className="text-[#234737]">{activeSession.package_name || 'Abhyanga Course'}</span>
              <span className="text-[#315C49]">1 of 3 Sessions Completed (33%)</span>
            </div>

            {/* Inset Progress Bar */}
            <div className="w-full h-3 rounded-full bg-[#F5F3EC] neu-inset p-0.5">
              <div className="h-full rounded-full bg-[#315C49] transition-all duration-500 w-1/3" />
            </div>

            <div className="grid grid-cols-1 md:grid-cols-3 gap-4 pt-2">
              <div className="p-3 rounded-xl bg-[#F5F3EC] neu-raised-sm text-xs">
                <span className="text-[#6F7B71] block">Session 1 (Purvakarma):</span>
                <span className="font-bold text-[#357A55]">Completed (Oct 24)</span>
              </div>
              <div className="p-3 rounded-xl bg-[#F5F3EC] neu-inset text-xs border border-[#315C49]/20">
                <span className="text-[#6F7B71] block">Session 2 (Pradhanakarma):</span>
                <span className="font-bold text-[#315C49]">Upcoming (Oct 29)</span>
              </div>
              <div className="p-3 rounded-xl bg-[#F5F3EC] neu-raised-sm text-xs opacity-60">
                <span className="text-[#6F7B71] block">Session 3 (Paschatkarma):</span>
                <span className="font-bold text-[#27352D]">Scheduled (Nov 03)</span>
              </div>
            </div>
          </div>
        ) : (
          <p className="text-xs text-[#6F7B71]">No active therapy cycles initiated.</p>
        )}
      </NeuCard>
    </div>
  );
};
