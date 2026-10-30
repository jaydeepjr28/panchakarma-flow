import React, { useState, useEffect } from 'react';
import { useSearchParams } from 'react-router-dom';
import { NeuCard } from '@/components/ui/NeuCard';
import { NeuButton } from '@/components/ui/NeuButton';
import { NeuBadge } from '@/components/ui/NeuBadge';
import { NeuInput } from '@/components/ui/NeuInput';
import { therapyApi } from '@/lib/api/client';
import { TherapySession } from '@/lib/api/types';
import {
  HeartPulse,
  Plus,
  CheckCircle2,
  AlertTriangle,
  Calendar,
  Send,
  FileCheck,
} from 'lucide-react';

export const TherapistFollowUpsPage: React.FC = () => {
  const [searchParams] = useSearchParams();
  const [sessions, setSessions] = useState<TherapySession[]>([]);
  const [selectedSessionId, setSelectedSessionId] = useState<string>(searchParams.get('sessionId') || '');
  const [instructions, setInstructions] = useState(
    'Continue with daily warm water sips. Schedule follow-up Abhyanga review after 7 days if lumbar stiffness persists.'
  );
  const [intervalDays, setIntervalDays] = useState(7);
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [successMsg, setSuccessMsg] = useState('');
  const [errorMsg, setErrorMsg] = useState('');

  useEffect(() => {
    therapyApi
      .listTherapistAssigned()
      .then((data) => {
        setSessions(data);
        if (!selectedSessionId && data.length > 0) {
          setSelectedSessionId(data[0].id);
        }
      })
      .catch((err) => console.error(err));
  }, []);

  const handleCreateFollowUp = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!selectedSessionId) {
      setErrorMsg('Please select a session to log follow-up instructions.');
      return;
    }
    setIsSubmitting(true);
    setErrorMsg('');
    try {
      await therapyApi.createFollowUp(selectedSessionId, {
        recommended_by: 'Ayurvedic Therapist',
        recommended_interval_days: intervalDays,
        instructions,
      });
      setSuccessMsg('Follow-up recommendation recorded and added to patient care cycle.');
    } catch (err: any) {
      setErrorMsg(err.message || 'Failed to submit follow-up');
    } finally {
      setIsSubmitting(false);
    }
  };

  return (
    <div className="space-y-6 max-w-4xl mx-auto">
      <div>
        <p className="text-xs uppercase tracking-widest font-bold text-[#357A55] mb-1">
          Post-Care & Continued Protocol
        </p>
        <h1 className="font-serif text-3xl font-bold text-[#234737]">
          Follow-Up Recommendations & Complications Log
        </h1>
        <p className="text-xs text-[#6F7B71]">
          Record post-therapy observations, recommended subsequent sessions, and review clinical complication audit logs.
        </p>
      </div>

      {successMsg && (
        <div className="p-4 rounded-xl bg-[#357A55]/10 border border-[#357A55]/20 text-xs text-[#357A55] flex items-center justify-between">
          <span className="font-semibold">{successMsg}</span>
          <button onClick={() => setSuccessMsg('')} className="font-bold">✕</button>
        </div>
      )}

      {errorMsg && (
        <div className="p-4 rounded-xl bg-[#C15C3D]/10 border border-[#C15C3D]/20 text-xs text-[#C15C3D] flex items-center justify-between">
          <span className="font-semibold">{errorMsg}</span>
          <button onClick={() => setErrorMsg('')} className="font-bold">✕</button>
        </div>
      )}

      {/* Form: Add Follow-up */}
      <NeuCard elevation="raised" className="p-6 space-y-4">
        <div className="flex items-center gap-2 border-b border-[#E3DFC9] pb-3">
          <HeartPulse className="w-5 h-5 text-[#315C49]" />
          <h2 className="font-serif text-lg font-bold text-[#234737]">
            Submit Recommended Follow-Up Care
          </h2>
        </div>

        <form onSubmit={handleCreateFollowUp} className="space-y-4">
          <div>
            <label className="text-xs font-semibold text-[#27352D] block mb-1">
              Therapy Session *
            </label>
            {sessions.length > 0 ? (
              <select
                value={selectedSessionId}
                onChange={(e) => setSelectedSessionId(e.target.value)}
                className="w-full px-4 py-2.5 rounded-xl bg-[#F5F3EC] neu-inset text-xs font-medium text-[#27352D] focus:outline-none"
              >
                {sessions.map((s) => (
                  <option key={s.id} value={s.id}>
                    Session #{s.id.slice(0, 8)} — Patient {s.patient_id.slice(0, 8)} ({s.status})
                  </option>
                ))}
              </select>
            ) : (
              <input
                type="text"
                value={selectedSessionId}
                onChange={(e) => setSelectedSessionId(e.target.value)}
                placeholder="Enter Session UUID..."
                className="w-full px-4 py-2.5 rounded-xl bg-[#F5F3EC] neu-inset text-xs font-medium text-[#27352D] focus:outline-none font-mono"
              />
            )}
          </div>

          <div>
            <label className="text-xs font-semibold text-[#27352D] block mb-1">
              Recommended Interval (Days until next session)
            </label>
            <input
              type="number"
              min={1}
              max={30}
              value={intervalDays}
              onChange={(e) => setIntervalDays(Number(e.target.value))}
              className="w-full px-4 py-2.5 rounded-xl bg-[#F5F3EC] neu-inset text-xs font-semibold text-[#27352D] focus:outline-none"
            />
          </div>

          <div>
            <label className="text-xs font-semibold text-[#27352D] block mb-1">
              Specific Post-Care & Dietary Guidance for Patient
            </label>
            <textarea
              value={instructions}
              onChange={(e) => setInstructions(e.target.value)}
              rows={3}
              required
              className="w-full p-3 rounded-xl bg-[#F5F3EC] neu-inset text-xs font-medium text-[#27352D] focus:outline-none"
            />
          </div>

          <div className="flex justify-end pt-2">
            <NeuButton
              type="submit"
              variant="primary"
              size="md"
              isLoading={isSubmitting}
              className="flex items-center gap-2"
            >
              <Send className="w-4 h-4" /> Save Follow-Up Recommendation
            </NeuButton>
          </div>
        </form>
      </NeuCard>

      {/* Recent Complications Audit Trail */}
      <NeuCard elevation="raised" className="p-6 space-y-4">
        <h3 className="font-serif text-lg font-bold text-[#234737] border-b border-[#E3DFC9] pb-3">
          Clinical Complications & Escalation Log
        </h3>

        <div className="space-y-3">
          <div className="p-4 rounded-xl bg-[#F5F3EC] neu-inset text-xs space-y-2">
            <div className="flex items-center justify-between">
              <span className="font-bold text-[#234737]">Incident #COMP-2026-10-24</span>
              <NeuBadge variant="success" size="sm">RESOLVED</NeuBadge>
            </div>
            <p className="text-[#6F7B71]">
              <strong className="text-[#27352D]">Symptoms:</strong> Transient lightheadedness during post-steam rest.
            </p>
            <p className="text-[#6F7B71]">
              <strong className="text-[#27352D]">Action:</strong> Patient positioned supine, oral electrolytes provided. Dr. Anand Vaidya cleared discharge after 30-minute vitals check.
            </p>
          </div>
        </div>
      </NeuCard>
    </div>
  );
};
