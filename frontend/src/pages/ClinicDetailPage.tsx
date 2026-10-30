import React, { useEffect, useState } from 'react';
import { useParams, Link } from 'react-router-dom';
import { Navbar } from '@/components/layout/Navbar';
import { NeuCard } from '@/components/ui/NeuCard';
import { NeuButton } from '@/components/ui/NeuButton';
import { NeuBadge } from '@/components/ui/NeuBadge';
import { NeuFeedbackStates } from '@/components/ui/NeuFeedbackStates';
import { clinicApi, packageApi, doctorApi } from '@/lib/api/client';
import { Clinic, TherapyPackage, Doctor } from '@/lib/api/types';
import {
  MapPin,
  Star,
  Clock,
  Phone,
  Mail,
  Languages,
  Calendar,
  CheckCircle2,
  Stethoscope,
  ArrowRight,
  ShieldCheck,
  Package,
} from 'lucide-react';

export const ClinicDetailPage: React.FC = () => {
  const { clinicId } = useParams<{ clinicId: string }>();
  const [clinic, setClinic] = useState<Clinic | null>(null);
  const [packages, setPackages] = useState<TherapyPackage[]>([]);
  const [doctors, setDoctors] = useState<Doctor[]>([]);
  const [isLoading, setIsLoading] = useState(true);

  useEffect(() => {
    if (!clinicId) return;

    Promise.all([
      clinicApi.getById(clinicId),
      packageApi.listByClinic(clinicId),
      doctorApi.listByClinic(clinicId),
    ])
      .then(([cData, pData, dData]) => {
        setClinic(cData);
        setPackages(pData);
        setDoctors(dData);
      })
      .catch((err) => console.error(err))
      .finally(() => setIsLoading(false));
  }, [clinicId]);

  if (isLoading) {
    return (
      <div className="min-h-screen bg-[#F5F3EC] flex flex-col">
        <Navbar />
        <div className="max-w-7xl mx-auto px-4 py-12 w-full">
          <NeuFeedbackStates.Skeleton count={3} />
        </div>
      </div>
    );
  }

  if (!clinic) {
    return (
      <div className="min-h-screen bg-[#F5F3EC] flex flex-col">
        <Navbar />
        <div className="max-w-md mx-auto my-auto p-4">
          <NeuFeedbackStates.EmptyState
            title="Clinic Not Found"
            description="The requested clinic profile could not be located in our registry."
            actionText="Back to Clinics"
            onAction={() => window.history.back()}
          />
        </div>
      </div>
    );
  }

  return (
    <div className="min-h-screen bg-[#F5F3EC] flex flex-col text-[#27352D]">
      <Navbar />

      <main className="flex-1 max-w-7xl w-full mx-auto px-4 sm:px-6 lg:px-8 py-10 space-y-10">
        {/* Clinic Header Banner */}
        <NeuCard elevation="raised-lg" className="p-8">
          <div className="flex flex-col md:flex-row md:items-start md:justify-between gap-6">
            <div className="space-y-3">
              <div className="flex items-center gap-2">
                <span className="text-xs font-bold px-3 py-1 rounded-full bg-[#315C49]/15 text-[#315C49]">
                  {clinic.code}
                </span>
                <NeuBadge variant="success">NABH Accredited</NeuBadge>
                <div className="flex items-center gap-1 text-xs font-bold text-[#B7792B] ml-2">
                  <Star className="w-4 h-4 fill-current" />
                  <span>{clinic.rating || 4.9}</span>
                  <span className="text-[#6F7B71] font-normal">({clinic.review_count || 128} reviews)</span>
                </div>
              </div>

              <h1 className="font-serif text-3xl sm:text-4xl font-bold text-[#234737]">{clinic.name}</h1>
              <p className="text-xs sm:text-sm text-[#6F7B71] flex items-center gap-2">
                <MapPin className="w-4 h-4 text-[#C98769] shrink-0" />
                {clinic.address}
              </p>
              <p className="text-xs sm:text-sm text-[#27352D]/85 leading-relaxed max-w-3xl pt-2">
                {clinic.description}
              </p>
            </div>

            <div className="flex flex-col sm:flex-row md:flex-col gap-3 min-w-[200px]">
              <Link to={`/patient/book?clinicId=${clinic.id}`}>
                <NeuButton variant="primary" size="md" className="w-full flex items-center justify-center gap-2">
                  <Calendar className="w-4 h-4" /> Book Appointment
                </NeuButton>
              </Link>
              <div className="p-4 rounded-xl bg-[#F5F3EC] neu-inset text-xs space-y-2 text-[#6F7B71]">
                <div className="flex items-center gap-2">
                  <Clock className="w-3.5 h-3.5 text-[#315C49]" />
                  <span>{clinic.hours || '08:00 - 18:00 IST'}</span>
                </div>
                <div className="flex items-center gap-2">
                  <Phone className="w-3.5 h-3.5 text-[#C98769]" />
                  <span>{clinic.phone}</span>
                </div>
                <div className="flex items-center gap-2">
                  <Languages className="w-3.5 h-3.5 text-[#357A55]" />
                  <span>{clinic.languages?.join(', ') || 'English, Malayalam'}</span>
                </div>
              </div>
            </div>
          </div>
        </NeuCard>

        {/* Available Therapy Packages Section */}
        <section className="space-y-4">
          <div className="flex items-center justify-between">
            <div>
              <p className="text-xs uppercase tracking-widest font-bold text-[#C98769]">Clinical Protocols</p>
              <h2 className="font-serif text-2xl font-bold text-[#234737]">
                Available Therapy Packages ({packages.length})
              </h2>
            </div>
          </div>

          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
            {packages.map((pkg) => (
              <NeuCard key={pkg.id} elevation="raised" className="p-6 flex flex-col justify-between">
                <div>
                  <div className="flex items-center justify-between mb-2">
                    <span className="text-[11px] font-bold text-[#315C49] px-2 py-0.5 rounded bg-[#315C49]/10">
                      {pkg.code}
                    </span>
                    <span className="text-xs font-bold text-[#234737]">
                      ₹{pkg.price?.toLocaleString('en-IN') || '3,500'}
                    </span>
                  </div>

                  <h3 className="font-serif text-lg font-bold text-[#234737] mb-1">{pkg.name}</h3>
                  <p className="text-xs text-[#6F7B71] mb-3 line-clamp-2">{pkg.description}</p>

                  <div className="space-y-1.5 text-[11px] text-[#6F7B71] py-3 border-t border-[#E3DFC9]">
                    <div className="flex justify-between">
                      <span>Total Sessions:</span>
                      <span className="font-semibold text-[#27352D]">{pkg.total_sessions} sessions</span>
                    </div>
                    <div className="flex justify-between">
                      <span>Duration per session:</span>
                      <span className="font-semibold text-[#27352D]">{pkg.duration_minutes} mins</span>
                    </div>
                    {pkg.recommended_interval_days && (
                      <div className="flex justify-between">
                        <span>Recommended Interval:</span>
                        <span className="font-semibold text-[#27352D]">Every {pkg.recommended_interval_days} days</span>
                      </div>
                    )}
                  </div>

                  {pkg.steps && pkg.steps.length > 0 && (
                    <div className="mt-2 pt-2 border-t border-[#E3DFC9]">
                      <p className="text-[10px] uppercase font-bold text-[#6F7B71] mb-1.5">Therapy Step Sequence:</p>
                      <div className="flex flex-wrap gap-1.5">
                        {pkg.steps.map((st) => (
                          <span
                            key={st.step_order}
                            className="text-[10px] px-2 py-0.5 rounded bg-[#F5F3EC] neu-inset text-[#27352D]"
                          >
                            {st.step_order}. {st.name} ({st.duration_minutes}m)
                          </span>
                        ))}
                      </div>
                    </div>
                  )}
                </div>

                <div className="pt-4 mt-4 border-t border-[#E3DFC9]">
                  <Link to={`/patient/book?clinicId=${clinic.id}&packageId=${pkg.id}`}>
                    <NeuButton variant="primary" size="sm" className="w-full flex items-center justify-center gap-1.5">
                      Select & Book Protocol <ArrowRight className="w-3.5 h-3.5" />
                    </NeuButton>
                  </Link>
                </div>
              </NeuCard>
            ))}
          </div>
        </section>

        {/* Consulting Doctors Section */}
        <section className="space-y-4">
          <div>
            <p className="text-xs uppercase tracking-widest font-bold text-[#315C49]">Medical Faculty</p>
            <h2 className="font-serif text-2xl font-bold text-[#234737]">
              Supervising Vaidyas & Doctors ({doctors.length})
            </h2>
          </div>

          <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
            {doctors.map((doc) => (
              <NeuCard key={doc.id} elevation="raised" className="p-6 flex items-start gap-4">
                <div className="w-14 h-14 rounded-2xl bg-[#315C49]/10 text-[#315C49] flex items-center justify-center shrink-0 neu-inset">
                  <Stethoscope className="w-7 h-7" />
                </div>
                <div className="flex-1 space-y-1">
                  <div className="flex items-center justify-between">
                    <h3 className="font-serif text-base font-bold text-[#234737]">{doc.full_name}</h3>
                    <NeuBadge variant="forest" size="sm">Active</NeuBadge>
                  </div>
                  <p className="text-xs text-[#C98769] font-medium">{doc.qualifications}</p>
                  <p className="text-xs text-[#6F7B71]">Specialties: {doc.specialties.join(', ')}</p>
                  {doc.bio && <p className="text-xs text-[#27352D]/80 pt-1 line-clamp-2">{doc.bio}</p>}
                </div>
              </NeuCard>
            ))}
          </div>
        </section>
      </main>
    </div>
  );
};
