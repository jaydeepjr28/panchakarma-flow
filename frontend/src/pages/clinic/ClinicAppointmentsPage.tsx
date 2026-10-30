import React, { useEffect, useState } from 'react';
import { NeuCard } from '@/components/ui/NeuCard';
import { NeuButton } from '@/components/ui/NeuButton';
import { NeuBadge } from '@/components/ui/NeuBadge';
import { NeuModal } from '@/components/ui/NeuModal';
import { NeuFeedbackStates } from '@/components/ui/NeuFeedbackStates';
import { appointmentApi, doctorApi, roomApi } from '@/lib/api/client';
import { Appointment, Doctor, TreatmentRoom } from '@/lib/api/types';
import { useAuth } from '@/context/AuthContext';
import {
  Calendar,
  Clock,
  Filter,
  Search,
  XCircle,
  RefreshCw,
  Plus,
  Stethoscope,
  DoorOpen,
} from 'lucide-react';

export const ClinicAppointmentsPage: React.FC = () => {
  const { user } = useAuth();
  const clinicId = user?.clinic_id || 'a3f8c872-9112-4c22-b5e1-7e8913c2f101';
  const [appointments, setAppointments] = useState<Appointment[]>([]);
  const [doctors, setDoctors] = useState<Doctor[]>([]);
  const [rooms, setRooms] = useState<TreatmentRoom[]>([]);
  const [isLoading, setIsLoading] = useState(true);

  // Filters
  const [statusFilter, setStatusFilter] = useState('ALL');
  const [doctorFilter, setDoctorFilter] = useState('ALL');
  const [dateFilter, setDateFilter] = useState('');

  // Cancel Modal
  const [selectedAppt, setSelectedAppt] = useState<Appointment | null>(null);
  const [cancelReason, setCancelReason] = useState('');
  const [isProcessing, setIsProcessing] = useState(false);

  const loadData = () => {
    setIsLoading(true);
    Promise.all([
      appointmentApi.listClinicAppointments(clinicId),
      doctorApi.listByClinic(clinicId),
      roomApi.listByClinic(clinicId),
    ])
      .then(([appts, docs, rms]) => {
        setAppointments(appts);
        setDoctors(docs);
        setRooms(rms);
      })
      .catch((err) => console.error(err))
      .finally(() => setIsLoading(false));
  };

  useEffect(() => {
    loadData();
  }, [clinicId]);

  const handleCancel = async () => {
    if (!selectedAppt) return;
    setIsProcessing(true);
    try {
      await appointmentApi.cancelAppointment(selectedAppt.id, cancelReason || 'Clinic admin cancellation');
      setSelectedAppt(null);
      setCancelReason('');
      loadData();
    } catch (err: any) {
      alert(err.message || 'Failed to cancel appointment');
    } finally {
      setIsProcessing(false);
    }
  };

  const filteredAppointments = appointments.filter((a) => {
    const matchesStatus = statusFilter === 'ALL' || a.status === statusFilter;
    const matchesDoctor = doctorFilter === 'ALL' || a.doctor_id === doctorFilter;
    const matchesDate = !dateFilter || a.start_time.startsWith(dateFilter);
    return matchesStatus && matchesDoctor && matchesDate;
  });

  return (
    <div className="space-y-6">
      <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4">
        <div>
          <p className="text-xs uppercase tracking-widest font-bold text-[#C98769] mb-1">
            Scheduling & Roster
          </p>
          <h1 className="font-serif text-3xl font-bold text-[#234737]">
            Clinic Appointment Calendar
          </h1>
          <p className="text-xs text-[#6F7B71]">
            Comprehensive schedule of all booked, active, and completed therapy sessions.
          </p>
        </div>
      </div>

      {/* Filter Toolbar */}
      <NeuCard elevation="raised" className="p-4">
        <div className="flex flex-wrap items-center gap-3 text-xs">
          <div className="flex items-center gap-2">
            <Filter className="w-4 h-4 text-[#6F7B71]" />
            <span className="font-semibold text-[#27352D]">Filters:</span>
          </div>

          <select
            value={statusFilter}
            onChange={(e) => setStatusFilter(e.target.value)}
            className="px-3 py-1.5 rounded-xl bg-[#F5F3EC] neu-raised-sm text-xs font-medium focus:outline-none"
          >
            <option value="ALL">All Statuses</option>
            <option value="CONFIRMED">CONFIRMED</option>
            <option value="IN_PROGRESS">IN_PROGRESS</option>
            <option value="COMPLETED">COMPLETED</option>
            <option value="CANCELLED">CANCELLED</option>
          </select>

          <select
            value={doctorFilter}
            onChange={(e) => setDoctorFilter(e.target.value)}
            className="px-3 py-1.5 rounded-xl bg-[#F5F3EC] neu-raised-sm text-xs font-medium focus:outline-none"
          >
            <option value="ALL">All Consulting Doctors</option>
            {doctors.map((d) => (
              <option key={d.id} value={d.id}>
                {d.full_name}
              </option>
            ))}
          </select>

          <input
            type="date"
            value={dateFilter}
            onChange={(e) => setDateFilter(e.target.value)}
            className="px-3 py-1.5 rounded-xl bg-[#F5F3EC] neu-inset text-xs font-medium focus:outline-none"
          />

          {(statusFilter !== 'ALL' || doctorFilter !== 'ALL' || dateFilter) && (
            <button
              onClick={() => {
                setStatusFilter('ALL');
                setDoctorFilter('ALL');
                setDateFilter('');
              }}
              className="text-[#B74747] font-semibold text-xs hover:underline ml-auto"
            >
              Clear Filters
            </button>
          )}
        </div>
      </NeuCard>

      {/* Appointments Data Table */}
      <NeuCard elevation="raised" className="p-6">
        {isLoading ? (
          <NeuFeedbackStates.Skeleton count={3} />
        ) : filteredAppointments.length === 0 ? (
          <NeuFeedbackStates.EmptyState
            title="No Appointments Match Filters"
            description="Try loosening your search criteria or resetting filters to see other appointments."
            actionText="Reset Filters"
            onAction={() => {
              setStatusFilter('ALL');
              setDoctorFilter('ALL');
              setDateFilter('');
            }}
          />
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full text-left text-xs">
              <thead>
                <tr className="border-b border-[#E3DFC9] text-[#6F7B71]">
                  <th className="py-3 font-semibold">Date & Time</th>
                  <th className="py-3 font-semibold">Patient</th>
                  <th className="py-3 font-semibold">Therapy Protocol</th>
                  <th className="py-3 font-semibold">Attending Vaidya</th>
                  <th className="py-3 font-semibold">Suite</th>
                  <th className="py-3 font-semibold">Status</th>
                  <th className="py-3 font-semibold text-right">Actions</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-[#E3DFC9]/60">
                {filteredAppointments.map((appt) => {
                  const startTime = new Date(appt.start_time).toLocaleString([], {
                    month: 'short',
                    day: 'numeric',
                    hour: '2-digit',
                    minute: '2-digit',
                  });

                  return (
                    <tr key={appt.id} className="hover:bg-[#EAE7DC]/30 transition-colors">
                      <td className="py-3.5 font-bold text-[#234737]">{startTime}</td>
                      <td className="py-3.5 font-semibold text-[#27352D]">
                        {appt.patient_name || 'Patient'}
                      </td>
                      <td className="py-3.5 text-[#6F7B71]">
                        {appt.package_name || 'Abhyanga & Swedana'}
                      </td>
                      <td className="py-3.5 text-[#27352D]">
                        {appt.doctor_name || 'Dr. Anand Vaidya'}
                      </td>
                      <td className="py-3.5 text-[#6F7B71]">
                        {appt.room_name || 'Room 101'}
                      </td>
                      <td className="py-3.5">
                        <NeuBadge
                          variant={
                            appt.status === 'CONFIRMED'
                              ? 'success'
                              : appt.status === 'IN_PROGRESS'
                              ? 'forest'
                              : appt.status === 'COMPLETED'
                              ? 'sage'
                              : 'error'
                          }
                          size="sm"
                        >
                          {appt.status}
                        </NeuBadge>
                      </td>
                      <td className="py-3.5 text-right">
                        {(appt.status === 'CONFIRMED' || appt.status === 'PENDING') && (
                          <button
                            onClick={() => setSelectedAppt(appt)}
                            className="text-[#B74747] font-semibold hover:underline"
                          >
                            Cancel
                          </button>
                        )}
                      </td>
                    </tr>
                  );
                })}
              </tbody>
            </table>
          </div>
        )}
      </NeuCard>

      {/* Cancellation Modal */}
      {selectedAppt && (
        <NeuModal
          isOpen={!!selectedAppt}
          onClose={() => setSelectedAppt(null)}
          title="Administrative Cancellation"
        >
          <div className="space-y-4">
            <p className="text-xs text-[#6F7B71]">
              Cancel appointment for patient <strong className="text-[#234737]">{selectedAppt.patient_name || 'Patient'}</strong> on{' '}
              {new Date(selectedAppt.start_time).toLocaleDateString()}?
            </p>

            <div>
              <label className="text-xs font-semibold text-[#27352D] block mb-1">
                Administrative Justification:
              </label>
              <textarea
                value={cancelReason}
                onChange={(e) => setCancelReason(e.target.value)}
                placeholder="e.g. Doctor emergency leave, suite maintenance..."
                className="w-full p-3 rounded-xl bg-[#F5F3EC] neu-inset text-xs text-[#27352D] focus:outline-none"
                rows={3}
              />
            </div>

            <div className="flex justify-end gap-3 pt-4 border-t border-[#E3DFC9]">
              <NeuButton variant="raised" onClick={() => setSelectedAppt(null)}>
                Dismiss
              </NeuButton>
              <NeuButton
                variant="danger"
                isLoading={isProcessing}
                onClick={handleCancel}
              >
                Confirm Cancellation
              </NeuButton>
            </div>
          </div>
        </NeuModal>
      )}
    </div>
  );
};
