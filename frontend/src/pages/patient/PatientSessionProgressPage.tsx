import React, { useEffect, useState } from 'react';
import { NeuCard } from '@/components/ui/NeuCard';
import { NeuButton } from '@/components/ui/NeuButton';
import { NeuBadge } from '@/components/ui/NeuBadge';
import { NeuFeedbackStates } from '@/components/ui/NeuFeedbackStates';
import { therapyApi } from '@/lib/api/client';
import { TherapySession, StepExecution } from '@/lib/api/types';
import {
  Activity,
  CheckCircle2,
  Clock,
  Heart,
  ShieldCheck,
  Sparkles,
  Info,
  Calendar,
  AlertCircle,
} from 'lucide-react';

export const PatientSessionProgressPage: React.FC = () => {
  const [sessions, setSessions] = useState<TherapySession[]>([]);
  const [selectedSession, setSelectedSession] = useState<TherapySession | null>(null);
  const [isLoading, setIsLoading] = useState(true);

  useEffect(() => {
    therapyApi
      .listPatientSessions()
      .then((data) => {
        setSessions(data);
        if (data.length > 0) {
          setSelectedSession(data[0]);
        }
      })
      .catch((err) => console.error(err))
      .finally(() => setIsLoading(false));
  }, []);

  if (isLoading) {
    return (
      <div className="space-y-6">
        <NeuFeedbackStates.Skeleton count={3} />
      </div>
    );
  }

  if (!selectedSession) {
    return (
      <NeuFeedbackStates.EmptyState
        title="No Therapy Sessions Found"
        description="You have no recorded therapy sessions. Progress will be displayed here once your appointment begins."
        actionText="Book Session"
        onAction={() => window.location.assign('/patient/book')}
      />
    );
  }

  const steps: StepExecution[] = selectedSession.steps || [];
  const completedSteps = steps.filter((s) => s.status === 'COMPLETED').length;
  const progressPercent = steps.length > 0 ? Math.round((completedSteps / steps.length) * 100) : 0;

  return (
    <div className="space-y-8 max-w-5xl mx-auto">
      <div>
        <p className="text-xs uppercase tracking-widest font-bold text-[#C98769] mb-1">
          Patient Therapy Monitoring
        </p>
        <h1 className="font-serif text-3xl font-bold text-[#234737]">
          Live Therapy Progress & Care Protocol
        </h1>
        <p className="text-xs text-[#6F7B71]">
          Monitor the progression of your classical treatment steps and view authorized post-care instructions.
        </p>
      </div>

      {/* Session Overview Card */}
      <NeuCard elevation="raised-lg" className="p-6">
        <div className="flex flex-col md:flex-row md:items-center md:justify-between gap-4 border-b border-[#E3DFC9] pb-4">
          <div className="space-y-1">
            <div className="flex items-center gap-2">
              <NeuBadge variant={selectedSession.status === 'IN_PROGRESS' ? 'forest' : 'success'}>
                {selectedSession.status}
              </NeuBadge>
              <span className="text-[11px] font-mono text-[#6F7B71]">
                Session ID: {selectedSession.id}
              </span>
            </div>
            <h2 className="font-serif text-2xl font-bold text-[#234737]">
              {selectedSession.package_name || 'Classical Panchakarma Protocol'}
            </h2>
            <p className="text-xs text-[#6F7B71]">
              Version {selectedSession.package_version || 1} • Room 101
            </p>
          </div>

          <div className="p-4 rounded-xl bg-[#F5F3EC] neu-inset text-xs space-y-1 text-right">
            <span className="text-[#6F7B71] block">Check-in Status:</span>
            <span className="font-bold text-[#357A55] flex items-center gap-1 justify-end">
              <CheckCircle2 className="w-3.5 h-3.5" /> Baseline Vitals Verified
            </span>
          </div>
        </div>

        {/* Progress Tracker Bar */}
        <div className="pt-6 space-y-3">
          <div className="flex justify-between text-xs font-semibold">
            <span className="text-[#234737]">Step Sequence Execution:</span>
            <span className="text-[#315C49] font-bold">
              {completedSteps} of {steps.length} Steps Completed ({progressPercent}%)
            </span>
          </div>

          <div className="w-full h-3 rounded-full bg-[#F5F3EC] neu-inset p-0.5">
            <div
              className="h-full rounded-full bg-[#315C49] transition-all duration-500"
              style={{ width: `${progressPercent}%` }}
            />
          </div>
        </div>
      </NeuCard>

      {/* Ordered Therapy Steps Timeline */}
      <section className="space-y-4">
        <h3 className="font-serif text-xl font-bold text-[#234737]">
          Prescribed Therapy Steps Sequence
        </h3>

        <div className="space-y-3">
          {steps.map((st) => {
            const isCompleted = st.status === 'COMPLETED';
            const isInProgress = st.status === 'IN_PROGRESS';

            return (
              <NeuCard
                key={st.id || st.step_order}
                elevation={isInProgress ? 'raised-lg' : 'raised-sm'}
                className={`p-5 transition-all ${
                  isInProgress ? 'border-2 border-[#315C49] bg-[#315C49]/5' : ''
                }`}
              >
                <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-3">
                  <div className="flex items-center gap-3.5">
                    <div
                      className={`w-9 h-9 rounded-xl flex items-center justify-center font-bold text-xs ${
                        isCompleted
                          ? 'bg-[#357A55] text-white shadow-sm'
                          : isInProgress
                          ? 'bg-[#315C49] text-white neu-raised animate-pulse'
                          : 'bg-[#F5F3EC] text-[#6F7B71] neu-inset'
                      }`}
                    >
                      {isCompleted ? <CheckCircle2 className="w-4 h-4" /> : st.step_order}
                    </div>

                    <div>
                      <h4 className="font-serif text-base font-bold text-[#234737] flex items-center gap-2">
                        {st.name}
                        {isInProgress && (
                          <span className="text-[10px] px-2 py-0.5 rounded-full bg-[#315C49] text-white font-sans animate-pulse">
                            Active Step
                          </span>
                        )}
                      </h4>
                      <p className="text-xs text-[#6F7B71]">
                        Planned Duration: {st.planned_duration_minutes} minutes
                        {st.actual_duration_seconds > 0 && (
                          <span> • Executed: {Math.round(st.actual_duration_seconds / 60)} mins</span>
                        )}
                      </p>
                    </div>
                  </div>

                  <div>
                    <NeuBadge
                      variant={
                        isCompleted
                          ? 'success'
                          : isInProgress
                          ? 'forest'
                          : 'sage'
                      }
                      size="sm"
                    >
                      {st.status}
                    </NeuBadge>
                  </div>
                </div>

                {/* Patient Privacy Guard: We deliberately DO NOT show therapist internal notes here */}
              </NeuCard>
            );
          })}
        </div>
      </section>

      {/* Authorized Post-Care Instructions Card */}
      <NeuCard elevation="raised" className="p-6 space-y-4">
        <div className="flex items-center gap-2.5 border-b border-[#E3DFC9] pb-3">
          <Heart className="w-5 h-5 text-[#C98769]" />
          <h3 className="font-serif text-lg font-bold text-[#234737]">
            Authorized Post-Care Instructions (Paschatkarma)
          </h3>
        </div>

        <div className="p-4 rounded-xl bg-[#F5F3EC] neu-inset text-xs text-[#27352D] leading-relaxed space-y-2">
          <p className="font-semibold text-[#315C49]">
            {selectedSession.post_care_instructions ||
              'Rest quietly in a draught-free room. Sip warm coriander water throughout the afternoon. Avoid direct sun exposure and bathing with cold water for 24 hours.'}
          </p>
          <p className="text-[#6F7B71] text-[11px]">
            Diet Recommendation: Soft-cooked rice with moong dal soup (Kitchari). Avoid raw salads, ice, and curd.
          </p>
        </div>

        <div className="flex items-center gap-2 text-[11px] text-[#6F7B71]">
          <ShieldCheck className="w-4 h-4 text-[#357A55]" />
          <span>Verified and dispatched by attending Ayurvedic Vaidya.</span>
        </div>
      </NeuCard>
    </div>
  );
};
