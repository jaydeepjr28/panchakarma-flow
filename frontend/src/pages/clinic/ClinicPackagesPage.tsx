import React, { useEffect, useState } from 'react';
import { NeuCard } from '@/components/ui/NeuCard';
import { NeuButton } from '@/components/ui/NeuButton';
import { NeuBadge } from '@/components/ui/NeuBadge';
import { NeuModal } from '@/components/ui/NeuModal';
import { NeuInput } from '@/components/ui/NeuInput';
import { NeuFeedbackStates } from '@/components/ui/NeuFeedbackStates';
import { packageApi } from '@/lib/api/client';
import { TherapyPackage, PackageStep } from '@/lib/api/types';
import { useAuth } from '@/context/AuthContext';
import {
  Package,
  Plus,
  Clock,
  Calendar,
  CheckCircle2,
  FileText,
  Sparkles,
  Layers,
} from 'lucide-react';

export const ClinicPackagesPage: React.FC = () => {
  const { user } = useAuth();
  const clinicId = user?.clinic_id || 'a3f8c872-9112-4c22-b5e1-7e8913c2f101';
  const [packages, setPackages] = useState<TherapyPackage[]>([]);
  const [isLoading, setIsLoading] = useState(true);

  // New Package Modal
  const [showAddModal, setShowAddModal] = useState(false);
  const [name, setName] = useState('');
  const [code, setCode] = useState('');
  const [description, setDescription] = useState('');
  const [durationMinutes, setDurationMinutes] = useState(90);
  const [totalSessions, setTotalSessions] = useState(3);
  const [price, setPrice] = useState(4500);
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [successMsg, setSuccessMsg] = useState('');

  const loadPackages = () => {
    setIsLoading(true);
    packageApi
      .listByClinic(clinicId)
      .then((data) => setPackages(data))
      .catch((err) => console.error(err))
      .finally(() => setIsLoading(false));
  };

  useEffect(() => {
    loadPackages();
  }, [clinicId]);

  const handleCreatePackage = async (e: React.FormEvent) => {
    e.preventDefault();
    setIsSubmitting(true);
    try {
      await packageApi.createPackage({
        clinic_id: clinicId,
        name,
        code,
        description,
        duration_minutes: durationMinutes,
        total_sessions: totalSessions,
        price,
        is_published: true,
        steps: [
          { step_order: 1, name: 'Snehana (Deep Oleation)', duration_minutes: 30 },
          { step_order: 2, name: 'Swedana (Herbal Steam)', duration_minutes: 20 },
          { step_order: 3, name: 'Snana & Samshamana (Rest)', duration_minutes: 40 },
        ],
      });
      setSuccessMsg(`Package "${name}" created and published successfully.`);
      setShowAddModal(false);
      setName('');
      setCode('');
      setDescription('');
      loadPackages();
    } catch (err: any) {
      alert(err.message || 'Failed to create package');
    } finally {
      setIsSubmitting(false);
    }
  };

  return (
    <div className="space-y-6">
      <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4">
        <div>
          <p className="text-xs uppercase tracking-widest font-bold text-[#C98769] mb-1">
            Standard Operating Procedures
          </p>
          <h1 className="font-serif text-3xl font-bold text-[#234737]">
            Therapy Packages & Step Protocols
          </h1>
          <p className="text-xs text-[#6F7B71]">
            Standardized Panchakarma protocols with version snapshots, ordered steps, and pre/post checklists.
          </p>
        </div>

        <NeuButton
          variant="primary"
          size="md"
          onClick={() => setShowAddModal(true)}
          className="flex items-center gap-2"
        >
          <Plus className="w-4 h-4" /> Create Therapy Protocol
        </NeuButton>
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
        <div className="space-y-6">
          {packages.map((pkg) => (
            <NeuCard key={pkg.id} elevation="raised" className="p-6 space-y-4">
              <div className="flex flex-col sm:flex-row sm:items-start sm:justify-between gap-4 border-b border-[#E3DFC9] pb-4">
                <div className="space-y-1">
                  <div className="flex items-center gap-2">
                    <span className="text-xs font-bold px-2.5 py-0.5 rounded bg-[#315C49]/10 text-[#315C49]">
                      {pkg.code}
                    </span>
                    <NeuBadge variant={pkg.is_published ? 'success' : 'warning'}>
                      {pkg.is_published ? 'Published & Active' : 'Draft'}
                    </NeuBadge>
                    <span className="text-[11px] text-[#6F7B71]">
                      Version {pkg.version || 1} • Snapshot Immutable
                    </span>
                  </div>
                  <h2 className="font-serif text-xl font-bold text-[#234737]">{pkg.name}</h2>
                  <p className="text-xs text-[#6F7B71] max-w-2xl">{pkg.description}</p>
                </div>

                <div className="text-right sm:text-right shrink-0">
                  <span className="text-xs text-[#6F7B71] block">Standard Fee:</span>
                  <span className="font-serif text-2xl font-bold text-[#234737]">
                    ₹{pkg.price?.toLocaleString('en-IN') || '3,500'}
                  </span>
                  <span className="text-[11px] text-[#6F7B71] block">
                    {pkg.total_sessions} sessions ({pkg.duration_minutes}m each)
                  </span>
                </div>
              </div>

              {/* Steps Sequence Grid */}
              <div>
                <p className="text-xs font-bold uppercase tracking-wider text-[#315C49] mb-2 flex items-center gap-1.5">
                  <Layers className="w-3.5 h-3.5" />
                  Standardized Step Progression Timeline:
                </p>

                <div className="grid grid-cols-1 sm:grid-cols-2 md:grid-cols-3 gap-3">
                  {(pkg.steps || []).map((step) => (
                    <div
                      key={step.step_order}
                      className="p-3 rounded-xl bg-[#F5F3EC] neu-inset text-xs space-y-1"
                    >
                      <div className="flex items-center justify-between font-bold text-[#234737]">
                        <span>
                          {step.step_order}. {step.name}
                        </span>
                        <span className="text-[#C98769] font-normal">{step.duration_minutes} mins</span>
                      </div>
                      <p className="text-[11px] text-[#6F7B71]">
                        Requires: {step.required_room_type || 'Specialized Droni Suite'}
                      </p>
                    </div>
                  ))}
                </div>
              </div>

              {/* Pre/Post Guidelines */}
              <div className="pt-3 border-t border-[#E3DFC9] grid grid-cols-1 sm:grid-cols-2 gap-4 text-xs text-[#6F7B71]">
                <div>
                  <span className="font-bold text-[#27352D] block mb-1">Pre-Care Checklist:</span>
                  <p className="text-[11px]">3-hour fast, lukewarm hydration, loose cotton clothing disclosure.</p>
                </div>
                <div>
                  <span className="font-bold text-[#27352D] block mb-1">Post-Care Discharge:</span>
                  <p className="text-[11px]">{pkg.post_care_instructions || 'Rest in warm room, warm water sips, avoid direct breeze.'}</p>
                </div>
              </div>
            </NeuCard>
          ))}
        </div>
      )}

      {/* Create Package Modal */}
      {showAddModal && (
        <NeuModal
          isOpen={showAddModal}
          onClose={() => setShowAddModal(false)}
          title="Create Standardized Therapy Package"
        >
          <form onSubmit={handleCreatePackage} className="space-y-4">
            <NeuInput
              label="Package Name"
              type="text"
              required
              value={name}
              onChange={(e) => setName(e.target.value)}
              placeholder="e.g. Classical Shirodhara Protocol"
            />

            <div className="grid grid-cols-2 gap-3">
              <NeuInput
                label="Package Code"
                type="text"
                required
                value={code}
                onChange={(e) => setCode(e.target.value)}
                placeholder="e.g. SHIRO-3"
              />
              <NeuInput
                label="Standard Price (₹)"
                type="number"
                required
                value={price}
                onChange={(e) => setPrice(Number(e.target.value))}
              />
            </div>

            <div>
              <label className="text-xs font-semibold text-[#27352D] block mb-1">
                Clinical Description
              </label>
              <textarea
                value={description}
                onChange={(e) => setDescription(e.target.value)}
                rows={2}
                required
                placeholder="Indications, dosha balancing properties, and overview..."
                className="w-full p-2.5 rounded-xl bg-[#F5F3EC] neu-inset text-xs font-medium focus:outline-none"
              />
            </div>

            <div className="grid grid-cols-2 gap-3">
              <div>
                <label className="text-xs font-semibold text-[#27352D] block mb-1">
                  Duration (Minutes)
                </label>
                <input
                  type="number"
                  min={30}
                  step={15}
                  value={durationMinutes}
                  onChange={(e) => setDurationMinutes(Number(e.target.value))}
                  className="w-full p-2.5 rounded-xl bg-[#F5F3EC] neu-inset text-xs font-semibold text-[#27352D] focus:outline-none"
                />
              </div>
              <div>
                <label className="text-xs font-semibold text-[#27352D] block mb-1">
                  Total Sessions in Course
                </label>
                <input
                  type="number"
                  min={1}
                  value={totalSessions}
                  onChange={(e) => setTotalSessions(Number(e.target.value))}
                  className="w-full p-2.5 rounded-xl bg-[#F5F3EC] neu-inset text-xs font-semibold text-[#27352D] focus:outline-none"
                />
              </div>
            </div>

            <div className="p-3 rounded-xl bg-[#F5F3EC] neu-inset text-[11px] text-[#6F7B71]">
              Publishing automatically creates an immutable version 1 snapshot for subsequent bookings.
            </div>

            <div className="flex justify-end gap-3 pt-4 border-t border-[#E3DFC9]">
              <NeuButton variant="raised" onClick={() => setShowAddModal(false)}>
                Cancel
              </NeuButton>
              <NeuButton type="submit" variant="primary" isLoading={isSubmitting}>
                Publish Protocol
              </NeuButton>
            </div>
          </form>
        </NeuModal>
      )}
    </div>
  );
};
