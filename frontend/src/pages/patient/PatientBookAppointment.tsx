import React, { useState, useEffect } from 'react';
import { useNavigate, useSearchParams, Link } from 'react-router-dom';
import { useAuth } from '@/context/AuthContext';
import { NeuCard } from '@/components/ui/NeuCard';
import { NeuButton } from '@/components/ui/NeuButton';
import { NeuBadge } from '@/components/ui/NeuBadge';
import { NeuFeedbackStates } from '@/components/ui/NeuFeedbackStates';
import {
  clinicApi,
  packageApi,
  doctorApi,
  appointmentApi,
} from '@/lib/api/client';
import {
  Clinic,
  TherapyPackage,
  Doctor,
  AvailableSlot,
  AppointmentHold,
  Appointment,
} from '@/lib/api/types';
import {
  CheckCircle2,
  Clock,
  Calendar,
  Building2,
  User,
  ArrowRight,
  ArrowLeft,
  Timer,
  AlertTriangle,
  Stethoscope,
  Sparkles,
  ClipboardList,
} from 'lucide-react';

export const PatientBookAppointment: React.FC = () => {
  const { user, profile } = useAuth();
  const navigate = useNavigate();
  const [searchParams] = useSearchParams();

  // Step Tracker (1 to 6)
  const [currentStep, setCurrentStep] = useState<number>(1);

  // Selections
  const [clinics, setClinics] = useState<Clinic[]>([]);
  const [selectedClinicId, setSelectedClinicId] = useState<string>(searchParams.get('clinicId') || '');
  
  const [packages, setPackages] = useState<TherapyPackage[]>([]);
  const [selectedPackageId, setSelectedPackageId] = useState<string>(searchParams.get('packageId') || '');
  
  const [doctors, setDoctors] = useState<Doctor[]>([]);
  const [selectedDoctorId, setSelectedDoctorId] = useState<string>('auto'); // 'auto' or specific docId
  
  const [selectedDate, setSelectedDate] = useState<string>(
    new Date(Date.now() + 86400000).toISOString().split('T')[0] // default tomorrow
  );
  const [availableSlots, setAvailableSlots] = useState<AvailableSlot[]>([]);
  const [selectedSlot, setSelectedSlot] = useState<AvailableSlot | null>(null);

  // Hold State
  const [hold, setHold] = useState<AppointmentHold | null>(null);
  const [secondsRemaining, setSecondsRemaining] = useState<number>(600); // 10 minutes
  const [isCreatingHold, setIsCreatingHold] = useState(false);
  const [isConfirming, setIsConfirming] = useState(false);
  const [confirmedAppointment, setConfirmedAppointment] = useState<Appointment | null>(null);

  const [isLoading, setIsLoading] = useState(true);
  const [errorMessage, setErrorMessage] = useState('');

  // 1. Load initial clinics
  useEffect(() => {
    clinicApi
      .list()
      .then((data) => {
        setClinics(data);
        if (!selectedClinicId && data.length > 0) {
          setSelectedClinicId(data[0].id);
        }
      })
      .finally(() => setIsLoading(false));
  }, []);

  // 2. Load packages & doctors when clinic changes
  useEffect(() => {
    if (!selectedClinicId) return;

    Promise.all([
      packageApi.listByClinic(selectedClinicId),
      doctorApi.listByClinic(selectedClinicId),
    ]).then(([pkgData, docData]) => {
      setPackages(pkgData);
      setDoctors(docData);
      if (!selectedPackageId && pkgData.length > 0) {
        setSelectedPackageId(pkgData[0].id);
      }
    });
  }, [selectedClinicId]);

  // 3. Load slots when reaching Step 4
  useEffect(() => {
    if (currentStep === 4 && selectedClinicId && selectedPackageId) {
      setIsLoading(true);
      setErrorMessage('');
      appointmentApi
        .searchSlots({
          clinic_id: selectedClinicId,
          package_id: selectedPackageId,
          doctor_id: selectedDoctorId === 'auto' ? undefined : selectedDoctorId,
          date: selectedDate,
        })
        .then((slots) => {
          setAvailableSlots(slots);
          if (slots.length > 0) {
            setSelectedSlot(slots[0]);
          } else {
            setSelectedSlot(null);
          }
        })
        .catch((err) => setErrorMessage(err.message || 'Error fetching slots'))
        .finally(() => setIsLoading(false));
    }
  }, [currentStep, selectedClinicId, selectedPackageId, selectedDoctorId, selectedDate]);

  // Hold Timer countdown
  useEffect(() => {
    if (currentStep !== 5 || !hold) return;

    const interval = setInterval(() => {
      setSecondsRemaining((prev) => {
        if (prev <= 1) {
          clearInterval(interval);
          setErrorMessage('Your 10-minute booking hold has expired. Please select a new slot.');
          setCurrentStep(4);
          return 0;
        }
        return prev - 1;
      });
    }, 1000);

    return () => clearInterval(interval);
  }, [currentStep, hold]);

  // Handler: Proceed to Step 5 (Create 10-minute hold)
  const handleProceedToReview = async () => {
    if (!selectedSlot || !selectedClinicId || !selectedPackageId) {
      setErrorMessage('Please select a valid time slot to proceed.');
      return;
    }

    setIsCreatingHold(true);
    setErrorMessage('');
    try {
      const newHold = await appointmentApi.createHold({
        clinic_id: selectedClinicId,
        package_id: selectedPackageId,
        doctor_id: selectedSlot.doctor_id || (selectedDoctorId !== 'auto' ? selectedDoctorId : undefined),
        room_id: selectedSlot.room_id || undefined,
        auto_assign_doctor: !selectedSlot.doctor_id && selectedDoctorId === 'auto',
        start_time: selectedSlot.start_time,
        duration_minutes: selectedSlot.duration_minutes || selectedPkg?.duration_minutes || 60,
      });
      setHold(newHold);
      setSecondsRemaining(600); // 10 minutes
      setCurrentStep(5);
    } catch (err: any) {
      setErrorMessage(err.message || 'Slot hold creation failed. The slot may have just been taken.');
    } finally {
      setIsCreatingHold(false);
    }
  };

  // Handler: Confirm Booking (Step 6)
  const handleConfirmBooking = async () => {
    if (!hold) return;

    setIsConfirming(true);
    setErrorMessage('');
    try {
      const appt = await appointmentApi.confirmHold(hold.hold_token);
      setConfirmedAppointment(appt);
      setCurrentStep(6);
    } catch (err: any) {
      setErrorMessage(err.message || 'Failed to confirm booking. Hold may have expired.');
    } finally {
      setIsConfirming(false);
    }
  };

  const selectedClinic = clinics.find((c) => c.id === selectedClinicId);
  const selectedPkg = packages.find((p) => p.id === selectedPackageId);
  const selectedDoc = doctors.find((d) => d.id === selectedDoctorId);

  const stepsList = [
    { num: 1, name: 'Clinic' },
    { num: 2, name: 'Package' },
    { num: 3, name: 'Doctor' },
    { num: 4, name: 'Time Slot' },
    { num: 5, name: 'Hold & Review' },
    { num: 6, name: 'Confirmed' },
  ];

  return (
    <div className="space-y-8">
      {/* Page Title */}
      <div>
        <p className="text-xs uppercase tracking-widest font-bold text-[#C98769] mb-1">Guided Scheduling</p>
        <h1 className="font-serif text-3xl font-bold text-[#234737]">Book Panchakarma Therapy</h1>
        <p className="text-xs text-[#6F7B71] mt-1">
          Follow the 6-step protocol with provisional hold protection to secure your therapy session.
        </p>
      </div>

      {/* 6-Step Stepper Header */}
      <NeuCard elevation="raised" className="p-4">
        <div className="grid grid-cols-6 gap-2 text-center">
          {stepsList.map((st) => {
            const isDone = currentStep > st.num;
            const isCurrent = currentStep === st.num;
            return (
              <div key={st.num} className="flex flex-col items-center">
                <div
                  className={`w-8 h-8 rounded-full flex items-center justify-center text-xs font-bold transition-all ${
                    isDone
                      ? 'bg-[#357A55] text-white shadow-sm'
                      : isCurrent
                      ? 'bg-[#315C49] text-white neu-raised-sm ring-2 ring-[#C98769]'
                      : 'bg-[#F5F3EC] text-[#6F7B71] neu-inset'
                  }`}
                >
                  {isDone ? <CheckCircle2 className="w-4 h-4" /> : st.num}
                </div>
                <span className={`text-[11px] mt-1 font-semibold hidden sm:block ${
                  isCurrent ? 'text-[#315C49]' : isDone ? 'text-[#357A55]' : 'text-[#6F7B71]'
                }`}>
                  {st.name}
                </span>
              </div>
            );
          })}
        </div>
      </NeuCard>

      {errorMessage && (
        <div className="p-4 rounded-xl bg-[#B74747]/10 border border-[#B74747]/20 flex items-center gap-3 text-xs text-[#B74747]">
          <AlertTriangle className="w-4 h-4 shrink-0" />
          <span>{errorMessage}</span>
        </div>
      )}

      {/* STEP 1: CHOOSE CLINIC */}
      {currentStep === 1 && (
        <NeuCard elevation="raised" className="p-6 space-y-6">
          <div className="flex items-center justify-between">
            <h2 className="font-serif text-xl font-bold text-[#234737]">Step 1: Select Ayurvedic Clinic</h2>
            <span className="text-xs text-[#6F7B71]">{clinics.length} centers available</span>
          </div>

          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            {clinics.map((c) => (
              <div
                key={c.id}
                onClick={() => setSelectedClinicId(c.id)}
                className={`p-5 rounded-2xl cursor-pointer transition-all ${
                  selectedClinicId === c.id
                    ? 'neu-inset border-2 border-[#315C49] bg-[#315C49]/5'
                    : 'neu-raised-sm hover:neu-pressed'
                }`}
              >
                <div className="flex items-center justify-between mb-2">
                  <span className="text-[11px] font-bold px-2 py-0.5 rounded bg-[#315C49]/10 text-[#315C49]">
                    {c.code}
                  </span>
                  <span className="text-xs text-[#6F7B71]">{c.hours || '08:00 - 18:00'}</span>
                </div>
                <h3 className="font-serif text-base font-bold text-[#234737] mb-1">{c.name}</h3>
                <p className="text-xs text-[#6F7B71] line-clamp-2">{c.address}</p>
              </div>
            ))}
          </div>

          <div className="flex justify-end pt-4 border-t border-[#E3DFC9]">
            <NeuButton
              variant="primary"
              disabled={!selectedClinicId}
              onClick={() => setCurrentStep(2)}
              className="flex items-center gap-2"
            >
              Continue to Packages <ArrowRight className="w-4 h-4" />
            </NeuButton>
          </div>
        </NeuCard>
      )}

      {/* STEP 2: CHOOSE THERAPY PACKAGE */}
      {currentStep === 2 && (
        <NeuCard elevation="raised" className="p-6 space-y-6">
          <div className="flex items-center justify-between">
            <div>
              <h2 className="font-serif text-xl font-bold text-[#234737]">Step 2: Select Therapy Package</h2>
              <p className="text-xs text-[#6F7B71]">Showing published protocols at {selectedClinic?.name}</p>
            </div>
            <NeuButton variant="raised" size="sm" onClick={() => setCurrentStep(1)}>
              Change Clinic
            </NeuButton>
          </div>

          <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
            {packages.map((pkg) => (
              <div
                key={pkg.id}
                onClick={() => setSelectedPackageId(pkg.id)}
                className={`p-5 rounded-2xl cursor-pointer transition-all flex flex-col justify-between ${
                  selectedPackageId === pkg.id
                    ? 'neu-inset border-2 border-[#315C49] bg-[#315C49]/5'
                    : 'neu-raised-sm hover:neu-pressed'
                }`}
              >
                <div>
                  <div className="flex justify-between items-center mb-2">
                    <span className="text-[11px] font-bold text-[#315C49]">{pkg.code}</span>
                    <span className="text-xs font-bold text-[#234737]">₹{pkg.price || '3,500'}</span>
                  </div>
                  <h3 className="font-serif text-base font-bold text-[#234737] mb-1">{pkg.name}</h3>
                  <p className="text-xs text-[#6F7B71] line-clamp-2 mb-3">{pkg.description}</p>
                  <div className="text-[11px] text-[#6F7B71] space-y-1">
                    <p>• {pkg.total_sessions} total sessions</p>
                    <p>• {pkg.duration_minutes} mins / session</p>
                  </div>
                </div>

                <div className="mt-4 pt-3 border-t border-[#E3DFC9] text-xs font-semibold text-[#315C49]">
                  {selectedPackageId === pkg.id ? '✓ Selected' : 'Click to select'}
                </div>
              </div>
            ))}
          </div>

          <div className="flex justify-between pt-4 border-t border-[#E3DFC9]">
            <NeuButton variant="raised" onClick={() => setCurrentStep(1)} className="flex items-center gap-2">
              <ArrowLeft className="w-4 h-4" /> Back
            </NeuButton>
            <NeuButton
              variant="primary"
              disabled={!selectedPackageId}
              onClick={() => setCurrentStep(3)}
              className="flex items-center gap-2"
            >
              Continue to Doctor Selection <ArrowRight className="w-4 h-4" />
            </NeuButton>
          </div>
        </NeuCard>
      )}

      {/* STEP 3: CHOOSE DOCTOR OR AUTO-ASSIGN */}
      {currentStep === 3 && (
        <NeuCard elevation="raised" className="p-6 space-y-6">
          <div className="flex items-center justify-between">
            <h2 className="font-serif text-xl font-bold text-[#234737]">Step 3: Select Consulting Vaidya</h2>
            <NeuBadge variant="forest">Protocol: {selectedPkg?.name}</NeuBadge>
          </div>

          <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
            {/* Auto Assign Option */}
            <div
              onClick={() => setSelectedDoctorId('auto')}
              className={`p-5 rounded-2xl cursor-pointer transition-all ${
                selectedDoctorId === 'auto'
                  ? 'neu-inset border-2 border-[#315C49] bg-[#315C49]/5'
                  : 'neu-raised-sm hover:neu-pressed'
              }`}
            >
              <div className="w-10 h-10 rounded-xl bg-[#315C49]/10 text-[#315C49] flex items-center justify-center font-bold mb-3 neu-inset">
                <Sparkles className="w-5 h-5 text-[#C98769]" />
              </div>
              <h3 className="font-serif text-base font-bold text-[#234737] mb-1">
                Auto-Assign Best Available
              </h3>
              <p className="text-xs text-[#6F7B71]">
                System assigns the first qualified Vaidya matching your requested slot and therapy room compatibility.
              </p>
              <div className="mt-4 pt-3 border-t border-[#E3DFC9] text-xs font-semibold text-[#315C49]">
                {selectedDoctorId === 'auto' ? '✓ Auto-assignment selected' : 'Select Auto'}
              </div>
            </div>

            {/* Individual Doctors */}
            {doctors.map((doc) => (
              <div
                key={doc.id}
                onClick={() => setSelectedDoctorId(doc.id)}
                className={`p-5 rounded-2xl cursor-pointer transition-all ${
                  selectedDoctorId === doc.id
                    ? 'neu-inset border-2 border-[#315C49] bg-[#315C49]/5'
                    : 'neu-raised-sm hover:neu-pressed'
                }`}
              >
                <div className="w-10 h-10 rounded-xl bg-[#315C49]/10 text-[#315C49] flex items-center justify-center font-bold mb-3 neu-inset">
                  <Stethoscope className="w-5 h-5" />
                </div>
                <h3 className="font-serif text-base font-bold text-[#234737] mb-0.5">{doc.full_name}</h3>
                <p className="text-[11px] text-[#C98769] font-medium mb-1">{doc.qualifications}</p>
                <p className="text-xs text-[#6F7B71] line-clamp-2">Specialties: {doc.specialties.join(', ')}</p>
                <div className="mt-4 pt-3 border-t border-[#E3DFC9] text-xs font-semibold text-[#315C49]">
                  {selectedDoctorId === doc.id ? '✓ Selected' : 'Select this doctor'}
                </div>
              </div>
            ))}
          </div>

          <div className="flex justify-between pt-4 border-t border-[#E3DFC9]">
            <NeuButton variant="raised" onClick={() => setCurrentStep(2)} className="flex items-center gap-2">
              <ArrowLeft className="w-4 h-4" /> Back
            </NeuButton>
            <NeuButton
              variant="primary"
              onClick={() => setCurrentStep(4)}
              className="flex items-center gap-2"
            >
              Select Available Time Slot <ArrowRight className="w-4 h-4" />
            </NeuButton>
          </div>
        </NeuCard>
      )}

      {/* STEP 4: TIME SLOT SELECTION */}
      {currentStep === 4 && (
        <NeuCard elevation="raised" className="p-6 space-y-6">
          <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4">
            <div>
              <h2 className="font-serif text-xl font-bold text-[#234737]">Step 4: Select Available Slot</h2>
              <p className="text-xs text-[#6F7B71]">
                Checking operating hours, room capacity, and staff availability.
              </p>
            </div>

            {/* Date Picker Input */}
            <div className="flex items-center gap-2">
              <Calendar className="w-4 h-4 text-[#315C49]" />
              <input
                type="date"
                value={selectedDate}
                min={new Date().toISOString().split('T')[0]}
                onChange={(e) => setSelectedDate(e.target.value)}
                className="px-3 py-2 rounded-xl bg-[#F5F3EC] neu-inset text-xs font-semibold text-[#27352D] focus:outline-none"
              />
            </div>
          </div>

          {isLoading ? (
            <NeuFeedbackStates.Skeleton count={2} />
          ) : availableSlots.length === 0 ? (
            <NeuFeedbackStates.EmptyState
              title="No Slots Available"
              description={`There are no open slots on ${selectedDate} for this therapy package. Try another date.`}
              actionText="Try Tomorrow"
              onAction={() => {
                const nextDay = new Date(new Date(selectedDate).getTime() + 86400000)
                  .toISOString()
                  .split('T')[0];
                setSelectedDate(nextDay);
              }}
            />
          ) : (
            <div className="grid grid-cols-1 sm:grid-cols-2 md:grid-cols-3 gap-4">
              {availableSlots.map((slot) => {
                const isSelected = selectedSlot?.slot_id === slot.slot_id;
                const startTimeStr = new Date(slot.start_time).toLocaleTimeString([], {
                  hour: '2-digit',
                  minute: '2-digit',
                });
                const endTimeStr = new Date(slot.end_time).toLocaleTimeString([], {
                  hour: '2-digit',
                  minute: '2-digit',
                });

                return (
                  <div
                    key={slot.slot_id}
                    onClick={() => setSelectedSlot(slot)}
                    className={`p-4 rounded-2xl cursor-pointer transition-all ${
                      isSelected
                        ? 'neu-inset border-2 border-[#315C49] bg-[#315C49]/5'
                        : 'neu-raised-sm hover:neu-pressed'
                    }`}
                  >
                    <div className="flex items-center justify-between mb-2">
                      <span className="text-xs font-bold text-[#234737] flex items-center gap-1.5">
                        <Clock className="w-3.5 h-3.5 text-[#C98769]" />
                        {startTimeStr} - {endTimeStr}
                      </span>
                      <NeuBadge variant="success" size="sm">Available</NeuBadge>
                    </div>

                    <div className="text-[11px] text-[#6F7B71] space-y-1">
                      <p>Vaidya: <span className="font-medium text-[#27352D]">{slot.doctor_name || 'Assigned Vaidya'}</span></p>
                      <p>Room: <span className="font-medium text-[#27352D]">{slot.room_name || 'Room 101'}</span></p>
                    </div>

                    <div className="mt-3 pt-2 border-t border-[#E3DFC9] text-[11px] font-semibold text-[#315C49]">
                      {isSelected ? '✓ Slot Selected' : 'Choose this slot'}
                    </div>
                  </div>
                );
              })}
            </div>
          )}

          <div className="flex justify-between pt-4 border-t border-[#E3DFC9]">
            <NeuButton variant="raised" onClick={() => setCurrentStep(3)} className="flex items-center gap-2">
              <ArrowLeft className="w-4 h-4" /> Back
            </NeuButton>
            <NeuButton
              variant="primary"
              disabled={!selectedSlot}
              isLoading={isCreatingHold}
              onClick={handleProceedToReview}
              className="flex items-center gap-2"
            >
              Lock Slot & Review (10-min Hold) <ArrowRight className="w-4 h-4" />
            </NeuButton>
          </div>
        </NeuCard>
      )}

      {/* STEP 5: 10-MINUTE HOLD & REVIEW */}
      {currentStep === 5 && hold && (
        <NeuCard elevation="raised-lg" className="p-8 space-y-6">
          {/* Active 10-Minute Hold Banner */}
          <div className="p-4 rounded-2xl bg-[#C98769]/10 border border-[#C98769]/30 flex flex-col sm:flex-row sm:items-center sm:justify-between gap-3">
            <div className="flex items-center gap-3">
              <div className="w-10 h-10 rounded-xl bg-[#C98769] text-white flex items-center justify-center font-bold">
                <Timer className="w-6 h-6 animate-pulse" />
              </div>
              <div>
                <h3 className="font-bold text-sm text-[#234737]">Provisional Slot Hold Active</h3>
                <p className="text-xs text-[#6F7B71]">
                  Hold Token: <code className="text-[#234737] font-mono">{hold.hold_token.substring(0, 16)}...</code>
                </p>
              </div>
            </div>

            <div className="text-right">
              <span className="text-xs text-[#6F7B71] block">Expires in:</span>
              <span className="font-mono text-xl font-bold text-[#B74747]">
                {Math.floor(secondsRemaining / 60)}:
                {String(secondsRemaining % 60).padStart(2, '0')}
              </span>
            </div>
          </div>

          <h2 className="font-serif text-2xl font-bold text-[#234737]">Step 5: Review Booking Details</h2>

          <div className="grid grid-cols-1 md:grid-cols-2 gap-6 bg-[#F5F3EC] p-6 rounded-2xl neu-inset">
            <div className="space-y-3 text-xs">
              <div>
                <span className="text-[#6F7B71] block">Patient Name:</span>
                <span className="font-bold text-[#27352D] text-sm">
                  {profile?.full_name || user?.email}
                </span>
              </div>
              <div>
                <span className="text-[#6F7B71] block">Clinic:</span>
                <span className="font-bold text-[#27352D]">{selectedClinic?.name}</span>
                <p className="text-[#6F7B71] text-[11px]">{selectedClinic?.address}</p>
              </div>
              <div>
                <span className="text-[#6F7B71] block">Supervising Doctor:</span>
                <span className="font-bold text-[#27352D]">
                  {selectedSlot?.doctor_name || selectedDoc?.full_name || 'Dr. Anand Vaidya'}
                </span>
              </div>
            </div>

            <div className="space-y-3 text-xs">
              <div>
                <span className="text-[#6F7B71] block">Therapy Protocol:</span>
                <span className="font-bold text-[#27352D] text-sm">{selectedPkg?.name}</span>
                <p className="text-[#6F7B71] text-[11px]">Duration: {selectedPkg?.duration_minutes} mins</p>
              </div>
              <div>
                <span className="text-[#6F7B71] block">Date & Local Time:</span>
                <span className="font-bold text-[#315C49] text-sm">
                  {new Date(hold.start_time).toLocaleDateString([], {
                    weekday: 'short',
                    month: 'short',
                    day: 'numeric',
                    year: 'numeric',
                  })}{' '}
                  at{' '}
                  {new Date(hold.start_time).toLocaleTimeString([], {
                    hour: '2-digit',
                    minute: '2-digit',
                  })}
                </span>
              </div>
              <div>
                <span className="text-[#6F7B71] block">Estimated Treatment Fee:</span>
                <span className="font-bold text-[#27352D] text-base">
                  ₹{selectedPkg?.price?.toLocaleString('en-IN') || '3,500'}
                </span>
              </div>
            </div>
          </div>

          <div className="p-4 rounded-xl bg-[#F5F3EC] neu-raised-sm text-xs text-[#6F7B71] space-y-1">
            <p className="font-semibold text-[#27352D]">Patient Preparation Notice:</p>
            <p>
              Panchakarma requires strict adherence to fasting (Laghu Ahara) and digestive rest prior to therapy.
              Please complete your Pre-Care Checklist at least 12 hours prior to arrival.
            </p>
          </div>

          <div className="flex justify-between pt-4 border-t border-[#E3DFC9]">
            <NeuButton variant="raised" onClick={() => setCurrentStep(4)}>
              Change Slot
            </NeuButton>
            <NeuButton
              variant="primary"
              size="lg"
              isLoading={isConfirming}
              onClick={handleConfirmBooking}
              className="flex items-center gap-2"
            >
              Confirm Appointment Now <CheckCircle2 className="w-5 h-5" />
            </NeuButton>
          </div>
        </NeuCard>
      )}

      {/* STEP 6: CONFIRMED STATE */}
      {currentStep === 6 && confirmedAppointment && (
        <NeuCard elevation="raised-lg" className="p-8 text-center space-y-6">
          <div className="w-20 h-20 mx-auto rounded-3xl bg-[#357A55]/15 text-[#357A55] flex items-center justify-center neu-inset">
            <CheckCircle2 className="w-10 h-10" />
          </div>

          <div className="space-y-2">
            <span className="text-xs font-bold uppercase tracking-wider text-[#357A55] px-3 py-1 rounded-full bg-[#357A55]/10">
              Booking Confirmed
            </span>
            <h2 className="font-serif text-3xl font-bold text-[#234737]">
              Appointment Successfully Scheduled!
            </h2>
            <p className="text-xs text-[#6F7B71]">
              Booking ID: <code className="font-bold text-[#234737]">{confirmedAppointment.id}</code>
            </p>
          </div>

          <div className="max-w-md mx-auto p-5 rounded-2xl bg-[#F5F3EC] neu-inset text-left text-xs space-y-2">
            <div className="flex justify-between">
              <span className="text-[#6F7B71]">Protocol:</span>
              <span className="font-bold text-[#27352D]">{confirmedAppointment.package_name || selectedPkg?.name}</span>
            </div>
            <div className="flex justify-between">
              <span className="text-[#6F7B71]">Clinic:</span>
              <span className="font-bold text-[#27352D]">{selectedClinic?.name}</span>
            </div>
            <div className="flex justify-between">
              <span className="text-[#6F7B71]">Date & Time:</span>
              <span className="font-bold text-[#315C49]">
                {new Date(confirmedAppointment.start_time).toLocaleString()}
              </span>
            </div>
            <div className="flex justify-between">
              <span className="text-[#6F7B71]">Status:</span>
              <NeuBadge variant="success" size="sm">CONFIRMED</NeuBadge>
            </div>
          </div>

          <div className="flex flex-col sm:flex-row items-center justify-center gap-4 pt-4">
            <Link to="/patient/checklists">
              <NeuButton variant="primary" size="md" className="w-full sm:w-auto flex items-center gap-2">
                <ClipboardList className="w-4 h-4" />
                Fill Pre-Care Checklist
              </NeuButton>
            </Link>
            <Link to="/patient/appointments">
              <NeuButton variant="raised" size="md" className="w-full sm:w-auto">
                View My Appointments
              </NeuButton>
            </Link>
          </div>
        </NeuCard>
      )}
    </div>
  );
};
