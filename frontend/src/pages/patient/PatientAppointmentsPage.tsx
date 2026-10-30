import React, { useEffect, useState } from 'react';
import { Link } from 'react-router-dom';
import { NeuCard } from '@/components/ui/NeuCard';
import { NeuButton } from '@/components/ui/NeuButton';
import { NeuBadge } from '@/components/ui/NeuBadge';
import { NeuModal } from '@/components/ui/NeuModal';
import { NeuFeedbackStates } from '@/components/ui/NeuFeedbackStates';
import { appointmentApi, feedbackApi } from '@/lib/api/client';
import { Appointment } from '@/lib/api/types';
import {
  Calendar,
  Clock,
  MapPin,
  Stethoscope,
  XCircle,
  RefreshCw,
  Plus,
  AlertTriangle,
  CheckCircle2,
  Star,
  MessageSquare,
  AlertOctagon,
} from 'lucide-react';

export const PatientAppointmentsPage: React.FC = () => {
  const [appointments, setAppointments] = useState<Appointment[]>([]);
  const [filter, setFilter] = useState<'ALL' | 'UPCOMING' | 'COMPLETED' | 'CANCELLED'>('ALL');
  const [isLoading, setIsLoading] = useState(true);

  // Cancellation Modal state
  const [cancelModalAppt, setCancelModalAppt] = useState<Appointment | null>(null);
  const [cancelReason, setCancelReason] = useState('');
  const [isCancelling, setIsCancelling] = useState(false);
  const [actionSuccess, setActionSuccess] = useState('');
  const [actionError, setActionError] = useState('');

  // Feedback Modal State
  const [feedbackAppt, setFeedbackAppt] = useState<Appointment | null>(null);
  const [rating, setRating] = useState<number>(5);
  const [feedbackComment, setFeedbackComment] = useState('');
  const [isAnonymous, setIsAnonymous] = useState(false);
  const [isSubmittingFeedback, setIsSubmittingFeedback] = useState(false);

  // Adverse Event Modal State
  const [adverseEventAppt, setAdverseEventAppt] = useState<Appointment | null>(null);
  const [severity, setSeverity] = useState<'MILD' | 'MODERATE' | 'SEVERE' | 'CRITICAL'>('MILD');
  const [symptoms, setSymptoms] = useState('');
  const [immediateActions, setImmediateActions] = useState('');
  const [isSubmittingAdverseEvent, setIsSubmittingAdverseEvent] = useState(false);

  const loadAppointments = () => {
    setIsLoading(true);
    appointmentApi
      .listPatientAppointments()
      .then((data) => setAppointments(data))
      .catch((err) => console.error(err))
      .finally(() => setIsLoading(false));
  };

  useEffect(() => {
    loadAppointments();
  }, []);

  const handleCancelConfirm = async () => {
    if (!cancelModalAppt) return;
    setIsCancelling(true);
    setActionError('');
    try {
      await appointmentApi.cancelAppointment(cancelModalAppt.id, cancelReason || 'Patient requested');
      setActionSuccess('Appointment cancelled successfully.');
      setCancelModalAppt(null);
      setCancelReason('');
      loadAppointments();
    } catch (err: any) {
      setActionError(err.message || 'Failed to cancel appointment');
    } finally {
      setIsCancelling(false);
    }
  };

  const handleSubmitFeedback = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!feedbackAppt) return;
    setIsSubmittingFeedback(true);
    setActionError('');
    try {
      await feedbackApi.submitFeedback({
        appointment_id: feedbackAppt.id,
        rating,
        comment: feedbackComment,
        is_anonymous: isAnonymous,
        clinic_id: feedbackAppt.clinic_id,
      });
      setActionSuccess('Thank you! Your feedback has been recorded.');
      setFeedbackAppt(null);
      setFeedbackComment('');
      setRating(5);
    } catch (err: any) {
      if (err.message && (err.message.includes('409') || err.message.includes('already exists') || err.message.includes('DUPLICATE'))) {
        setActionError('Feedback has already been submitted for this appointment (duplicate submissions are restricted).');
      } else {
        setActionError(err.message || 'Failed to submit feedback');
      }
    } finally {
      setIsSubmittingFeedback(false);
    }
  };

  const handleReportAdverseEvent = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!adverseEventAppt) return;
    setIsSubmittingAdverseEvent(true);
    setActionError('');
    try {
      await feedbackApi.reportAdverseEvent({
        appointment_id: adverseEventAppt.id,
        clinic_id: adverseEventAppt.clinic_id,
        severity,
        symptoms,
        immediate_actions: immediateActions || 'Reported by patient post-procedure',
      });
      setActionSuccess('Adverse event reported. Clinical staff have been notified for immediate review.');
      setAdverseEventAppt(null);
      setSymptoms('');
      setImmediateActions('');
    } catch (err: any) {
      setActionError(err.message || 'Failed to report adverse event');
    } finally {
      setIsSubmittingAdverseEvent(false);
    }
  };

  const filteredAppointments = appointments.filter((a) => {
    if (filter === 'UPCOMING') return a.status === 'CONFIRMED' || a.status === 'PENDING' || a.status === 'IN_PROGRESS';
    if (filter === 'COMPLETED') return a.status === 'COMPLETED';
    if (filter === 'CANCELLED') return a.status === 'CANCELLED';
    return true;
  });

  return (
    <div className="space-y-6">
      <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4">
        <div>
          <p className="text-xs uppercase tracking-widest font-bold text-[#C98769] mb-1">Schedule & History</p>
          <h1 className="font-serif text-3xl font-bold text-[#234737]">My Therapy Appointments</h1>
          <p className="text-xs text-[#6F7B71]">Review scheduled sessions, cancellations, and completed care records.</p>
        </div>

        <Link to="/patient/book">
          <NeuButton variant="primary" size="md" className="flex items-center gap-2">
            <Plus className="w-4 h-4" /> Book New Session
          </NeuButton>
        </Link>
      </div>

      {actionSuccess && (
        <div className="p-4 rounded-xl bg-[#357A55]/10 border border-[#357A55]/20 text-xs text-[#357A55] flex items-center justify-between">
          <span>{actionSuccess}</span>
          <button onClick={() => setActionSuccess('')} className="font-bold">✕</button>
        </div>
      )}

      {actionError && (
        <div className="p-4 rounded-xl bg-[#C15C3D]/10 border border-[#C15C3D]/20 text-xs text-[#C15C3D] flex items-center justify-between">
          <span>{actionError}</span>
          <button onClick={() => setActionError('')} className="font-bold">✕</button>
        </div>
      )}

      {/* Filter Tabs */}
      <div className="flex items-center gap-2 border-b border-[#E3DFC9] pb-3">
        {(['ALL', 'UPCOMING', 'COMPLETED', 'CANCELLED'] as const).map((tab) => (
          <button
            key={tab}
            onClick={() => setFilter(tab)}
            className={`px-3.5 py-1.5 rounded-xl text-xs font-semibold transition-all ${
              filter === tab
                ? 'bg-[#315C49] text-white shadow-sm'
                : 'bg-[#F5F3EC] text-[#6F7B71] hover:text-[#234737] neu-raised-sm hover:neu-pressed'
            }`}
          >
            {tab}
          </button>
        ))}
      </div>

      {/* List */}
      {isLoading ? (
        <NeuFeedbackStates.Skeleton count={3} />
      ) : filteredAppointments.length === 0 ? (
        <NeuFeedbackStates.EmptyState
          title="No Appointments Found"
          description={`No ${filter !== 'ALL' ? filter.toLowerCase() : ''} appointments found.`}
          actionText="Book New Appointment"
          onAction={() => window.location.assign('/patient/book')}
        />
      ) : (
        <div className="space-y-4">
          {filteredAppointments.map((appt) => {
            const isCancelable = appt.status === 'CONFIRMED' || appt.status === 'PENDING';
            const dateStr = new Date(appt.start_time).toLocaleDateString([], {
              weekday: 'short',
              month: 'short',
              day: 'numeric',
              year: 'numeric',
            });
            const timeStr = `${new Date(appt.start_time).toLocaleTimeString([], {
              hour: '2-digit',
              minute: '2-digit',
            })} - ${new Date(appt.end_time).toLocaleTimeString([], {
              hour: '2-digit',
              minute: '2-digit',
            })}`;

            return (
              <NeuCard key={appt.id} elevation="raised" className="p-6">
                <div className="flex flex-col md:flex-row md:items-center md:justify-between gap-4">
                  <div className="space-y-2">
                    <div className="flex items-center gap-2">
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
                      >
                        {appt.status}
                      </NeuBadge>
                      <span className="text-[11px] font-mono text-[#6F7B71]">ID: {appt.id}</span>
                    </div>

                    <h3 className="font-serif text-lg font-bold text-[#234737]">
                      {appt.package_name || 'Ayurvedic Therapy Protocol'}
                    </h3>

                    <div className="flex flex-wrap items-center gap-4 text-xs text-[#6F7B71]">
                      <div className="flex items-center gap-1.5">
                        <Calendar className="w-3.5 h-3.5 text-[#315C49]" />
                        <span className="font-semibold text-[#27352D]">{dateStr}</span>
                      </div>
                      <div className="flex items-center gap-1.5">
                        <Clock className="w-3.5 h-3.5 text-[#C98769]" />
                        <span>{timeStr}</span>
                      </div>
                      <div className="flex items-center gap-1.5">
                        <Stethoscope className="w-3.5 h-3.5 text-[#357A55]" />
                        <span>{appt.doctor_name || 'Dr. Anand Vaidya'}</span>
                      </div>
                    </div>
                  </div>

                  {/* Actions */}
                  <div className="flex flex-wrap items-center gap-2">
                    {appt.status === 'CONFIRMED' && (
                      <Link to="/patient/checklists">
                        <NeuButton variant="raised" size="sm">
                          Checklist
                        </NeuButton>
                      </Link>
                    )}
                    {appt.status === 'COMPLETED' && (
                      <>
                        <NeuButton
                          variant="raised"
                          size="sm"
                          onClick={() => {
                            setActionError('');
                            setFeedbackAppt(appt);
                          }}
                          className="flex items-center gap-1.5 text-xs text-[#315C49]"
                        >
                          <Star className="w-3.5 h-3.5 text-[#C98769]" /> Rate & Review
                        </NeuButton>
                        <NeuButton
                          variant="ghost"
                          size="sm"
                          onClick={() => {
                            setActionError('');
                            setAdverseEventAppt(appt);
                          }}
                          className="flex items-center gap-1.5 text-xs text-[#C15C3D] hover:bg-[#C15C3D]/10"
                        >
                          <AlertTriangle className="w-3.5 h-3.5" /> Report Issue
                        </NeuButton>
                      </>
                    )}
                    {isCancelable && (
                      <NeuButton
                        variant="danger"
                        size="sm"
                        onClick={() => setCancelModalAppt(appt)}
                        className="flex items-center gap-1"
                      >
                        <XCircle className="w-3.5 h-3.5" /> Cancel
                      </NeuButton>
                    )}
                  </div>
                </div>
              </NeuCard>
            );
          })}
        </div>
      )}

      {/* Cancellation Modal */}
      {cancelModalAppt && (
        <NeuModal
          isOpen={!!cancelModalAppt}
          onClose={() => setCancelModalAppt(null)}
          title="Cancel Appointment"
        >
          <div className="space-y-4">
            <p className="text-xs text-[#6F7B71]">
              Are you sure you want to cancel your appointment for{' '}
              <strong className="text-[#234737]">{cancelModalAppt.package_name}</strong> on{' '}
              {new Date(cancelModalAppt.start_time).toLocaleDateString()}?
            </p>

            <div>
              <label className="text-xs font-semibold text-[#27352D] block mb-1">
                Reason for cancellation (optional):
              </label>
              <textarea
                value={cancelReason}
                onChange={(e) => setCancelReason(e.target.value)}
                placeholder="e.g. Schedule conflict, illness..."
                className="w-full p-3 rounded-xl bg-[#F5F3EC] neu-inset text-xs text-[#27352D] focus:outline-none"
                rows={3}
              />
            </div>

            <div className="flex justify-end gap-3 pt-4 border-t border-[#E3DFC9]">
              <NeuButton variant="raised" onClick={() => setCancelModalAppt(null)}>
                Keep Appointment
              </NeuButton>
              <NeuButton
                variant="danger"
                isLoading={isCancelling}
                onClick={handleCancelConfirm}
              >
                Confirm Cancellation
              </NeuButton>
            </div>
          </div>
        </NeuModal>
      )}

      {/* Feedback Modal */}
      {feedbackAppt && (
        <NeuModal
          isOpen={!!feedbackAppt}
          onClose={() => setFeedbackAppt(null)}
          title="Therapy Experience Feedback"
        >
          <form onSubmit={handleSubmitFeedback} className="space-y-4">
            <p className="text-xs text-[#6F7B71]">
              Please rate your completed Panchakarma session for{' '}
              <strong className="text-[#234737]">{feedbackAppt.package_name}</strong>.
            </p>

            <div>
              <label className="text-xs font-semibold text-[#27352D] block mb-1">
                Overall Experience Rating (1 - 5)
              </label>
              <div className="flex items-center gap-2">
                {[1, 2, 3, 4, 5].map((s) => (
                  <button
                    type="button"
                    key={s}
                    onClick={() => setRating(s)}
                    className={`p-2 rounded-xl transition-all ${
                      rating >= s ? 'text-[#C98769] scale-110' : 'text-[#A2ADA4]'
                    }`}
                  >
                    <Star className="w-6 h-6 fill-current" />
                  </button>
                ))}
                <span className="text-xs font-bold text-[#234737] ml-2">
                  {rating} of 5 {rating === 5 ? '(Excellent)' : rating >= 4 ? '(Good)' : rating >= 3 ? '(Average)' : '(Needs Improvement)'}
                </span>
              </div>
            </div>

            <div>
              <label className="text-xs font-semibold text-[#27352D] block mb-1">
                Comments & Observations
              </label>
              <textarea
                value={feedbackComment}
                onChange={(e) => setFeedbackComment(e.target.value)}
                placeholder="Share your experience regarding therapy technique, therapist care, and post-session wellness..."
                rows={4}
                className="w-full p-3 rounded-xl bg-[#F5F3EC] neu-inset text-xs text-[#27352D] focus:outline-none"
              />
            </div>

            <div className="flex items-center gap-2 pt-1">
              <input
                type="checkbox"
                id="isAnon"
                checked={isAnonymous}
                onChange={(e) => setIsAnonymous(e.target.checked)}
                className="rounded accent-[#315C49]"
              />
              <label htmlFor="isAnon" className="text-xs text-[#6F7B71] select-none cursor-pointer">
                Submit anonymously (protect your name from clinic staff)
              </label>
            </div>

            <div className="flex justify-end gap-3 pt-4 border-t border-[#E3DFC9]">
              <NeuButton type="button" variant="raised" onClick={() => setFeedbackAppt(null)}>
                Cancel
              </NeuButton>
              <NeuButton
                type="submit"
                variant="primary"
                isLoading={isSubmittingFeedback}
              >
                Submit Feedback
              </NeuButton>
            </div>
          </form>
        </NeuModal>
      )}

      {/* Adverse Event Modal */}
      {adverseEventAppt && (
        <NeuModal
          isOpen={!!adverseEventAppt}
          onClose={() => setAdverseEventAppt(null)}
          title="Report Adverse Reaction or Concern"
        >
          <form onSubmit={handleReportAdverseEvent} className="space-y-4">
            <div className="p-3 rounded-xl bg-[#C15C3D]/10 border border-[#C15C3D]/20 text-xs text-[#C15C3D] flex items-center gap-2">
              <AlertOctagon className="w-4 h-4 shrink-0" />
              <span>
                If you are experiencing severe pain, difficulty breathing, or acute allergic reactions, seek emergency medical care immediately.
              </span>
            </div>

            <div>
              <label className="text-xs font-semibold text-[#27352D] block mb-1">
                Reaction Severity Level *
              </label>
              <select
                value={severity}
                onChange={(e) => setSeverity(e.target.value as any)}
                className="w-full p-2.5 rounded-xl bg-[#F5F3EC] neu-inset text-xs text-[#27352D] focus:outline-none font-medium"
              >
                <option value="MILD">MILD - Mild skin irritation, mild fatigue, temporary discomfort</option>
                <option value="MODERATE">MODERATE - Localized rash, dizziness, pronounced soreness</option>
                <option value="SEVERE">SEVERE - Intense dizziness, widespread swelling, fever</option>
                <option value="CRITICAL">CRITICAL - Severe allergic response, acute distress</option>
              </select>
            </div>

            <div>
              <label className="text-xs font-semibold text-[#27352D] block mb-1">
                Symptoms Description *
              </label>
              <textarea
                required
                value={symptoms}
                onChange={(e) => setSymptoms(e.target.value)}
                placeholder="Describe your symptoms in detail (e.g. onset time, area of body affected, sensation)..."
                rows={3}
                className="w-full p-3 rounded-xl bg-[#F5F3EC] neu-inset text-xs text-[#27352D] focus:outline-none"
              />
            </div>

            <div>
              <label className="text-xs font-semibold text-[#27352D] block mb-1">
                Immediate Actions Taken (optional)
              </label>
              <textarea
                value={immediateActions}
                onChange={(e) => setImmediateActions(e.target.value)}
                placeholder="e.g. Cleansed skin with water, rested, took prescribed herbal soothing tea..."
                rows={2}
                className="w-full p-3 rounded-xl bg-[#F5F3EC] neu-inset text-xs text-[#27352D] focus:outline-none"
              />
            </div>

            <div className="flex justify-end gap-3 pt-4 border-t border-[#E3DFC9]">
              <NeuButton type="button" variant="raised" onClick={() => setAdverseEventAppt(null)}>
                Cancel
              </NeuButton>
              <NeuButton
                type="submit"
                variant="danger"
                isLoading={isSubmittingAdverseEvent}
              >
                Submit Report to Clinical Staff
              </NeuButton>
            </div>
          </form>
        </NeuModal>
      )}
    </div>
  );
};
