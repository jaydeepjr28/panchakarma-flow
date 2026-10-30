import React, { useEffect, useState } from 'react';
import { Link } from 'react-router-dom';
import { NeuCard } from '@/components/ui/NeuCard';
import { NeuButton } from '@/components/ui/NeuButton';
import { NeuBadge } from '@/components/ui/NeuBadge';
import { NeuFeedbackStates } from '@/components/ui/NeuFeedbackStates';
import { therapyApi } from '@/lib/api/client';
import { TherapySession } from '@/lib/api/types';
import {
  Activity,
  Calendar,
  Clock,
  UserCheck,
  CheckCircle2,
  AlertTriangle,
  Play,
  ArrowRight,
  ShieldAlert,
  ClipboardCheck,
} from 'lucide-react';

export const TherapistDashboard: React.FC = () => {
  const [sessions, setSessions] = useState<TherapySession[]>([]);
  const [isLoading, setIsLoading] = useState(true);

  useEffect(() => {
    therapyApi
      .listTherapistAssigned()
      .then((data) => setSessions(data))
      .catch((err) => console.error(err))
      .finally(() => setIsLoading(false));
  }, []);

  return (
    <div className="space-y-6">
      {/* Workspace Header */}
      <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4">
        <div>
          <p className="text-xs uppercase tracking-widest font-bold text-[#357A55] mb-1">
            Clinical Practice Workspace
          </p>
          <h1 className="font-serif text-3xl font-bold text-[#234737]">
            Therapist Assigned Treatment Queue
          </h1>
          <p className="text-xs text-[#6F7B71]">
            Today's scheduled therapies, baseline readiness verification, and active step trackers.
          </p>
        </div>

        <Link to="/therapist/session-tracker">
          <NeuButton variant="primary" size="md" className="flex items-center gap-2">
            <Activity className="w-4 h-4 animate-pulse" /> Open Active Tracker
          </NeuButton>
        </Link>
      </div>

      {/* Safety Notice Banner */}
      <div className="p-4 rounded-2xl bg-[#F5F3EC] neu-raised-sm border-l-4 border-[#315C49] flex items-center justify-between text-xs">
        <div className="flex items-center gap-3">
          <ClipboardCheck className="w-5 h-5 text-[#315C49]" />
          <div>
            <span className="font-bold text-[#234737] block">
              Ayush Protocol Safety Check
            </span>
            <span className="text-[#6F7B71]">
              Every session requires verified baseline vitals (BP, Pulse, Temp) prior to server session initialization.
            </span>
          </div>
        </div>
        <NeuBadge variant="success" size="sm">Doctor On-Call</NeuBadge>
      </div>

      {/* Assigned Patient Queue */}
      <div className="space-y-4">
        <h2 className="font-serif text-xl font-bold text-[#234737]">
          Assigned Patients for Today ({sessions.length})
        </h2>

        {isLoading ? (
          <NeuFeedbackStates.Skeleton count={3} />
        ) : sessions.length === 0 ? (
          <NeuFeedbackStates.EmptyState
            title="No Assigned Treatments"
            description="You currently have no patient appointments assigned to your shift queue."
          />
        ) : (
          <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
            {sessions.map((sess) => {
              const completedSteps = (sess.steps || []).filter((s) => s.status === 'COMPLETED').length;
              const totalSteps = (sess.steps || []).length;
              const isCheckedIn = sess.status === 'CHECKED_IN';
              const isInProgress = sess.status === 'IN_PROGRESS';

              return (
                <NeuCard
                  key={sess.id}
                  elevation={isInProgress ? 'raised-lg' : 'raised'}
                  className={`p-6 flex flex-col justify-between space-y-4 transition-all ${
                    isInProgress ? 'border-2 border-[#315C49] bg-[#315C49]/5' : ''
                  }`}
                >
                  <div>
                    <div className="flex items-start justify-between gap-2 mb-3">
                      <NeuBadge
                        variant={
                          isInProgress
                            ? 'forest'
                            : sess.status === 'COMPLETED'
                            ? 'success'
                            : 'terracotta'
                        }
                      >
                        {sess.status}
                      </NeuBadge>
                      <span className="text-[11px] font-mono text-[#6F7B71]">Room 101</span>
                    </div>

                    <h3 className="font-serif text-xl font-bold text-[#234737] mb-1">
                      {sess.patient_name || 'Patient'}
                    </h3>
                    <p className="text-xs font-semibold text-[#315C49] mb-3">
                      {sess.package_name || 'Abhyanga & Swedana'} (Ver. {sess.package_version || 1})
                    </p>

                    <div className="space-y-2 text-xs text-[#6F7B71] bg-[#F5F3EC] p-3 rounded-xl neu-inset">
                      <div className="flex justify-between">
                        <span>Checklist Verified:</span>
                        <span className="font-bold text-[#357A55]">✓ Patient Confirmed</span>
                      </div>
                      <div className="flex justify-between">
                        <span>Baseline Vitals:</span>
                        <span className="font-bold text-[#27352D]">
                          {sess.baseline_vitals
                            ? `${sess.baseline_vitals.bp_systolic}/${sess.baseline_vitals.bp_diastolic} mmHg • ${sess.baseline_vitals.pulse} bpm`
                            : 'Pending entry on check-in'}
                        </span>
                      </div>
                      <div className="flex justify-between">
                        <span>Step Progress:</span>
                        <span className="font-bold text-[#234737]">
                          {completedSteps} / {totalSteps} Steps Complete
                        </span>
                      </div>
                    </div>
                  </div>

                  <div className="pt-3 border-t border-[#E3DFC9] flex items-center justify-between gap-3">
                    <Link to="/therapist/session-tracker" className="w-full">
                      <NeuButton
                        variant={isInProgress ? 'primary' : 'raised'}
                        size="sm"
                        className="w-full flex items-center justify-center gap-2"
                      >
                        <Play className="w-3.5 h-3.5" />
                        {isInProgress ? 'Resume Live Step Tracker' : 'Open Session Controls'}
                      </NeuButton>
                    </Link>
                  </div>
                </NeuCard>
              );
            })}
          </div>
        )}
      </div>
    </div>
  );
};
