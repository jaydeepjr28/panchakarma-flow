import React, { useEffect, useState } from 'react';
import { NeuCard } from '@/components/ui/NeuCard';
import { NeuButton } from '@/components/ui/NeuButton';
import { NeuBadge } from '@/components/ui/NeuBadge';
import { NeuModal } from '@/components/ui/NeuModal';
import { NeuInput } from '@/components/ui/NeuInput';
import { NeuFeedbackStates } from '@/components/ui/NeuFeedbackStates';
import { doctorApi } from '@/lib/api/client';
import { Doctor, DoctorLeave } from '@/lib/api/types';
import { useAuth } from '@/context/AuthContext';
import {
  Stethoscope,
  Calendar,
  Clock,
  Plus,
  AlertTriangle,
  CheckCircle2,
  CalendarOff,
} from 'lucide-react';

export const ClinicDoctorsPage: React.FC = () => {
  const { user } = useAuth();
  const clinicId = user?.clinic_id || 'a3f8c872-9112-4c22-b5e1-7e8913c2f101';
  const [doctors, setDoctors] = useState<Doctor[]>([]);
  const [isLoading, setIsLoading] = useState(true);

  // Leave Modal State
  const [selectedDocForLeave, setSelectedDocForLeave] = useState<Doctor | null>(null);
  const [startDate, setStartDate] = useState('');
  const [endDate, setEndDate] = useState('');
  const [leaveReason, setLeaveReason] = useState('');
  const [isSubmittingLeave, setIsSubmittingLeave] = useState(false);
  const [successMsg, setSuccessMsg] = useState('');

  const loadDoctors = () => {
    setIsLoading(true);
    doctorApi
      .listByClinic(clinicId)
      .then((data) => setDoctors(data))
      .catch((err) => console.error(err))
      .finally(() => setIsLoading(false));
  };

  useEffect(() => {
    loadDoctors();
  }, [clinicId]);

  const handleCreateLeave = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!selectedDocForLeave || !startDate || !endDate) return;

    setIsSubmittingLeave(true);
    try {
      await doctorApi.createLeave(selectedDocForLeave.id, {
        start_date: startDate,
        end_date: endDate,
        reason: leaveReason || 'Planned medical leave',
      });
      setSuccessMsg(`Leave recorded for ${selectedDocForLeave.full_name}. Overlapping booking slots are now blocked.`);
      setSelectedDocForLeave(null);
      setStartDate('');
      setEndDate('');
      setLeaveReason('');
    } catch (err: any) {
      alert(err.message || 'Failed to record leave exception');
    } finally {
      setIsSubmittingLeave(false);
    }
  };

  return (
    <div className="space-y-6">
      <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4">
        <div>
          <p className="text-xs uppercase tracking-widest font-bold text-[#C98769] mb-1">
            Staff & Availability
          </p>
          <h1 className="font-serif text-3xl font-bold text-[#234737]">
            Consulting Doctors & Schedules
          </h1>
          <p className="text-xs text-[#6F7B71]">
            Manage doctor clinical specializations, weekly recurring rosters, and planned leave exceptions.
          </p>
        </div>
      </div>

      {successMsg && (
        <div className="p-4 rounded-xl bg-[#357A55]/10 border border-[#357A55]/20 text-xs text-[#357A55] flex items-center justify-between">
          <span>{successMsg}</span>
          <button onClick={() => setSuccessMsg('')} className="font-bold">✕</button>
        </div>
      )}

      {isLoading ? (
        <NeuFeedbackStates.Skeleton count={3} />
      ) : (
        <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
          {doctors.map((doc) => (
            <NeuCard key={doc.id} elevation="raised" className="p-6 flex flex-col justify-between space-y-4">
              <div>
                <div className="flex items-start justify-between gap-3 mb-2">
                  <div className="flex items-center gap-3">
                    <div className="w-12 h-12 rounded-2xl bg-[#315C49]/10 text-[#315C49] flex items-center justify-center neu-inset shrink-0">
                      <Stethoscope className="w-6 h-6" />
                    </div>
                    <div>
                      <h2 className="font-serif text-lg font-bold text-[#234737]">{doc.full_name}</h2>
                      <p className="text-xs text-[#C98769] font-medium">{doc.qualifications}</p>
                    </div>
                  </div>
                  <NeuBadge variant={doc.is_active ? 'forest' : 'error'} size="sm">
                    {doc.is_active ? 'Active' : 'Inactive'}
                  </NeuBadge>
                </div>

                <div className="space-y-2 text-xs pt-3 border-t border-[#E3DFC9]">
                  <div>
                    <span className="text-[#6F7B71] block">Specialties:</span>
                    <span className="font-semibold text-[#27352D]">{doc.specialties.join(', ')}</span>
                  </div>
                  <div>
                    <span className="text-[#6F7B71] block">Weekly Standard Hours:</span>
                    <span className="font-semibold text-[#315C49]">Mon - Sat: 09:00 - 17:00 IST (60m slots)</span>
                  </div>
                  <div>
                    <span className="text-[#6F7B71] block">Max Daily Capacity:</span>
                    <span className="font-semibold text-[#27352D]">{doc.max_daily_capacity || 8} sessions/day</span>
                  </div>
                </div>
              </div>

              <div className="pt-4 border-t border-[#E3DFC9] flex items-center gap-3">
                <NeuButton
                  variant="raised"
                  size="sm"
                  onClick={() => setSelectedDocForLeave(doc)}
                  className="w-full flex items-center justify-center gap-1.5"
                >
                  <CalendarOff className="w-3.5 h-3.5 text-[#C98769]" />
                  Add Leave Exception
                </NeuButton>
              </div>
            </NeuCard>
          ))}
        </div>
      )}

      {/* Leave Modal */}
      {selectedDocForLeave && (
        <NeuModal
          isOpen={!!selectedDocForLeave}
          onClose={() => setSelectedDocForLeave(null)}
          title={`Record Leave Exception — ${selectedDocForLeave.full_name}`}
        >
          <form onSubmit={handleCreateLeave} className="space-y-4">
            <p className="text-xs text-[#6F7B71]">
              Dates marked here will automatically block booking availability and offer safe cancellation alerts for existing provisional holds.
            </p>

            <div className="grid grid-cols-2 gap-3">
              <div>
                <label className="text-xs font-semibold text-[#27352D] block mb-1">Start Date</label>
                <input
                  type="date"
                  required
                  value={startDate}
                  onChange={(e) => setStartDate(e.target.value)}
                  className="w-full p-2.5 rounded-xl bg-[#F5F3EC] neu-inset text-xs font-medium focus:outline-none"
                />
              </div>
              <div>
                <label className="text-xs font-semibold text-[#27352D] block mb-1">End Date</label>
                <input
                  type="date"
                  required
                  value={endDate}
                  onChange={(e) => setEndDate(e.target.value)}
                  className="w-full p-2.5 rounded-xl bg-[#F5F3EC] neu-inset text-xs font-medium focus:outline-none"
                />
              </div>
            </div>

            <div>
              <label className="text-xs font-semibold text-[#27352D] block mb-1">Reason for Leave</label>
              <textarea
                value={leaveReason}
                onChange={(e) => setLeaveReason(e.target.value)}
                placeholder="e.g. Annual Ayush conference, personal medical..."
                rows={3}
                className="w-full p-2.5 rounded-xl bg-[#F5F3EC] neu-inset text-xs font-medium focus:outline-none"
              />
            </div>

            <div className="flex justify-end gap-3 pt-4 border-t border-[#E3DFC9]">
              <NeuButton variant="raised" onClick={() => setSelectedDocForLeave(null)}>
                Cancel
              </NeuButton>
              <NeuButton type="submit" variant="primary" isLoading={isSubmittingLeave}>
                Save Leave Exception
              </NeuButton>
            </div>
          </form>
        </NeuModal>
      )}
    </div>
  );
};
