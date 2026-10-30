import React, { useState } from 'react';
import { NeuCard } from '@/components/ui/NeuCard';
import { NeuButton } from '@/components/ui/NeuButton';
import { NeuBadge } from '@/components/ui/NeuBadge';
import { NeuCheckbox } from '@/components/ui/NeuCheckbox';
import {
  ClipboardList,
  CheckCircle2,
  AlertTriangle,
  Info,
  ShieldCheck,
  Save,
} from 'lucide-react';

export const PatientChecklistsPage: React.FC = () => {
  const [checklist, setChecklist] = useState({
    fasting_completed: true,
    warm_water_hydration: true,
    light_diet_observed: false,
    cotton_clothing_ready: false,
    no_intense_exercise: true,
    medication_disclosed: true,
  });

  const [savedSuccess, setSavedSuccess] = useState(false);
  const [isSaving, setIsSaving] = useState(false);

  const toggleItem = (key: keyof typeof checklist) => {
    setChecklist((prev) => ({ ...prev, [key]: !prev[key] }));
    setSavedSuccess(false);
  };

  const handleSave = () => {
    setIsSaving(true);
    setTimeout(() => {
      setIsSaving(false);
      setSavedSuccess(true);
    }, 400);
  };

  const completedCount = Object.values(checklist).filter(Boolean).length;
  const totalCount = Object.keys(checklist).length;
  const progressPct = Math.round((completedCount / totalCount) * 100);

  return (
    <div className="space-y-6 max-w-4xl mx-auto">
      <div>
        <p className="text-xs uppercase tracking-widest font-bold text-[#C98769] mb-1">
          Safety Protocols & Preparation
        </p>
        <h1 className="font-serif text-3xl font-bold text-[#234737]">
          Pre-Therapy Patient Checklist
        </h1>
        <p className="text-xs text-[#6F7B71]">
          Accurate adherence to pre-care protocols is essential to prevent adverse physiological reactions during Panchakarma.
        </p>
      </div>

      {savedSuccess && (
        <div className="p-4 rounded-xl bg-[#357A55]/10 border border-[#357A55]/20 text-xs text-[#357A55] flex items-center gap-2">
          <CheckCircle2 className="w-4 h-4 shrink-0" />
          <span>Checklist responses successfully recorded on your patient record. Attending therapist will verify vitals at check-in.</span>
        </div>
      )}

      {/* Progress Card */}
      <NeuCard elevation="raised" className="p-6">
        <div className="flex items-center justify-between mb-3">
          <div className="flex items-center gap-2">
            <ClipboardList className="w-5 h-5 text-[#315C49]" />
            <h2 className="font-bold text-base text-[#234737]">Readiness Score</h2>
          </div>
          <NeuBadge variant={progressPct === 100 ? 'success' : 'warning'}>
            {completedCount} of {totalCount} Items Completed ({progressPct}%)
          </NeuBadge>
        </div>

        <div className="w-full h-3 rounded-full bg-[#F5F3EC] neu-inset p-0.5 mb-2">
          <div
            className={`h-full rounded-full transition-all duration-500 ${
              progressPct === 100 ? 'bg-[#357A55]' : 'bg-[#C98769]'
            }`}
            style={{ width: `${progressPct}%` }}
          />
        </div>

        <div className="p-3 rounded-xl bg-[#F5F3EC] neu-inset text-[11px] text-[#6F7B71] flex items-center gap-2 mt-4">
          <Info className="w-4 h-4 text-[#315C49] shrink-0" />
          <span>
            Note: Completing this form documents your preparation. Final clinical readiness is verified by the attending therapist during baseline vitals inspection.
          </span>
        </div>
      </NeuCard>

      {/* Checklist Form */}
      <NeuCard elevation="raised" className="p-6 space-y-5">
        <h3 className="font-serif text-lg font-bold text-[#234737] border-b border-[#E3DFC9] pb-3">
          Preparation Items for Upcoming Session
        </h3>

        <div className="space-y-4">
          <div
            onClick={() => toggleItem('fasting_completed')}
            className={`p-4 rounded-xl cursor-pointer transition-all flex items-start gap-3.5 ${
              checklist.fasting_completed ? 'neu-inset bg-[#357A55]/5 border border-[#357A55]/30' : 'neu-raised-sm hover:neu-pressed'
            }`}
          >
            <input
              type="checkbox"
              checked={checklist.fasting_completed}
              onChange={() => {}}
              className="mt-1 rounded text-[#315C49] focus:ring-[#315C49]"
            />
            <div>
              <p className="text-xs font-bold text-[#27352D]">
                Observed Minimum 3-Hour Fast Prior to Appointment
              </p>
              <p className="text-[11px] text-[#6F7B71] mt-0.5">
                Stomach should be free of heavy food to prevent nausea and abdominal distress during abdominal massage (Abhyanga).
              </p>
            </div>
          </div>

          <div
            onClick={() => toggleItem('warm_water_hydration')}
            className={`p-4 rounded-xl cursor-pointer transition-all flex items-start gap-3.5 ${
              checklist.warm_water_hydration ? 'neu-inset bg-[#357A55]/5 border border-[#357A55]/30' : 'neu-raised-sm hover:neu-pressed'
            }`}
          >
            <input
              type="checkbox"
              checked={checklist.warm_water_hydration}
              onChange={() => {}}
              className="mt-1 rounded text-[#315C49] focus:ring-[#315C49]"
            />
            <div>
              <p className="text-xs font-bold text-[#27352D]">
                Adequate Warm Water Hydration (Ushnodaka)
              </p>
              <p className="text-[11px] text-[#6F7B71] mt-0.5">
                Drank at least 2 cups of lukewarm water 1 hour prior to encourage sweat gland responsiveness for Swedana (steam).
              </p>
            </div>
          </div>

          <div
            onClick={() => toggleItem('light_diet_observed')}
            className={`p-4 rounded-xl cursor-pointer transition-all flex items-start gap-3.5 ${
              checklist.light_diet_observed ? 'neu-inset bg-[#357A55]/5 border border-[#357A55]/30' : 'neu-raised-sm hover:neu-pressed'
            }`}
          >
            <input
              type="checkbox"
              checked={checklist.light_diet_observed}
              onChange={() => {}}
              className="mt-1 rounded text-[#315C49] focus:ring-[#315C49]"
            />
            <div>
              <p className="text-xs font-bold text-[#27352D]">
                Avoided Fermented, Deep-Fried & Chilled Foods (Laghu Ahara)
              </p>
              <p className="text-[11px] text-[#6F7B71] mt-0.5">
                No refrigerated water, cold sodas, or dairy cheese for the preceding 24 hours.
              </p>
            </div>
          </div>

          <div
            onClick={() => toggleItem('cotton_clothing_ready')}
            className={`p-4 rounded-xl cursor-pointer transition-all flex items-start gap-3.5 ${
              checklist.cotton_clothing_ready ? 'neu-inset bg-[#357A55]/5 border border-[#357A55]/30' : 'neu-raised-sm hover:neu-pressed'
            }`}
          >
            <input
              type="checkbox"
              checked={checklist.cotton_clothing_ready}
              onChange={() => {}}
              className="mt-1 rounded text-[#315C49] focus:ring-[#315C49]"
            />
            <div>
              <p className="text-xs font-bold text-[#27352D]">
                Wearing Loose, Breathable Cotton Garments
              </p>
              <p className="text-[11px] text-[#6F7B71] mt-0.5">
                Protects skin after herbal oil application and accommodates medicated vapors without constriction.
              </p>
            </div>
          </div>

          <div
            onClick={() => toggleItem('no_intense_exercise')}
            className={`p-4 rounded-xl cursor-pointer transition-all flex items-start gap-3.5 ${
              checklist.no_intense_exercise ? 'neu-inset bg-[#357A55]/5 border border-[#357A55]/30' : 'neu-raised-sm hover:neu-pressed'
            }`}
          >
            <input
              type="checkbox"
              checked={checklist.no_intense_exercise}
              onChange={() => {}}
              className="mt-1 rounded text-[#315C49] focus:ring-[#315C49]"
            />
            <div>
              <p className="text-xs font-bold text-[#27352D]">
                Refrained from Strenuous Physical Workouts / Gym
              </p>
              <p className="text-[11px] text-[#6F7B71] mt-0.5">
                Conserves Ojas and prevents cardiovascular strain before thermal herbal treatments.
              </p>
            </div>
          </div>

          <div
            onClick={() => toggleItem('medication_disclosed')}
            className={`p-4 rounded-xl cursor-pointer transition-all flex items-start gap-3.5 ${
              checklist.medication_disclosed ? 'neu-inset bg-[#357A55]/5 border border-[#357A55]/30' : 'neu-raised-sm hover:neu-pressed'
            }`}
          >
            <input
              type="checkbox"
              checked={checklist.medication_disclosed}
              onChange={() => {}}
              className="mt-1 rounded text-[#315C49] focus:ring-[#315C49]"
            />
            <div>
              <p className="text-xs font-bold text-[#27352D]">
                All Recent Allopathic & Ayurvedic Medications Disclosed
              </p>
              <p className="text-[11px] text-[#6F7B71] mt-0.5">
                Ensures no counter-indications with medicated oils (Tailas) or herbal decoctions (Kashayas).
              </p>
            </div>
          </div>
        </div>

        <div className="pt-4 border-t border-[#E3DFC9] flex justify-end">
          <NeuButton
            variant="primary"
            size="md"
            isLoading={isSaving}
            onClick={handleSave}
            className="flex items-center gap-2"
          >
            <Save className="w-4 h-4" /> Save Checklist Responses
          </NeuButton>
        </div>
      </NeuCard>
    </div>
  );
};
