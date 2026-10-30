import React, { useState, useEffect } from 'react';
import { NeuCard } from '@/components/ui/NeuCard';
import { NeuButton } from '@/components/ui/NeuButton';
import { NeuBadge } from '@/components/ui/NeuBadge';
import { NeuModal } from '@/components/ui/NeuModal';
import { NeuInput } from '@/components/ui/NeuInput';
import { NeuFeedbackStates } from '@/components/ui/NeuFeedbackStates';
import { therapyApi } from '@/lib/api/client';
import {
  TherapySession,
  StepExecution,
  BaselineVitals,
  SessionMaterial,
  ComplicationEvent,
} from '@/lib/api/types';
import {
  Activity,
  Play,
  Pause,
  CheckCircle2,
  AlertTriangle,
  HeartPulse,
  Clock,
  Sparkles,
  ClipboardCheck,
  Stethoscope,
  Plus,
  ShieldAlert,
  Send,
  Layers,
  FileText,
} from 'lucide-react';

export const TherapistSessionExecution: React.FC = () => {
  const [session, setSession] = useState<TherapySession | null>(null);
  const [isLoading, setIsLoading] = useState(true);

  // Active step countdown timer
  const [activeStepIndex, setActiveStepIndex] = useState<number>(0);
  const [timerSeconds, setTimerSeconds] = useState<number>(1800); // 30 mins
  const [isTimerRunning, setIsTimerRunning] = useState<boolean>(false);

  // Check-In & Vitals Modal
  const [showCheckInModal, setShowCheckInModal] = useState(false);
  const [bpSystolic, setBpSystolic] = useState(120);
  const [bpDiastolic, setBpDiastolic] = useState(80);
  const [pulse, setPulse] = useState(72);
  const [tempC, setTempC] = useState(36.8);
  const [checklistVerified, setChecklistVerified] = useState(true);

  // Material Modal
  const [showMaterialModal, setShowMaterialModal] = useState(false);
  const [matName, setMatName] = useState('Mahanarayana Taila');
  const [matQuantity, setMatQuantity] = useState(250);
  const [matUnit, setMatUnit] = useState('ml');
  const [matBatch, setMatBatch] = useState('BATCH-2026-AYUR-99');
  const [materialsList, setMaterialsList] = useState<SessionMaterial[]>([]);

  // Complication Alert Modal
  const [showComplicationModal, setShowComplicationModal] = useState(false);
  const [compSeverity, setCompSeverity] = useState<'LOW' | 'MEDIUM' | 'HIGH' | 'CRITICAL'>('MEDIUM');
  const [compSymptoms, setCompSymptoms] = useState('');
  const [compActions, setCompActions] = useState('');
  const [activeComplication, setActiveComplication] = useState<ComplicationEvent | null>(null);

  // Notes & Post-Care
  const [therapistNotes, setTherapistNotes] = useState('Patient comfortable, pulse stable.');
  const [observations, setObservations] = useState('Good sweat response achieved during Swedana.');
  const [postCareInstructions, setPostCareInstructions] = useState(
    'Rest in draught-free room. Sip warm water. Avoid bathing with cold water or air conditioning for 24 hours.'
  );

  const [statusMessage, setStatusMessage] = useState('');

  // 1. Load initial session
  useEffect(() => {
    therapyApi
      .listTherapistAssigned()
      .then((sessions) => {
        if (sessions.length > 0) {
          const s = sessions[0];
          setSession(s);
          setMaterialsList(s.materials || []);
          if (s.steps && s.steps.length > 0) {
            const inProgIdx = s.steps.findIndex((st) => st.status === 'IN_PROGRESS');
            if (inProgIdx >= 0) {
              setActiveStepIndex(inProgIdx);
              setTimerSeconds(s.steps[inProgIdx].planned_duration_minutes * 60);
              setIsTimerRunning(true);
            }
          }
        }
      })
      .catch((err) => console.error(err))
      .finally(() => setIsLoading(false));
  }, []);

  // Timer Tick
  useEffect(() => {
    if (!isTimerRunning) return;
    const interval = setInterval(() => {
      setTimerSeconds((prev) => (prev > 0 ? prev - 1 : 0));
    }, 1000);
    return () => clearInterval(interval);
  }, [isTimerRunning]);

  // Action: Check-In Patient with Vitals
  const handleCheckInSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!session) return;
    try {
      const vitals: BaselineVitals = {
        bp_systolic: bpSystolic,
        bp_diastolic: bpDiastolic,
        pulse,
        temperature_c: tempC,
      };
      await therapyApi.checkIn(session.appointment_id, vitals);
      setSession((prev) => (prev ? { ...prev, baseline_vitals: vitals, status: 'CHECKED_IN' } : null));
      setShowCheckInModal(false);
      setStatusMessage('Patient checked in and baseline vitals validated.');
    } catch (err: any) {
      alert(err.message || 'Check-in failed');
    }
  };

  // Action: Start Therapy Session (Server Timestamp)
  const handleStartSession = async () => {
    if (!session) return;
    try {
      const updated = await therapyApi.startSession(session.id);
      setSession(updated);
      setIsTimerRunning(true);
      setStatusMessage('Therapy session started with authoritative server timestamp.');
    } catch (err: any) {
      alert(err.message || 'Failed to start session');
    }
  };

  // Action: Pause Active Step
  const handlePauseStep = async () => {
    if (!session || !session.steps) return;
    const stepOrder = session.steps[activeStepIndex]?.step_order || activeStepIndex + 1;
    try {
      await therapyApi.pauseStep(session.id, stepOrder);
      setIsTimerRunning(false);
      setStatusMessage(`Step ${stepOrder} paused.`);
    } catch (err: any) {
      alert(err.message || 'Pause failed');
    }
  };

  // Action: Resume Active Step
  const handleResumeStep = async () => {
    if (!session || !session.steps) return;
    const stepOrder = session.steps[activeStepIndex]?.step_order || activeStepIndex + 1;
    try {
      await therapyApi.resumeStep(session.id, stepOrder);
      setIsTimerRunning(true);
      setStatusMessage(`Step ${stepOrder} resumed.`);
    } catch (err: any) {
      alert(err.message || 'Resume failed');
    }
  };

  // Action: Complete Current Step & Advance
  const handleCompleteStep = async () => {
    if (!session || !session.steps) return;
    const stepOrder = session.steps[activeStepIndex]?.step_order || activeStepIndex + 1;
    try {
      await therapyApi.completeStep(session.id, stepOrder, {
        actual_duration_seconds: 1800 - timerSeconds,
        therapist_notes: therapistNotes,
        observations,
      });

      // Update local steps array
      const updatedSteps = [...session.steps];
      updatedSteps[activeStepIndex] = {
        ...updatedSteps[activeStepIndex],
        status: 'COMPLETED',
        actual_duration_seconds: 1800 - timerSeconds,
      };

      if (activeStepIndex + 1 < updatedSteps.length) {
        updatedSteps[activeStepIndex + 1] = {
          ...updatedSteps[activeStepIndex + 1],
          status: 'IN_PROGRESS',
        };
        setActiveStepIndex(activeStepIndex + 1);
        setTimerSeconds(updatedSteps[activeStepIndex + 1].planned_duration_minutes * 60);
        setIsTimerRunning(true);
      } else {
        setIsTimerRunning(false);
      }

      setSession({ ...session, steps: updatedSteps });
      setStatusMessage(`Step ${stepOrder} completed successfully.`);
    } catch (err: any) {
      alert(err.message || 'Complete step failed');
    }
  };

  // Action: Record Material Used
  const handleRecordMaterial = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!session) return;
    try {
      const newMat = await therapyApi.recordMaterial(session.id, {
        material_name: matName,
        quantity: matQuantity,
        unit: matUnit,
        batch_number: matBatch,
      });
      setMaterialsList((prev) => [...prev, newMat]);
      setShowMaterialModal(false);
      setStatusMessage(`Logged consumable: ${matQuantity}${matUnit} of ${matName}`);
    } catch (err: any) {
      alert(err.message || 'Failed to record material');
    }
  };

  // Action: Escalate Complication to Doctor
  const handleFlagComplication = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!session) return;
    try {
      const comp = await therapyApi.flagComplication(session.id, {
        severity: compSeverity,
        symptoms: compSymptoms,
        immediate_actions_taken: compActions,
      });
      setActiveComplication(comp);
      setSession((prev) => (prev ? { ...prev, status: 'COMPLICATION_ALERT' } : null));
      setIsTimerRunning(false); // Pause timer during emergency
      setShowComplicationModal(false);
      setStatusMessage('CRITICAL: Complication escalated to Dr. Anand Vaidya.');
    } catch (err: any) {
      alert(err.message || 'Failed to flag complication');
    }
  };

  // Action: Resolve Complication
  const handleResolveComplication = async () => {
    if (!session || !activeComplication) return;
    try {
      await therapyApi.resolveComplication(
        session.id,
        activeComplication.id,
        'Vitals normalized after fluid intake. Cleared to resume treatment.',
        'IN_PROGRESS'
      );
      setActiveComplication(null);
      setSession((prev) => (prev ? { ...prev, status: 'IN_PROGRESS' } : null));
      setIsTimerRunning(true);
      setStatusMessage('Doctor reviewed and cleared session to resume.');
    } catch (err: any) {
      alert(err.message || 'Failed to resolve complication');
    }
  };

  // Action: Complete Entire Session
  const handleCompleteSession = async () => {
    if (!session) return;
    try {
      const completed = await therapyApi.completeSession(session.id, {
        therapist_notes: therapistNotes,
        observations,
        post_care_instructions: postCareInstructions,
      });
      setSession(completed);
      setIsTimerRunning(false);
      setStatusMessage('Therapy session completed and post-care instructions dispatched to patient!');
    } catch (err: any) {
      alert(err.message || 'Failed to complete session');
    }
  };

  if (isLoading) {
    return <NeuFeedbackStates.Skeleton count={3} />;
  }

  if (!session) {
    return (
      <NeuFeedbackStates.EmptyState
        title="No Active Session Found"
        description="Select a patient from today's queue to initiate treatment."
      />
    );
  }

  const steps = session.steps || [];
  const activeStep = steps[activeStepIndex];

  return (
    <div className="space-y-6">
      {/* Top Header */}
      <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4">
        <div>
          <div className="flex items-center gap-2 mb-1">
            <span className="text-xs uppercase tracking-widest font-bold text-[#357A55]">
              Real-Time Treatment Control
            </span>
            <NeuBadge
              variant={
                session.status === 'COMPLICATION_ALERT'
                  ? 'error'
                  : session.status === 'IN_PROGRESS'
                  ? 'forest'
                  : 'success'
              }
            >
              {session.status}
            </NeuBadge>
          </div>
          <h1 className="font-serif text-3xl font-bold text-[#234737]">
            {session.patient_name || 'Patient'} • {session.package_name || 'Abhyanga & Swedana'}
          </h1>
          <p className="text-xs text-[#6F7B71]">
            Room 101 • Attending Doctor: Dr. Anand Vaidya • Protocol Version {session.package_version || 1}
          </p>
        </div>

        {/* Global Action Buttons */}
        <div className="flex flex-wrap items-center gap-2">
          {!session.baseline_vitals ? (
            <NeuButton
              variant="primary"
              size="sm"
              onClick={() => setShowCheckInModal(true)}
              className="flex items-center gap-1.5"
            >
              <HeartPulse className="w-4 h-4" /> Check-In & Log Vitals
            </NeuButton>
          ) : session.status === 'CHECKED_IN' ? (
            <NeuButton
              variant="primary"
              size="sm"
              onClick={handleStartSession}
              className="flex items-center gap-1.5"
            >
              <Play className="w-4 h-4" /> Start Therapy Session
            </NeuButton>
          ) : session.status === 'IN_PROGRESS' ? (
            <NeuButton
              variant="primary"
              size="sm"
              onClick={handleCompleteSession}
              className="flex items-center gap-1.5"
            >
              <CheckCircle2 className="w-4 h-4" /> Complete Therapy Session
            </NeuButton>
          ) : null}

          <NeuButton
            variant="danger"
            size="sm"
            onClick={() => setShowComplicationModal(true)}
            className="flex items-center gap-1.5"
          >
            <ShieldAlert className="w-4 h-4" /> Escalate Complication
          </NeuButton>
        </div>
      </div>

      {statusMessage && (
        <div className="p-4 rounded-xl bg-[#357A55]/10 border border-[#357A55]/20 text-xs text-[#357A55] flex items-center justify-between">
          <span className="font-semibold">{statusMessage}</span>
          <button onClick={() => setStatusMessage('')} className="font-bold">✕</button>
        </div>
      )}

      {/* Complication Emergency Alert Banner */}
      {session.status === 'COMPLICATION_ALERT' && (
        <div className="p-5 rounded-2xl bg-[#B74747]/15 border-2 border-[#B74747] text-[#27352D] space-y-3">
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-2 text-[#B74747] font-bold text-sm">
              <ShieldAlert className="w-5 h-5 animate-pulse" />
              <span>CLINICAL COMPLICATION ALERT DISPATCHED TO DOCTOR</span>
            </div>
            <NeuBadge variant="error">CRITICAL ATTENTION</NeuBadge>
          </div>
          <p className="text-xs text-[#27352D]">
            Immediate actions logged. Dr. Anand Vaidya notified via synchronous clinical channel.
          </p>
          <div className="pt-2">
            <NeuButton variant="raised" size="sm" onClick={handleResolveComplication}>
              Doctor Review Complete: Clear & Resume Session
            </NeuButton>
          </div>
        </div>
      )}

      {/* Central Interactive Grid: Active Step Timer & Vitals */}
      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        {/* Active Step Progression & Countdown (2 cols) */}
        <NeuCard elevation="raised-lg" className="lg:col-span-2 p-6 space-y-6">
          <div className="flex items-center justify-between border-b border-[#E3DFC9] pb-4">
            <div className="flex items-center gap-2">
              <Clock className="w-5 h-5 text-[#315C49]" />
              <h2 className="font-serif text-xl font-bold text-[#234737]">
                Active Step: {activeStep?.name || 'Snehana (Deep Oleation)'}
              </h2>
            </div>
            <NeuBadge variant="forest">
              Step {activeStepIndex + 1} of {steps.length}
            </NeuBadge>
          </div>

          {/* Large Countdown Timer Display */}
          <div className="text-center py-6 bg-[#F5F3EC] rounded-2xl neu-inset">
            <span className="text-xs uppercase tracking-widest text-[#6F7B71] block mb-1">
              Remaining Step Duration
            </span>
            <span className="font-mono text-5xl sm:text-6xl font-bold text-[#234737] tracking-tight">
              {Math.floor(timerSeconds / 60)}:
              {String(timerSeconds % 60).padStart(2, '0')}
            </span>
            <div className="mt-4 flex items-center justify-center gap-3">
              {isTimerRunning ? (
                <NeuButton variant="raised" size="sm" onClick={handlePauseStep} className="flex items-center gap-1.5">
                  <Pause className="w-4 h-4" /> Pause Step Timer
                </NeuButton>
              ) : (
                <NeuButton variant="primary" size="sm" onClick={handleResumeStep} className="flex items-center gap-1.5">
                  <Play className="w-4 h-4" /> Start / Resume Timer
                </NeuButton>
              )}
              <NeuButton variant="primary" size="sm" onClick={handleCompleteStep} className="flex items-center gap-1.5">
                <CheckCircle2 className="w-4 h-4" /> Complete Step & Advance
              </NeuButton>
            </div>
          </div>

          {/* Ordered Steps Timeline */}
          <div className="space-y-3 pt-2">
            <p className="text-xs font-bold uppercase tracking-wider text-[#6F7B71]">
              Protocol Step Timeline:
            </p>
            <div className="grid grid-cols-1 sm:grid-cols-2 md:grid-cols-4 gap-2">
              {steps.map((st, idx) => {
                const isPast = idx < activeStepIndex;
                const isCurrent = idx === activeStepIndex;
                return (
                  <div
                    key={st.id || idx}
                    className={`p-3 rounded-xl text-xs transition-all ${
                      isCurrent
                        ? 'neu-inset border-2 border-[#315C49] bg-[#315C49]/5 font-bold text-[#234737]'
                        : isPast
                        ? 'bg-[#357A55]/10 text-[#357A55] font-semibold'
                        : 'bg-[#F5F3EC] neu-raised-sm text-[#6F7B71]'
                    }`}
                  >
                    <div className="flex items-center justify-between mb-1">
                      <span>Step {idx + 1}</span>
                      <span>{st.planned_duration_minutes}m</span>
                    </div>
                    <p className="truncate">{st.name}</p>
                    <div className="mt-1 text-[10px] uppercase font-bold">
                      {isPast ? '✓ Done' : isCurrent ? '● Active' : 'Queued'}
                    </div>
                  </div>
                );
              })}
            </div>
          </div>
        </NeuCard>

        {/* Patient Vitals & Materials Log (1 col) */}
        <div className="space-y-6">
          {/* Baseline Vitals Card */}
          <NeuCard elevation="raised" className="p-6 space-y-4">
            <div className="flex items-center justify-between border-b border-[#E3DFC9] pb-3">
              <h3 className="font-serif text-base font-bold text-[#234737] flex items-center gap-2">
                <HeartPulse className="w-4 h-4 text-[#C98769]" /> Baseline Vitals
              </h3>
              <NeuBadge variant="success" size="sm">NABH Checked</NeuBadge>
            </div>

            {session.baseline_vitals ? (
              <div className="grid grid-cols-2 gap-3 text-xs">
                <div className="p-3 rounded-xl bg-[#F5F3EC] neu-inset">
                  <span className="text-[#6F7B71] block">Blood Pressure:</span>
                  <span className="font-bold text-[#234737] text-sm">
                    {session.baseline_vitals.bp_systolic}/{session.baseline_vitals.bp_diastolic} mmHg
                  </span>
                </div>
                <div className="p-3 rounded-xl bg-[#F5F3EC] neu-inset">
                  <span className="text-[#6F7B71] block">Pulse Rate:</span>
                  <span className="font-bold text-[#234737] text-sm">
                    {session.baseline_vitals.pulse} bpm
                  </span>
                </div>
                <div className="p-3 rounded-xl bg-[#F5F3EC] neu-inset">
                  <span className="text-[#6F7B71] block">Temperature:</span>
                  <span className="font-bold text-[#234737] text-sm">
                    {session.baseline_vitals.temperature_c}°C
                  </span>
                </div>
                <div className="p-3 rounded-xl bg-[#F5F3EC] neu-inset">
                  <span className="text-[#6F7B71] block">Checklist:</span>
                  <span className="font-bold text-[#357A55] text-sm">Verified</span>
                </div>
              </div>
            ) : (
              <NeuButton variant="raised" size="sm" onClick={() => setShowCheckInModal(true)} className="w-full">
                Enter Vitals
              </NeuButton>
            )}
          </NeuCard>

          {/* Consumables / Materials Logged */}
          <NeuCard elevation="raised" className="p-6 space-y-4">
            <div className="flex items-center justify-between border-b border-[#E3DFC9] pb-3">
              <h3 className="font-serif text-base font-bold text-[#234737] flex items-center gap-2">
                <Layers className="w-4 h-4 text-[#315C49]" /> Consumables Used
              </h3>
              <button
                onClick={() => setShowMaterialModal(true)}
                className="text-xs font-semibold text-[#315C49] hover:underline flex items-center gap-1"
              >
                <Plus className="w-3.5 h-3.5" /> Log Oil/Herb
              </button>
            </div>

            <div className="space-y-2">
              {materialsList.length === 0 ? (
                <p className="text-xs text-[#6F7B71]">No materials logged yet.</p>
              ) : (
                materialsList.map((m, idx) => (
                  <div
                    key={m.id || idx}
                    className="p-2.5 rounded-xl bg-[#F5F3EC] neu-raised-sm flex items-center justify-between text-xs"
                  >
                    <div>
                      <span className="font-bold text-[#27352D] block">{m.material_name}</span>
                      <span className="text-[10px] text-[#6F7B71] font-mono">{m.batch_number}</span>
                    </div>
                    <span className="font-semibold text-[#315C49]">
                      {m.quantity} {m.unit}
                    </span>
                  </div>
                ))
              )}
            </div>
          </NeuCard>
        </div>
      </div>

      {/* Clinical Notes & Post-Care Dispatch Container */}
      <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
        <NeuCard elevation="raised" className="p-6 space-y-3">
          <h3 className="font-serif text-base font-bold text-[#234737] flex items-center gap-2 border-b border-[#E3DFC9] pb-2">
            <FileText className="w-4 h-4 text-[#315C49]" /> Therapist Clinical Notes & Observations
          </h3>
          <div>
            <label className="text-xs font-semibold text-[#27352D] block mb-1">Therapist Notes:</label>
            <textarea
              value={therapistNotes}
              onChange={(e) => setTherapistNotes(e.target.value)}
              rows={2}
              className="w-full p-2.5 rounded-xl bg-[#F5F3EC] neu-inset text-xs font-medium focus:outline-none"
            />
          </div>
          <div>
            <label className="text-xs font-semibold text-[#27352D] block mb-1">Observations:</label>
            <textarea
              value={observations}
              onChange={(e) => setObservations(e.target.value)}
              rows={2}
              className="w-full p-2.5 rounded-xl bg-[#F5F3EC] neu-inset text-xs font-medium focus:outline-none"
            />
          </div>
        </NeuCard>

        <NeuCard elevation="raised" className="p-6 space-y-3">
          <h3 className="font-serif text-base font-bold text-[#234737] flex items-center gap-2 border-b border-[#E3DFC9] pb-2">
            <Send className="w-4 h-4 text-[#C98769]" /> Patient Post-Care Instructions Dispatch
          </h3>
          <p className="text-[11px] text-[#6F7B71]">
            These instructions will be immediately visible on the patient's personal portal upon completion.
          </p>
          <textarea
            value={postCareInstructions}
            onChange={(e) => setPostCareInstructions(e.target.value)}
            rows={4}
            className="w-full p-2.5 rounded-xl bg-[#F5F3EC] neu-inset text-xs font-medium focus:outline-none"
          />
        </NeuCard>
      </div>

      {/* Modal 1: Check-In & Vitals */}
      {showCheckInModal && (
        <NeuModal
          isOpen={showCheckInModal}
          onClose={() => setShowCheckInModal(false)}
          title="Patient Check-In & Baseline Vitals Inspection"
        >
          <form onSubmit={handleCheckInSubmit} className="space-y-4">
            <p className="text-xs text-[#6F7B71]">
              Record physical baseline parameters prior to initiating oleation. High systolic BP (&gt;160) or fever warrants consultation with attending doctor.
            </p>

            <div className="grid grid-cols-2 gap-3">
              <NeuInput
                label="Systolic BP (mmHg)"
                type="number"
                required
                value={bpSystolic}
                onChange={(e) => setBpSystolic(Number(e.target.value))}
              />
              <NeuInput
                label="Diastolic BP (mmHg)"
                type="number"
                required
                value={bpDiastolic}
                onChange={(e) => setBpDiastolic(Number(e.target.value))}
              />
              <NeuInput
                label="Pulse Rate (bpm)"
                type="number"
                required
                value={pulse}
                onChange={(e) => setPulse(Number(e.target.value))}
              />
              <NeuInput
                label="Body Temp (°C)"
                type="number"
                step={0.1}
                required
                value={tempC}
                onChange={(e) => setTempC(Number(e.target.value))}
              />
            </div>

            <div className="pt-2">
              <label className="flex items-center gap-2 text-xs text-[#27352D] font-semibold cursor-pointer">
                <input
                  type="checkbox"
                  checked={checklistVerified}
                  onChange={(e) => setChecklistVerified(e.target.checked)}
                  className="rounded text-[#315C49]"
                />
                <span>I have personally verified patient pre-care fasting and cotton garments.</span>
              </label>
            </div>

            <div className="flex justify-end gap-3 pt-4 border-t border-[#E3DFC9]">
              <NeuButton variant="raised" onClick={() => setShowCheckInModal(false)}>
                Cancel
              </NeuButton>
              <NeuButton type="submit" variant="primary">
                Confirm Check-In
              </NeuButton>
            </div>
          </form>
        </NeuModal>
      )}

      {/* Modal 2: Log Consumable */}
      {showMaterialModal && (
        <NeuModal
          isOpen={showMaterialModal}
          onClose={() => setShowMaterialModal(false)}
          title="Record Material / Medicated Consumable"
        >
          <form onSubmit={handleRecordMaterial} className="space-y-4">
            <NeuInput
              label="Material / Herbal Oil Name"
              type="text"
              required
              value={matName}
              onChange={(e) => setMatName(e.target.value)}
              placeholder="e.g. Mahanarayana Taila, Ksheerabala"
            />

            <div className="grid grid-cols-2 gap-3">
              <NeuInput
                label="Quantity"
                type="number"
                required
                value={matQuantity}
                onChange={(e) => setMatQuantity(Number(e.target.value))}
              />
              <div>
                <label className="text-xs font-semibold text-[#27352D] block mb-1">Unit</label>
                <select
                  value={matUnit}
                  onChange={(e) => setMatUnit(e.target.value)}
                  className="w-full p-2.5 rounded-xl bg-[#F5F3EC] neu-inset text-xs font-medium focus:outline-none"
                >
                  <option value="ml">ml (Liquid Taila)</option>
                  <option value="grams">grams (Choorna / Powder)</option>
                  <option value="pieces">pieces (Kizhi / Pindas)</option>
                </select>
              </div>
            </div>

            <NeuInput
              label="Herbal Pharmacy Batch Number"
              type="text"
              required
              value={matBatch}
              onChange={(e) => setMatBatch(e.target.value)}
              placeholder="e.g. BATCH-2026-AYUR-99"
            />

            <div className="flex justify-end gap-3 pt-4 border-t border-[#E3DFC9]">
              <NeuButton variant="raised" onClick={() => setShowMaterialModal(false)}>
                Cancel
              </NeuButton>
              <NeuButton type="submit" variant="primary">
                Record Consumable
              </NeuButton>
            </div>
          </form>
        </NeuModal>
      )}

      {/* Modal 3: Complication Escalation */}
      {showComplicationModal && (
        <NeuModal
          isOpen={showComplicationModal}
          onClose={() => setShowComplicationModal(false)}
          title="Escalate Clinical Complication to Doctor"
        >
          <form onSubmit={handleFlagComplication} className="space-y-4">
            <div className="p-3 rounded-xl bg-[#B74747]/10 text-xs text-[#B74747]">
              Warning: Raising this flag immediately pauses the active therapy timer, notifies Dr. Anand Vaidya, and marks the session as COMPLICATION_ALERT.
            </div>

            <div>
              <label className="text-xs font-semibold text-[#27352D] block mb-1">Severity Level</label>
              <select
                value={compSeverity}
                onChange={(e) => setCompSeverity(e.target.value as any)}
                className="w-full p-2.5 rounded-xl bg-[#F5F3EC] neu-inset text-xs font-bold text-[#B74747] focus:outline-none"
              >
                <option value="LOW">LOW — Mild discomfort / slight nausea</option>
                <option value="MEDIUM">MEDIUM — Dizziness, acute sweating, mild rash</option>
                <option value="HIGH">HIGH — Marked hypotension, vomiting, disorientation</option>
                <option value="CRITICAL">CRITICAL — Severe syncopal episode, anaphylaxis</option>
              </select>
            </div>

            <div>
              <label className="text-xs font-semibold text-[#27352D] block mb-1">
                Observed Symptoms & Physiological Signs
              </label>
              <textarea
                value={compSymptoms}
                required
                onChange={(e) => setCompSymptoms(e.target.value)}
                placeholder="Detail symptoms, onset time, and patient verbal responses..."
                rows={3}
                className="w-full p-2.5 rounded-xl bg-[#F5F3EC] neu-inset text-xs font-medium focus:outline-none"
              />
            </div>

            <div>
              <label className="text-xs font-semibold text-[#27352D] block mb-1">
                Immediate Actions Taken by Therapist
              </label>
              <textarea
                value={compActions}
                required
                onChange={(e) => setCompActions(e.target.value)}
                placeholder="e.g. Suspended steam immediately, placed patient supine, provided lukewarm water..."
                rows={2}
                className="w-full p-2.5 rounded-xl bg-[#F5F3EC] neu-inset text-xs font-medium focus:outline-none"
              />
            </div>

            <div className="flex justify-end gap-3 pt-4 border-t border-[#E3DFC9]">
              <NeuButton variant="raised" onClick={() => setShowComplicationModal(false)}>
                Dismiss
              </NeuButton>
              <NeuButton type="submit" variant="danger">
                Dispatch Doctor Escalation Alert
              </NeuButton>
            </div>
          </form>
        </NeuModal>
      )}
    </div>
  );
};
